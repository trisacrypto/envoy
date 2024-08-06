package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/gravatar"

	jwt "github.com/golang-jwt/jwt/v4"
	"github.com/oklog/ulid/v2"
)

type Claims struct {
	jwt.RegisteredClaims
	ClientID     string   `json:"clientID,omitempty"`
	Name         string   `json:"name,omitempty"`
	Email        string   `json:"email,omitempty"`
	Gravatar     string   `json:"gravatar,omitempty"`
	Organization string   `json:"org,omitempty"`
	Role         string   `json:"role,omitempty"`
	Permissions  []string `json:"permissions,omitempty"`
}

type SubjectType rune

const (
	SubjectUser   = SubjectType('u')
	SubjectAPIKey = SubjectType('k')
)

var organization string

func NewClaims(ctx context.Context, model any) (*Claims, error) {
	switch t := model.(type) {
	case *models.User:
		return NewClaimsForUser(ctx, t)
	case *models.APIKey:
		return NewClaimsForAPIClient(ctx, t)
	default:
		return nil, fmt.Errorf("unknown model type %T: cannot create claims", t)
	}
}

func NewClaimsForUser(ctx context.Context, user *models.User) (claims *Claims, err error) {
	claims = &Claims{
		Name:         user.Name.String,
		Email:        user.Email,
		Gravatar:     gravatar.New(user.Email, nil),
		Organization: organization,
		Permissions:  user.Permissions(),
	}

	var role *models.Role
	if role, err = user.Role(); err != nil {
		return nil, err
	}

	claims.Role = role.Title
	claims.SetSubjectID(SubjectUser, user.ID)
	return claims, nil
}

func NewClaimsForAPIClient(ctx context.Context, key *models.APIKey) (claims *Claims, err error) {
	claims = &Claims{
		ClientID:    key.ClientID,
		Permissions: key.Permissions(),
	}

	claims.SetSubjectID(SubjectAPIKey, key.ID)
	return claims, nil
}

func (c *Claims) SetSubjectID(sub SubjectType, id ulid.ULID) {
	c.Subject = fmt.Sprintf("%c%s", sub, id)
}

func (c Claims) SubjectID() (SubjectType, ulid.ULID, error) {
	sub := SubjectType(c.Subject[0])
	id, err := ulid.Parse(c.Subject[1:])
	return sub, id, err
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
	if len(required) == 0 {
		return false
	}

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

// Set the organization for any new user claims.
func SetOrganization(o string) {
	organization = o
}

// Get the current organization that is being used for all new user claims.
func GetOrganization() string {
	return organization
}
