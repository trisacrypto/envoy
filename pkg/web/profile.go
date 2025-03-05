package web

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/auth/passwords"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"go.rtnl.ai/ulid"
)

func (s *Server) ProfileDetail(c *gin.Context) {
	var (
		err  error
		user *models.User
		out  *api.User
	)

	// Retreive the user from the request context
	if user, err = s.retrieveProfile(c); err != nil {
		switch {
		case errors.Is(err, auth.ErrNotAuthorized):
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.Error(err))
		case errors.Is(err, dberr.ErrNotFound):
			c.JSON(http.StatusNotFound, api.Error("profile not found"))
		default:
			c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, api.Error("unable to retrieve profile"))
		}
		return
	}

	if out, err = api.NewUser(user); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to retrieve profile"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/profile/form.html",
	})
}

func (s *Server) UpdateProfile(c *gin.Context) {
	var (
		err  error
		user *models.User
		in   *api.User
		out  *api.User
	)

	in = &api.User{}
	if err = c.BindJSON(in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error("could not parse user profile"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Retreive the user from the request context
	if user, err = s.retrieveProfile(c); err != nil {
		switch {
		case errors.Is(err, auth.ErrNotAuthorized):
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.Error(err))
		case errors.Is(err, dberr.ErrNotFound):
			c.JSON(http.StatusNotFound, api.Error("profile not found"))
		default:
			c.Error(err)
			c.AbortWithStatusJSON(http.StatusInternalServerError, api.Error("unable to retrieve profile"))
		}
		return
	}

	// The only thing the user can update with this form is their name and email address.
	user.Name = sql.NullString{String: in.Name, Valid: in.Name != ""}
	user.Email = in.Email

	if err = s.store.UpdateUser(c.Request.Context(), user); err != nil {
		// TODO: handle email unique constraint violation.
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to update user profile"))
		return
	}

	if out, err = api.NewUser(user); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to fetch specified user"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/profile/form.html",
	})
}

func (s *Server) DeleteProfile(c *gin.Context) {
	c.JSON(http.StatusNotImplemented, api.Error("not implemented"))
}

func (s *Server) ChangeProfilePassword(c *gin.Context) {
	negotiate := gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "partials/profile/changePassword.html",
	}

	var (
		err        error
		in         *api.ProfilePassword
		user       *models.User
		derivedKey string
	)

	in = &api.ProfilePassword{}
	if err = c.BindJSON(in); err != nil {
		negotiate.JSONData = api.Error("could not parse password change request")
		negotiate.HTMLData = gin.H{"Error": "could not parse password change request"}
		c.Negotiate(http.StatusBadRequest, negotiate)
		return
	}

	if err = in.Validate(); err != nil {
		negotiate.JSONData = api.Error(err)
		if verr, ok := err.(api.ValidationErrors); ok {
			negotiate.HTMLData = gin.H{"FieldErrors": verr.Map()}
		} else {
			negotiate.HTMLData = gin.H{"Error": err.Error()}
		}

		c.Negotiate(http.StatusUnprocessableEntity, negotiate)
		return
	}

	// Retreive the user from the request context
	if user, err = s.retrieveProfile(c); err != nil {
		// By default in change password we'll return 400 to display the error alert.
		// Only if something is really bad we will redirect to error page.
		statusCode := http.StatusBadRequest

		switch {
		case errors.Is(err, auth.ErrNotAuthorized) || errors.Is(err, dberr.ErrNotFound):
			negotiate.JSONData = api.Error("could not change password")
			negotiate.HTMLData = gin.H{"Error": "could not change password"}
		default:
			c.Error(err)
			statusCode = http.StatusInternalServerError
			negotiate.JSONData = api.Error("could not complete change password request")
			negotiate.HTMLData = gin.H{"Error": "could not change password"}
		}

		c.Negotiate(statusCode, negotiate)
		return
	}

	// Confirm the current password is correct
	if verified, err := passwords.VerifyDerivedKey(user.Password, in.Current); err != nil || !verified {
		negotiate.JSONData = api.Error("current password is incorrect")
		negotiate.HTMLData = gin.H{"FieldErrors": map[string]string{"current": "password is incorrect"}}
		c.Negotiate(http.StatusBadRequest, negotiate)
		return
	}

	// Create derived key from requested password reset
	if derivedKey, err = passwords.CreateDerivedKey(in.Password); err != nil {
		c.Error(err)
		negotiate.JSONData = api.Error("could not complete change password request")
		c.Negotiate(http.StatusInternalServerError, negotiate)
		return
	}

	// Set the password for the specified user
	if err = s.store.SetUserPassword(c.Request.Context(), user.ID, derivedKey); err != nil {
		c.Error(err)
		negotiate.JSONData = api.Error("could not complete change password request")
		c.Negotiate(http.StatusInternalServerError, negotiate)
		return
	}

	// Success! Log the user out and redirect to the login page.
	auth.ClearAuthCookies(c, s.conf.Web.Auth.CookieDomain)

	// Send the user to the login page
	if c.NegotiateFormat(binding.MIMEHTML, binding.MIMEJSON) == binding.MIMEHTML {
		htmx.Redirect(c, http.StatusFound, "/login")
		return
	}

	c.JSON(http.StatusOK, &api.Reply{Success: true})
}

func (s *Server) retrieveUserID(c *gin.Context) (userID ulid.ULID, err error) {
	var (
		claims  *auth.Claims
		subject auth.SubjectType
	)

	if claims, err = auth.GetClaims(c); err != nil {
		return ulid.Null, auth.ErrNotAuthorized
	}

	if subject, userID, err = claims.SubjectID(); err != nil {
		return ulid.Null, fmt.Errorf("could not parse subject ID from claims: %w", err)
	}

	if subject != auth.SubjectUser {
		return ulid.Null, dberr.ErrNotFound
	}

	return userID, nil
}

// Helper method to retrieve the profile being managed by the currently logged in user.
func (s *Server) retrieveProfile(c *gin.Context) (user *models.User, err error) {
	var userID ulid.ULID
	if userID, err = s.retrieveUserID(c); err != nil {
		return nil, err
	}

	// Fetch the model from the database
	if user, err = s.store.RetrieveUser(c.Request.Context(), userID); err != nil {
		return nil, err
	}

	return user, nil
}
