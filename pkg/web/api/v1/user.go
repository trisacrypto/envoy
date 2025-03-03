package api

import (
	"database/sql"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/auth/passwords"
	"go.rtnl.ai/ulid"
)

const (
	DetailUser     = "user"
	DetailPassword = "password"
)

type User struct {
	ID        ulid.ULID  `json:"id,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Password  string     `json:"password,omitempty"`
	Role      string     `json:"role"`
	LastLogin *time.Time `json:"last_login,omitempty"`
	Created   time.Time  `json:"created,omitempty"`
	Modified  time.Time  `json:"modified,omitempty"`
}

type UserList struct {
	Page  *PageQuery `json:"page"`
	Users []*User    `json:"users"`
}

type UserPassword struct {
	Password  string `json:"password"`
	SendEmail bool   `json:"send_email"`
}

type UserQuery struct {
	Detail string `json:"detail" url:"detail,omitempty" form:"detail"`
}

func NewUser(model *models.User) (out *User, err error) {
	out = &User{
		ID:       model.ID,
		Name:     model.Name.String,
		Email:    model.Email,
		Created:  model.Created,
		Modified: model.Modified,
	}

	if model.LastLogin.Valid {
		out.LastLogin = &model.LastLogin.Time
	}

	if role, err := model.Role(); err == nil {
		out.Role = role.Title
	}

	return out, nil
}

func NewUserList(page *models.UserPage) (out *UserList, err error) {
	out = &UserList{
		Page:  &PageQuery{},
		Users: make([]*User, 0, len(page.Users)),
	}

	for _, model := range page.Users {
		var user *User
		if user, err = NewUser(model); err != nil {
			return nil, err
		}
		out.Users = append(out.Users, user)
	}

	return out, nil
}

func (u *User) Validate() (err error) {
	if u.Email == "" {
		err = ValidationError(err, MissingField("email"))
	}

	if u.Password != "" {
		err = ValidationError(err, ReadOnlyField("password"))
	}

	if u.Role == "" {
		err = ValidationError(err, MissingField("role"))
	}

	if u.LastLogin != nil {
		err = ValidationError(err, ReadOnlyField("last_login"))
	}

	// NOTE: role cannot be verified without a database query
	return err
}

func (u *User) Model() (model *models.User, err error) {
	// NOTE: the role must be set by the external caller who has database access.
	model = &models.User{
		Model: models.Model{
			ID:       u.ID,
			Created:  u.Created,
			Modified: u.Modified,
		},
		Name:  sql.NullString{String: u.Name, Valid: u.Name != ""},
		Email: u.Email,
	}

	if u.LastLogin != nil {
		model.LastLogin = sql.NullTime{
			Time:  *u.LastLogin,
			Valid: true,
		}
	}

	return model, nil
}

func (u UserPassword) Validate() (err error) {
	// Password cannot be empty
	if u.Password == "" {
		return ValidationError(err, MissingField("password"))
	}

	// Validate the password strength
	if _, verr := passwords.Strength(u.Password); verr != nil {
		err = ValidationError(err, IncorrectField("password", verr.Error()))
	}

	return err
}

//===========================================================================
// User Query
//===========================================================================

func (q *UserQuery) Validate() (err error) {
	q.Detail = strings.ToLower(strings.TrimSpace(q.Detail))
	if q.Detail == "" {
		q.Detail = DetailUser
	}

	if q.Detail != DetailUser && q.Detail != DetailPassword {
		err = ValidationError(err, IncorrectField("detail", "should either be 'user' or 'password'"))
	}

	return err
}
