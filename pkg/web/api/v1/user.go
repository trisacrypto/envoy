package api

import (
	"database/sql"
	"time"
	"unicode"

	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

type User struct {
	ID        ulid.ULID  `json:"id,omitempty"`
	Name      string     `json:"name"`
	Email     string     `json:"email"`
	Passsword string     `json:"password,omitempty"`
	Role      string     `json:"role"`
	LastLogin *time.Time `json:"last_login"`
	Created   time.Time  `json:"created,omitempty"`
	Modified  time.Time  `json:"modified,omitempty"`
}

type UserList struct {
	Page  *PageQuery `json:"page"`
	Users []*User    `json:"users"`
}

type UserPassword struct {
	Password string `json:"password"`
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

	if u.Passsword != "" {
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
	model = &models.User{
		Model: models.Model{
			ID:       u.ID,
			Created:  u.Created,
			Modified: u.Modified,
		},
		Name:  sql.NullString{String: u.Name, Valid: u.Name != ""},
		Email: u.Email,
	}

	// TODO: manage the role to associate it with the user.

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

	// Password must be at least 8 characters
	if len(u.Password) < 8 {
		return ValidationError(err, IncorrectField("password", "too short: must be at least 8 characters"))
	}

	// Password must not start or end with whitespace
	if unicode.IsSpace(rune(u.Password[0])) || unicode.IsSpace(rune(u.Password[len(u.Password)-1])) {
		return ValidationError(err, IncorrectField("password", "password must not start or end with whitespace"))
	}

	// Check password strength
	var strength = []uint8{0, 0, 0, 0}
	for _, c := range u.Password {
		switch {
		case unicode.IsNumber(c):
			strength[0] = 1
		case unicode.IsUpper(c):
			strength[1] = 1
		case unicode.IsLower(c):
			strength[2] = 1
		case unicode.IsPunct(c) || unicode.IsSymbol(c):
			strength[3] = 1
		}
	}

	var strengthScore uint8
	for _, component := range strength {
		strengthScore = strengthScore + component
	}

	if strengthScore < 3 {
		err = ValidationError(err, IncorrectField("password", "password must contain uppercase letters, lowercase letters, numbers, and special characters"))
	}

	return err
}
