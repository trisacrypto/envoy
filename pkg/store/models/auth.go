package models

import (
	"database/sql"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/verification"
	"go.rtnl.ai/ulid"
)

type User struct {
	Model
	Name        sql.NullString
	Email       string
	Password    string
	RoleID      int64
	LastLogin   sql.NullTime
	role        *Role
	permissions []string
}

type APIKey struct {
	Model
	Description sql.NullString
	ClientID    string
	Secret      string
	LastSeen    sql.NullTime
	permissions []string
}

type Role struct {
	ID          int64
	Title       string
	Description string
	IsDefault   bool
	Created     time.Time
	Modified    time.Time
	permissions []*Permission
}

type Permission struct {
	ID          int64
	Title       string
	Description string
	Created     time.Time
	Modified    time.Time
}

type UserPageInfo struct {
	PageInfo
	Role string `json:"role,omitempty"`
}

//===========================================================================
// Associated Fields and Models
//===========================================================================

func (u User) Role() (*Role, error) {
	if u.role == nil {
		return nil, errors.ErrMissingAssociation
	}
	return u.role, nil
}

func (u *User) SetRole(role *Role) {
	u.role = role
	u.RoleID = role.ID
}

func (u User) Permissions() []string {
	return u.permissions
}

func (u *User) SetPermissions(permissions []string) {
	u.permissions = permissions
}

func (k APIKey) Permissions() []string {
	return k.permissions
}

func (k *APIKey) SetPermissions(permissions []string) {
	k.permissions = permissions
}

func (r Role) Permissions() ([]*Permission, error) {
	if r.permissions == nil {
		return nil, errors.ErrMissingAssociation
	}
	return r.permissions, nil
}

func (r *Role) SetPermissions(permissions []*Permission) {
	r.permissions = permissions
}

//===========================================================================
// Scan and Params
//===========================================================================

func (u *User) Scan(scanner Scanner) error {
	return scanner.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.Password,
		&u.RoleID,
		&u.LastLogin,
		&u.Created,
		&u.Modified,
	)
}

func (u *User) ScanSummary(scanner Scanner) error {
	return scanner.Scan(
		&u.ID,
		&u.Name,
		&u.Email,
		&u.RoleID,
		&u.LastLogin,
		&u.Created,
		&u.Modified,
	)
}

func (u *User) Params() []any {
	return []any{
		sql.Named("id", u.ID),
		sql.Named("name", u.Name),
		sql.Named("email", u.Email),
		sql.Named("password", u.Password),
		sql.Named("roleID", u.RoleID),
		sql.Named("lastLogin", u.LastLogin),
		sql.Named("created", u.Created),
		sql.Named("modified", u.Modified),
	}
}

func (k *APIKey) Scan(scanner Scanner) error {
	return scanner.Scan(
		&k.ID,
		&k.Description,
		&k.ClientID,
		&k.Secret,
		&k.LastSeen,
		&k.Created,
		&k.Modified,
	)
}

func (k *APIKey) ScanSummary(scanner Scanner) error {
	return scanner.Scan(
		&k.ID,
		&k.Description,
		&k.ClientID,
		&k.LastSeen,
		&k.Created,
		&k.Modified,
	)
}

func (k *APIKey) Params() []any {
	return []any{
		sql.Named("id", k.ID),
		sql.Named("description", k.Description),
		sql.Named("clientID", k.ClientID),
		sql.Named("secret", k.Secret),
		sql.Named("lastSeen", k.LastSeen),
		sql.Named("created", k.Created),
		sql.Named("modified", k.Modified),
	}
}

func (r *Role) Scan(scanner Scanner) error {
	return scanner.Scan(
		&r.ID,
		&r.Title,
		&r.Description,
		&r.IsDefault,
		&r.Created,
		&r.Modified,
	)
}

func (r *Role) Params() []any {
	return []any{
		sql.Named("id", r.ID),
		sql.Named("title", r.Title),
		sql.Named("description", r.Description),
		sql.Named("isDefault", r.IsDefault),
		sql.Named("created", r.Created),
		sql.Named("modified", r.Modified),
	}
}

func (p *Permission) Scan(scanner Scanner) error {
	return scanner.Scan(
		&p.ID,
		&p.Title,
		&p.Description,
		&p.Created,
		&p.Modified,
	)
}

func (p *Permission) Params() []any {
	return []any{
		sql.Named("id", p.ID),
		sql.Named("title", p.Title),
		sql.Named("description", p.Description),
		sql.Named("created", p.Created),
		sql.Named("modified", p.Modified),
	}
}

//===========================================================================
// ResetPasswordLink
//===========================================================================

type ResetPasswordLink struct {
	Model
	UserID     ulid.ULID                 // A foreign key reference to the user's account
	Email      string                    // Email address of recipient the token was sent to
	Expiration time.Time                 // The timestamp that the sunrise verification token is no longer valid
	Signature  *verification.SignedToken // The signed token produced by the sunrise package for verification purposes
	SentOn     sql.NullTime              // The timestamp that the email message was sent
	VerifiedOn sql.NullTime              // The timestamp that the user verified the token
}

// Scans a complete SELECT into the ResetPasswordLink model
func (s *ResetPasswordLink) Scan(scanner Scanner) error {
	return scanner.Scan(
		&s.ID,
		&s.UserID,
		&s.Email,
		&s.Expiration,
		&s.Signature,
		&s.SentOn,
		&s.VerifiedOn,
		&s.Created,
		&s.Modified,
	)
}

// Get the complete named params of the ResetPasswordLink message from the model.
func (s *ResetPasswordLink) Params() []any {
	return []any{
		sql.Named("id", s.ID),
		sql.Named("userID", s.UserID),
		sql.Named("email", s.Email),
		sql.Named("expiration", s.Expiration),
		sql.Named("signature", s.Signature),
		sql.Named("sentOn", s.SentOn),
		sql.Named("verifiedOn", s.VerifiedOn),
		sql.Named("created", s.Created),
		sql.Named("modified", s.Modified),
	}
}

// IsExpired returns true if the link expiration is before the current time.
func (s *ResetPasswordLink) IsExpired() bool {
	return time.Now().After(s.Expiration)
}
