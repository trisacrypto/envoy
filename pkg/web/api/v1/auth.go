package api

import "strings"

//===========================================================================
// Authentication Resources
//===========================================================================

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Next     string `json:"next"`
}

type APIAuthentication struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

type LoginReply struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
}

type ReauthenticateRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ResetPasswordRequest struct {
	Email string `json:"email"`
}

//TODO: reset password verification request

func (r *LoginRequest) Validate() (err error) {
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" {
		err = ValidationError(err, MissingField("email"))
	}

	r.Password = strings.TrimSpace(r.Password)
	if r.Password == "" {
		err = ValidationError(err, MissingField("password"))
	}

	return err
}

func (r *APIAuthentication) Validate() (err error) {
	r.ClientID = strings.TrimSpace(r.ClientID)
	if r.ClientID == "" {
		err = ValidationError(err, MissingField("client_id"))
	}

	r.ClientSecret = strings.TrimSpace(r.ClientSecret)
	if r.ClientSecret == "" {
		err = ValidationError(err, MissingField("client_secret"))
	}

	return err
}

func (r *ReauthenticateRequest) Validate() (err error) {
	r.RefreshToken = strings.TrimSpace(r.RefreshToken)
	if r.RefreshToken == "" {
		err = ValidationError(err, MissingField("refresh_token"))
	}
	return err
}

//TODO: reset password verification code (see sunrise.go in this folder)
