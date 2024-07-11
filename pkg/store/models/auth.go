package models

import (
	"database/sql"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/errors"
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
