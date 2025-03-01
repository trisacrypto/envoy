package auth_test

import (
	"crypto/rand"
	"crypto/rsa"
	"strings"
	"testing"
	"time"

	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/web/auth"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/suite"
	"go.rtnl.ai/ulid"
)

type TokenTestSuite struct {
	suite.Suite
	testdata map[string]string
}

func (s *TokenTestSuite) SetupSuite() {
	// Create the keys map from the testdata directory to create new token managers.
	s.testdata = make(map[string]string)
	s.testdata["01GE6191AQTGMCJ9BN0QC3CCVG"] = "testdata/01GE6191AQTGMCJ9BN0QC3CCVG.pem"
	s.testdata["01GE62EXXR0X0561XD53RDFBQJ"] = "testdata/01GE62EXXR0X0561XD53RDFBQJ.pem"
}

func (s *TokenTestSuite) TestTokenManager() {
	require := s.Require()
	conf := config.AuthConfig{
		Keys:            s.testdata,
		Audience:        "http://localhost:3000",
		Issuer:          "http://localhost:3001",
		CookieDomain:    "localhost",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -15 * time.Minute,
	}

	tm, err := auth.NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	keys := tm.Keys()
	require.Len(keys, 2)
	require.Equal("01GE62EXXR0X0561XD53RDFBQJ", tm.CurrentKey().String())

	// Create an access token from simple claims
	creds := &auth.Claims{
		Email: "kate@example.com",
		Name:  "Kate Holland",
	}

	accessToken, err := tm.CreateAccessToken(creds)
	require.NoError(err, "could not create access token from claims")
	require.IsType(&auth.Claims{}, accessToken.Claims)

	time.Sleep(500 * time.Millisecond)
	now := time.Now()

	// Check access token claims
	ac := accessToken.Claims.(*auth.Claims)
	require.NotZero(ac.ID)
	require.Equal(jwt.ClaimStrings{"http://localhost:3000"}, ac.Audience)
	require.Equal("http://localhost:3001", ac.Issuer)
	require.True(ac.IssuedAt.Before(now))
	require.True(ac.NotBefore.Before(now))
	require.True(ac.ExpiresAt.After(now))
	require.Equal(creds.Email, ac.Email)
	require.Equal(creds.Name, ac.Name)

	// Create a refresh token from the access token
	refreshToken, err := tm.CreateRefreshToken(accessToken)
	require.NoError(err, "could not create refresh token from access token")
	require.IsType(&auth.Claims{}, refreshToken.Claims)

	// Check refresh token claims
	// Check access token claims
	rc := refreshToken.Claims.(*auth.Claims)
	require.Equal(ac.ID, rc.ID, "access and refresh tokens must have same jid")
	require.NotEqual(ac.Audience, rc.Audience, "expected refresh token to have refresh audience")
	require.Equal(jwt.ClaimStrings{"http://localhost:3000", "http://localhost:3001/v1/reauthenticate"}, rc.Audience)
	require.Equal(ac.Issuer, rc.Issuer)
	require.Equal(ac.Subject, rc.Subject)
	require.True(rc.IssuedAt.Equal(ac.IssuedAt.Time))
	require.True(rc.NotBefore.After(now))
	require.True(rc.ExpiresAt.After(rc.NotBefore.Time))
	require.Empty(rc.Email)

	// Verify relative nbf and exp claims of access and refresh tokens
	require.True(ac.IssuedAt.Equal(rc.IssuedAt.Time), "access and refresh tokens do not have same iss timestamp")
	require.Equal(45*time.Minute, rc.NotBefore.Sub(ac.IssuedAt.Time), "refresh token nbf is not 45 minutes after access token iss")
	require.Equal(15*time.Minute, ac.ExpiresAt.Sub(rc.NotBefore.Time), "refresh token active does not overlap active token active by 15 minutes")
	require.Equal(60*time.Minute, rc.ExpiresAt.Sub(ac.ExpiresAt.Time), "refresh token does not expire 1 hour after access token")

	// Sign the access token
	atks, err := tm.Sign(accessToken)
	require.NoError(err, "could not sign access token")

	// Sign the refresh token
	rtks, err := tm.Sign(refreshToken)
	require.NoError(err, "could not sign refresh token")
	require.NotEqual(atks, rtks, "identical access and refresh tokens")

	// Validate the access token
	_, err = tm.Verify(atks)
	require.NoError(err, "could not validate access token")

	// Validate the refresh token (should be invalid because of not before in the future)
	_, err = tm.Verify(rtks)
	require.Error(err, "refresh token is valid?")
}

func (s *TokenTestSuite) TestValidTokens() {
	require := s.Require()
	conf := config.AuthConfig{
		Keys:            s.testdata,
		Audience:        "http://localhost:3000",
		Issuer:          "http://localhost:3001",
		CookieDomain:    "localhost",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -15 * time.Minute,
	}

	tm, err := auth.NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	// Default creds
	creds := &auth.Claims{
		Email: "kate@example.com",
		Name:  "Kate Holland",
	}

	// TODO: add validation steps and test
	_, err = tm.CreateAccessToken(creds)
	require.NoError(err)
}

func (s *TokenTestSuite) TestInvalidTokens() {
	// Create the token manager
	require := s.Require()
	conf := config.AuthConfig{
		Keys:            s.testdata,
		Audience:        "http://localhost:3000",
		Issuer:          "http://localhost:3001",
		CookieDomain:    "localhost",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -15 * time.Minute,
	}

	tm, err := auth.NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	// Manually create a token to validate with the token manager
	now := time.Now()
	claims := &auth.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        ulid.Make().String(),                           // id not validated
			Audience:  jwt.ClaimStrings{"http://foo.example.com"},     // wrong audience
			IssuedAt:  jwt.NewNumericDate(now.Add(-1 * time.Hour)),    // iat not validated
			NotBefore: jwt.NewNumericDate(now.Add(15 * time.Minute)),  // nbf is validated and is after now
			ExpiresAt: jwt.NewNumericDate(now.Add(-30 * time.Minute)), // exp is validated and is before now
		},
		Email: "kate@example.com",
		Name:  "Kate Holland",
	}

	// Test validation signed with wrong kid
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "01GE63H600NKHE7B8Y7MHW1VGV"
	badkey, err := rsa.GenerateKey(rand.Reader, 1024)
	require.NoError(err, "could not generate bad rsa keys")
	tks, err := token.SignedString(badkey)
	require.NoError(err, "could not sign token with bad kid")

	_, err = tm.Verify(tks)
	require.EqualError(err, "unknown signing key")

	// Test validation signed with good kid but wrong key
	token.Header["kid"] = "01GE62EXXR0X0561XD53RDFBQJ"
	tks, err = token.SignedString(badkey)
	require.NoError(err, "could not sign token with bad keys and good kid")

	_, err = tm.Verify(tks)
	require.EqualError(err, "crypto/rsa: verification error")

	// Test time-based validation: nbf
	tks, err = tm.Sign(token)
	require.NoError(err, "could not sign token with good keys")

	_, err = tm.Verify(tks)
	require.EqualError(err, "token is not valid yet")

	// Test time-based validation: exp
	claims.NotBefore = jwt.NewNumericDate(now.Add(-1 * time.Hour))
	tks, err = tm.Sign(jwt.NewWithClaims(jwt.SigningMethodRS256, claims))
	require.NoError(err, "could not sign token with good keys")

	// NOTE: actual error message is "token is expired by 30m0s" however, if the clock
	// minute happens to tick over the message could be "token is expired by 30m1s" so
	// to prevent test failure, we're only testing the prefix.
	_, err = tm.Verify(tks)
	require.True(strings.HasPrefix(err.Error(), "token is expired"))

	// Test audience verification
	claims.ExpiresAt = jwt.NewNumericDate(now.Add(1 * time.Hour))
	tks, err = tm.Sign(jwt.NewWithClaims(jwt.SigningMethodRS256, claims))
	require.NoError(err, "could not sign token with good keys")

	_, err = tm.Verify(tks)
	require.ErrorIs(err, auth.ErrInvalidAudience)

	// Token is finally valid
	claims.Audience = jwt.ClaimStrings{"http://localhost:3000"}
	claims.Issuer = "http://localhost:3001"
	tks, err = tm.Sign(jwt.NewWithClaims(jwt.SigningMethodRS256, claims))
	require.NoError(err, "could not sign token with good keys")
	_, err = tm.Verify(tks)
	require.NoError(err, "claims are still not valid")
}

// Test that a token signed with an old cert can still be verified.
// This also tests that the correct signing key is required.
func (s *TokenTestSuite) TestKeyRotation() {
	require := s.Require()

	// Create the "old token manager"
	testdata := make(map[string]string)
	testdata["01GE6191AQTGMCJ9BN0QC3CCVG"] = "testdata/01GE6191AQTGMCJ9BN0QC3CCVG.pem"

	conf := config.AuthConfig{
		Keys:            testdata,
		Audience:        "http://localhost:3000",
		Issuer:          "http://localhost:3001",
		CookieDomain:    "localhost",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -15 * time.Minute,
	}

	oldTM, err := auth.NewIssuer(conf)
	require.NoError(err, "could not initialize old token manager")

	// Create the "new" token manager with the new key
	conf2 := config.AuthConfig{
		Keys:            s.testdata,
		Audience:        "http://localhost:3000",
		Issuer:          "http://localhost:3001",
		CookieDomain:    "localhost",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -15 * time.Minute,
	}

	newTM, err := auth.NewIssuer(conf2)
	require.NoError(err, "could not initialize new token manager")

	// Create a valid token with the "old token manager"
	token, err := oldTM.CreateAccessToken(&auth.Claims{
		Email: "kate@example.com",
		Name:  "Kate Holland",
	})
	require.NoError(err)

	tks, err := oldTM.Sign(token)
	require.NoError(err)

	// Validate token with "new token manager"
	_, err = newTM.Verify(tks)
	require.NoError(err)

	// A token created by the "new token mangaer" should not be verified by the old one.
	tks, err = newTM.Sign(token)
	require.NoError(err)

	_, err = oldTM.Verify(tks)
	require.Error(err)
}

// Test that a token can be parsed even if it is expired. This is necessary to parse
// access tokens in order to use a refresh token to extract the claims.
func (s *TokenTestSuite) TestParseExpiredToken() {
	require := s.Require()
	conf := config.AuthConfig{
		Keys:            s.testdata,
		Audience:        "http://localhost:3000",
		Issuer:          "http://localhost:3001",
		CookieDomain:    "localhost",
		AccessTokenTTL:  1 * time.Hour,
		RefreshTokenTTL: 2 * time.Hour,
		TokenOverlap:    -15 * time.Minute,
	}

	tm, err := auth.NewIssuer(conf)
	require.NoError(err, "could not initialize token manager")

	// Default creds
	creds := &auth.Claims{
		Email: "kate@example.com",
		Name:  "Kate Holland",
	}

	accessToken, err := tm.CreateAccessToken(creds)
	require.NoError(err, "could not create access token from claims")
	require.IsType(&auth.Claims{}, accessToken.Claims)

	// Modify claims to be expired
	claims := accessToken.Claims.(*auth.Claims)
	claims.IssuedAt = jwt.NewNumericDate(claims.IssuedAt.Add(-24 * time.Hour))
	claims.ExpiresAt = jwt.NewNumericDate(claims.ExpiresAt.Add(-24 * time.Hour))
	claims.NotBefore = jwt.NewNumericDate(claims.NotBefore.Add(-24 * time.Hour))
	accessToken.Claims = claims

	// Create signed token
	tks, err := tm.Sign(accessToken)
	require.NoError(err, "could not create expired access token from claims")

	// Ensure that verification fails; claims are invalid.
	pclaims, err := tm.Verify(tks)
	require.Error(err, "expired token was somehow validated?")
	require.Empty(pclaims, "verify returned claims even after error")

	// Parse token without verifying claims but verifying the signature
	pclaims, err = tm.Parse(tks)
	require.NoError(err, "claims were validated in parse")
	require.NotEmpty(pclaims, "parsing returned empty claims without error")

	// Check claims
	require.Equal(claims.ID, pclaims.ID)
	require.Equal(claims.ExpiresAt, pclaims.ExpiresAt)
	require.Equal(creds.Email, claims.Email)
	require.Equal(creds.Name, claims.Name)

	// Ensure signature is still validated on parse
	tks += "abcdefg"
	claims, err = tm.Parse(tks)
	require.Error(err, "claims were parsed with bad signature")
	require.Empty(claims, "bad signature token returned non-empty claims")
}

func (s *TokenTestSuite) TestRefreshAudience() {
	require := s.Require()

	tests := []struct {
		issuer   string
		expected string
	}{
		{
			"https://trisa.example.com",
			"https://trisa.example.com/v1/reauthenticate",
		},
		{
			"",
			auth.DefaultRefreshAudience,
		},
	}

	for i, tc := range tests {
		conf := config.AuthConfig{
			Audience: tc.issuer,
			Issuer:   tc.issuer,
		}

		tm, err := auth.NewIssuer(conf)
		require.NoError(err, "could not initialize token manager")

		audience := tm.RefreshAudience()
		require.Equal(tc.expected, audience, "test case %d failed", i)
	}
}

// Execute suite as a go test.
func TestTokenTestSuite(t *testing.T) {
	suite.Run(t, new(TokenTestSuite))
}
