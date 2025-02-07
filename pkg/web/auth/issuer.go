package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/rs/zerolog/log"
	"go.rtnl.ai/ulid"
)

const (
	refreshPath            = "/v1/reauthenticate"
	DefaultRefreshAudience = "http://localhost:8000/v1/reauthenticate"
)

// Global variables that should really not be changed except between major versions.
// NOTE: the signing method should match the value returned by the JWKS
var (
	signingMethod = jwt.SigningMethodRS256
	nilID         = ulid.ULID{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
	entropy       = ulid.Monotonic(rand.Reader, 1000)
	entropyMu     sync.Mutex
)

type ClaimsIssuer struct {
	conf            config.AuthConfig
	keyID           ulid.ULID
	key             *rsa.PrivateKey
	publicKeys      map[ulid.ULID]*rsa.PublicKey
	refreshAudience string
}

func NewIssuer(conf config.AuthConfig) (_ *ClaimsIssuer, err error) {
	issuer := &ClaimsIssuer{conf: conf, publicKeys: make(map[ulid.ULID]*rsa.PublicKey, len(conf.Keys))}

	for kid, path := range conf.Keys {
		var keyID ulid.ULID
		if keyID, err = ulid.Parse(kid); err != nil {
			return nil, fmt.Errorf("could not parse %s as a key id: %w", kid, err)
		}

		var data []byte
		if data, err = os.ReadFile(path); err != nil {
			return nil, fmt.Errorf("could not read %s: %w", path, err)
		}

		var key *rsa.PrivateKey
		if key, err = jwt.ParseRSAPrivateKeyFromPEM(data); err != nil {
			return nil, fmt.Errorf("could not parse private key %s: %w", path, err)
		}

		issuer.publicKeys[keyID] = &key.PublicKey

		if issuer.key == nil || keyID.Time() > issuer.keyID.Time() {
			issuer.key = key
			issuer.keyID = keyID
		}
	}

	// If we have no keys, generate one for use (e.g. for testing or simple deployment)
	if issuer.key == nil {
		if issuer.key, err = rsa.GenerateKey(rand.Reader, 4096); err != nil {
			return nil, err
		}

		issuer.keyID = ulid.MustNew(ulid.Now(), entropy)
		issuer.publicKeys[issuer.keyID] = &issuer.key.PublicKey
		log.Warn().Str("keyID", issuer.keyID.String()).Msg("generated volatile claims issuer rsa key")
	}

	return issuer, nil
}

func (tm *ClaimsIssuer) Verify(tks string) (claims *Claims, err error) {
	var token *jwt.Token
	if token, err = jwt.ParseWithClaims(tks, &Claims{}, tm.keyFunc); err != nil {
		return nil, err
	}

	var ok bool
	if claims, ok = token.Claims.(*Claims); ok && token.Valid {
		if !claims.VerifyAudience(tm.conf.Audience, true) {
			return nil, ErrInvalidAudience
		}

		if !claims.VerifyIssuer(tm.conf.Issuer, true) {
			return nil, ErrInvalidIssuer
		}

		return claims, nil
	}

	return nil, ErrUnparsableClaims
}

// Parse an access or refresh token verifying its signature but without verifying its
// claims. This ensures that valid JWT tokens are still accepted but claims can be
// handled on a case-by-case basis; for example by validating an expired access token
// during reauthentication.
func (tm *ClaimsIssuer) Parse(tks string) (claims *Claims, err error) {
	parser := &jwt.Parser{SkipClaimsValidation: true}
	claims = &Claims{}
	if _, err = parser.ParseWithClaims(tks, claims, tm.keyFunc); err != nil {
		return nil, err
	}
	return claims, nil
}

func (tm *ClaimsIssuer) Sign(token *jwt.Token) (tks string, err error) {
	token.Header["kid"] = tm.keyID.String()
	return token.SignedString(tm.key)
}

func (tm *ClaimsIssuer) CreateAccessToken(claims *Claims) (_ *jwt.Token, err error) {
	now := time.Now()
	sub := claims.RegisteredClaims.Subject

	claims.RegisteredClaims = jwt.RegisteredClaims{
		ID:        newULID().String(),
		Subject:   sub,
		Audience:  jwt.ClaimStrings{tm.conf.Audience},
		Issuer:    tm.conf.Issuer,
		IssuedAt:  jwt.NewNumericDate(now),
		NotBefore: jwt.NewNumericDate(now),
		ExpiresAt: jwt.NewNumericDate(now.Add(tm.conf.AccessTokenTTL)),
	}

	return jwt.NewWithClaims(signingMethod, claims), nil
}

func (tm *ClaimsIssuer) CreateRefreshToken(accessToken *jwt.Token) (_ *jwt.Token, err error) {
	accessClaims, ok := accessToken.Claims.(*Claims)
	if !ok {
		return nil, ErrUnparsableClaims
	}

	// Add the refresh audience to the audience claims
	audience := append(accessClaims.Audience, tm.RefreshAudience())

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        accessClaims.ID,
			Audience:  audience,
			Issuer:    accessClaims.Issuer,
			Subject:   accessClaims.Subject,
			IssuedAt:  accessClaims.IssuedAt,
			NotBefore: jwt.NewNumericDate(accessClaims.ExpiresAt.Add(tm.conf.TokenOverlap)),
			ExpiresAt: jwt.NewNumericDate(accessClaims.IssuedAt.Add(tm.conf.RefreshTokenTTL)),
		},
	}

	return jwt.NewWithClaims(signingMethod, claims), nil
}

// CreateTokens creates and signs an access and refresh token in one step.
func (tm *ClaimsIssuer) CreateTokens(claims *Claims) (signedAccessToken, signedRefreshToken string, err error) {
	var accessToken, refreshToken *jwt.Token

	if accessToken, err = tm.CreateAccessToken(claims); err != nil {
		return "", "", fmt.Errorf("could not create access token: %w", err)
	}

	if refreshToken, err = tm.CreateRefreshToken(accessToken); err != nil {
		return "", "", fmt.Errorf("could not create refresh token: %w", err)
	}

	if signedAccessToken, err = tm.Sign(accessToken); err != nil {
		return "", "", fmt.Errorf("could not sign access token: %w", err)
	}

	if signedRefreshToken, err = tm.Sign(refreshToken); err != nil {
		return "", "", fmt.Errorf("could not sign refresh token: %w", err)
	}

	return signedAccessToken, signedRefreshToken, nil
}

// Keys returns the map of ulid to public key for use externally.
func (tm *ClaimsIssuer) Keys() map[ulid.ULID]*rsa.PublicKey {
	return tm.publicKeys
}

// CurrentKey returns the ulid of the current key being used to sign tokens.
func (tm *ClaimsIssuer) CurrentKey() ulid.ULID {
	return tm.keyID
}

func (tm *ClaimsIssuer) RefreshAudience() string {
	if tm.refreshAudience == "" {
		if tm.conf.Issuer != "" {
			if aud, err := url.Parse(tm.conf.Issuer); err == nil {
				tm.refreshAudience = aud.ResolveReference(&url.URL{Path: refreshPath}).String()
			}
		}

		if tm.refreshAudience == "" {
			tm.refreshAudience = DefaultRefreshAudience
		}
	}
	return tm.refreshAudience
}

// keyFunc is an jwt.KeyFunc that selects the RSA public key from the list of managed
// internal keys based on the kid in the token header. If the kid does not exist an
// error is returned and the token will not be able to be verified.
func (tm *ClaimsIssuer) keyFunc(token *jwt.Token) (key interface{}, err error) {
	// Per JWT security notice: do not forget to validate alg is expected
	if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
		return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
	}

	// Fetch the kid from the header
	kid, ok := token.Header["kid"]
	if !ok {
		return nil, ErrNoKeyID
	}

	// Parse the kid
	var keyID ulid.ULID
	if keyID, err = ulid.Parse(kid.(string)); err != nil {
		return nil, fmt.Errorf("could not parse kid: %w", err)
	}

	if keyID.Compare(nilID) == 0 {
		return nil, ErrInvalidKeyID
	}

	// Fetch the key from the list of managed keys
	if key, ok = tm.publicKeys[keyID]; !ok {
		return nil, ErrUnknownSigningKey
	}
	return key, nil
}

func newULID() ulid.ULID {
	entropyMu.Lock()
	defer entropyMu.Unlock()
	return ulid.MustNew(ulid.Now(), entropy)
}
