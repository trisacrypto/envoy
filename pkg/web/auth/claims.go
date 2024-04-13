package auth

import (
	"context"
	"errors"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/oklog/ulid/v2"
)

type Claims struct {
	jwt.RegisteredClaims
	ClientID    string   `json:"clientID,omitempty"`
	Email       string   `json:"email,omitempty"`
	Gravatar    string   `json:"gravatar,omitempty"`
	Role        string   `json:"role,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
}

func NewClaims(ctx context.Context) (*Claims, error) {
	// TODO: identify user type or apikey type and use appropriate method
	return NewClaimsForUser(ctx)
}

func NewClaimsForUser(ctx context.Context) (claims *Claims, err error) {
	return nil, errors.New("not implemented yet")
}

func NewClaimsForAPIClient(ctx context.Context) (claims *Claims, err error) {
	return nil, errors.New("not implemented yet")
}

func (c *Claims) SetSubjectID(uid ulid.ULID) {
	c.Subject = uid.String()
}

func (c Claims) SubjectID() (ulid.ULID, error) {
	return ulid.Parse(c.Subject)
}

func (c Claims) HasPermission(required string) bool {
	for _, permisison := range c.Permissions {
		if permisison == required {
			return true
		}
	}
	return false
}

func (c Claims) HasAllPermissions(required ...string) bool {
	for _, perm := range required {
		if !c.HasPermission(perm) {
			return false
		}
	}
	return true
}

// Used to extract expiration and not before timestamps without having to use public keys
var tsparser = &jwt.Parser{SkipClaimsValidation: true}

func ParseUnverified(tks string) (claims *jwt.RegisteredClaims, err error) {
	claims = &jwt.RegisteredClaims{}
	if _, _, err = tsparser.ParseUnverified(tks, claims); err != nil {
		return nil, err
	}
	return claims, nil
}

func ExpiresAt(tks string) (_ time.Time, err error) {
	var claims *jwt.RegisteredClaims
	if claims, err = ParseUnverified(tks); err != nil {
		return time.Time{}, err
	}
	return claims.ExpiresAt.Time, nil
}

func NotBefore(tks string) (_ time.Time, err error) {
	var claims *jwt.RegisteredClaims
	if claims, err = ParseUnverified(tks); err != nil {
		return time.Time{}, err
	}
	return claims.NotBefore.Time, nil
}
