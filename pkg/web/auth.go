package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/trisacrypto/envoy/pkg/logger"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/auth/passwords"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"github.com/trisacrypto/envoy/pkg/web/scene"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/ulid"
)

func (s *Server) Login(c *gin.Context) {
	var (
		err    error
		user   *models.User
		in     *api.LoginRequest
		claims *auth.Claims
		out    *api.LoginReply
		ctx    context.Context
	)

	if err = c.BindJSON(&in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse login request"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Retrieve the user by email to validate password
	ctx = c.Request.Context()
	if user, err = s.store.RetrieveUser(ctx, in.Email); err != nil {
		// If user is not found, return a 403
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusForbidden, api.Error("invalid login credentials"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process login request"))
		return
	}

	// Check that the password supplied in the request is correct
	if verified, err := passwords.VerifyDerivedKey(user.Password, in.Password); err != nil || !verified {
		log := logger.Tracing(ctx)
		log.Debug().Err(err).Msg("invalid login credentials")

		c.JSON(http.StatusForbidden, api.Error("invalid login credentials"))
		return
	}

	// Update user last login timestamp
	user.LastLogin = sql.NullTime{Valid: true, Time: time.Now()}
	if err = s.store.SetUserLastLogin(ctx, user.ID, user.LastLogin.Time); err != nil {
		log := logger.Tracing(ctx)
		log.Warn().Err(err).Msg("unable to update user last login timestamp")

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process login request"))
		return
	}

	// Create access and refresh tokens for authentication
	if claims, err = auth.NewClaims(ctx, user); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process login request"))
		return
	}

	out = &api.LoginReply{}
	if out.AccessToken, out.RefreshToken, err = s.issuer.CreateTokens(claims); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process login request"))
		return
	}

	// Set the tokens as cookies for the front-end
	if err = auth.SetAuthCookies(c, out.AccessToken, out.RefreshToken, s.conf.Web.Auth.CookieDomain); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process login request"))
		return
	}

	// Content negotiation and redirect if html
	switch c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) {
	case binding.MIMEJSON:
		c.JSON(http.StatusOK, out)
	case binding.MIMEHTML:
		if in.Next != "" {
			htmx.Redirect(c, http.StatusFound, in.Next)
			return
		}
		htmx.Redirect(c, http.StatusFound, "/")
	default:
		c.AbortWithError(http.StatusNotAcceptable, ErrNotAccepted)
	}
}

func (s *Server) Logout(c *gin.Context) {
	// Clear the client cookies
	auth.ClearAuthCookies(c, s.conf.Web.Auth.CookieDomain)

	// Send the user to the login page
	htmx.Redirect(c, http.StatusFound, "/login")
}

func (s *Server) Authenticate(c *gin.Context) {
	var (
		err    error
		ctx    context.Context
		apikey *models.APIKey
		in     *api.APIAuthentication
		out    *api.LoginReply
		claims *auth.Claims
	)

	if err = c.BindJSON(&in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse authenticate request"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Retrieve the apikey by clientID to validate the client secret
	ctx = c.Request.Context()
	if apikey, err = s.store.RetrieveAPIKey(ctx, in.ClientID); err != nil {
		// If the API Key is not found, return a 403
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusForbidden, api.Error("invalid api credentials"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process authenticate request"))
		return
	}

	// Verify the client secret is correct
	if verified, err := passwords.VerifyDerivedKey(apikey.Secret, in.ClientSecret); err != nil || !verified {
		log := logger.Tracing(ctx)
		log.Debug().Err(err).Msg("invalid api key credentials")

		c.JSON(http.StatusForbidden, api.Error("invalid api credentials"))
		return
	}

	// Update api key last seen timestamp
	apikey.LastSeen = sql.NullTime{Valid: true, Time: time.Now()}
	if err = s.store.UpdateAPIKey(ctx, apikey); err != nil {
		log := logger.Tracing(ctx)
		log.Warn().Err(err).Msg("unable to update api key last seen timestamp")

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process authenticate request"))
		return
	}

	// Create access and refresh tokens and return them to the user
	if claims, err = auth.NewClaims(ctx, apikey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process authenticate request"))
		return
	}

	out = &api.LoginReply{}
	if out.AccessToken, out.RefreshToken, err = s.issuer.CreateTokens(claims); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process authenticate request"))
		return
	}

	// Set the tokens as cookies in case the api client has a cookie jar
	if err = auth.SetAuthCookies(c, out.AccessToken, out.RefreshToken, s.conf.Web.Auth.CookieDomain); err != nil {
		log := logger.Tracing(c.Request.Context())
		log.Warn().Err(err).Msg("could not set cookies on api authenticate")
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) Reauthenticate(c *gin.Context) {
	var (
		err    error
		ctx    context.Context
		claims *auth.Claims
		sub    auth.SubjectType
		subID  ulid.ULID
		in     *api.ReauthenticateRequest
		out    *api.LoginReply
	)

	if err = c.BindJSON(&in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("unable to parse reauthenticate request"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Verify the refresh token
	if claims, err = s.issuer.Verify(in.RefreshToken); err != nil {
		c.Error(err)
		c.JSON(http.StatusForbidden, api.Error("invalid reauthentication credentials"))
		return
	}

	// Ensure the token is a refresh token
	ctx = c.Request.Context()
	if !claims.VerifyAudience(s.issuer.RefreshAudience(), true) {
		log := logger.Tracing(ctx)
		log.Warn().Msg("valid refresh token does not contain refresh audience")

		c.JSON(http.StatusForbidden, api.Error("invalid reauthentication credentials"))
		return
	}

	// Identify if the subject is a user or an api key
	if sub, subID, err = claims.SubjectID(); err != nil {
		log := logger.Tracing(ctx)
		log.Warn().Err(err).Msg("could not parse subject id from claims")

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process reauthenticate request"))
		return
	}

	// Load new claims from database (don't simply reuse old claims)
	// Update last seen or last login timestamp
	switch sub {
	case auth.SubjectUser:
		if claims, err = s.reauthenticateUser(c, subID); err != nil {
			// Error logging and response is handled in method
			return
		}
	case auth.SubjectAPIKey:
		if claims, err = s.reauthenticateAPIKey(c, subID); err != nil {
			// Error logging and response is handled in method
			return
		}
	default:
		c.Error(fmt.Errorf("unknown subject type %c", sub))
		c.JSON(http.StatusForbidden, api.Error("invalid reauthentication credentials"))
		return
	}

	// Create new access token/refresh token pair
	out = &api.LoginReply{}
	if out.AccessToken, out.RefreshToken, err = s.issuer.CreateTokens(claims); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process reauthenticate request"))
		return
	}

	// Set the tokens as cookies for the front-end/api cookie jar
	if err = auth.SetAuthCookies(c, out.AccessToken, out.RefreshToken, s.conf.Web.Auth.CookieDomain); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process reauthenticate request"))
		return
	}

	// Content negotiation and redirect if html
	switch c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) {
	case binding.MIMEJSON:
		c.JSON(http.StatusOK, out)
	case binding.MIMEHTML:
		htmx.Redirect(c, http.StatusFound, "/")
	default:
		c.AbortWithError(http.StatusNotAcceptable, ErrNotAccepted)
	}
}

func (s *Server) reauthenticateUser(c *gin.Context, userID ulid.ULID) (_ *auth.Claims, err error) {
	ctx := c.Request.Context()

	var user *models.User
	if user, err = s.store.RetrieveUser(ctx, userID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusForbidden, api.Error("invalid reauthentication credentials"))
			return nil, err
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process reauthenticate request"))
		return nil, err
	}

	user.LastLogin = sql.NullTime{Valid: true, Time: time.Now()}
	if err = s.store.SetUserLastLogin(ctx, user.ID, user.LastLogin.Time); err != nil {
		log := logger.Tracing(ctx)
		log.Warn().Err(err).Msg("unable to update user last login timestamp")

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process reauthenticate request"))
		return
	}

	return auth.NewClaims(ctx, user)
}

func (s *Server) reauthenticateAPIKey(c *gin.Context, keyID ulid.ULID) (_ *auth.Claims, err error) {
	ctx := c.Request.Context()

	var apikey *models.APIKey
	if apikey, err = s.store.RetrieveAPIKey(c.Request.Context(), keyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusForbidden, api.Error("invalid reauthentication credentials"))
			return nil, err
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process reauthenticate request"))
		return nil, err
	}

	apikey.LastSeen = sql.NullTime{Valid: true, Time: time.Now()}
	if err = s.store.UpdateAPIKey(ctx, apikey); err != nil {
		log := logger.Tracing(ctx)
		log.Warn().Err(err).Msg("unable to update api key last seen timestamp")

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process reauthenticate request"))
		return
	}

	return auth.NewClaims(ctx, apikey)
}

func (s *Server) ChangePassword(c *gin.Context) {
	var (
		err         error
		claims      *auth.Claims
		in          *api.UserPassword
		userID      ulid.ULID
		subjectType auth.SubjectType
		derivedKey  string
	)

	// Get the claims of the currently authenticated user to change the password for.
	if claims, err = auth.GetClaims(c); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not initiate change user password request"))
		return
	}

	// Get the user ID from the subject of the claims
	if subjectType, userID, err = claims.SubjectID(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not initiate change user password request"))
		return
	}

	// Validate the subject type
	if subjectType != auth.SubjectUser {
		c.Error(fmt.Errorf("cannot change password for subject type %d", subjectType))
		c.JSON(http.StatusBadRequest, api.Error("could not initiate change user password request"))
		return
	}

	// Parse the user data for the update request
	in = &api.UserPassword{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse password change request"))
		return
	}

	// Validation
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create derived key from requested password reset
	if derivedKey, err = passwords.CreateDerivedKey(in.Password); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete change user password request"))
		return
	}

	// Set the password for the specified user
	if err = s.store.SetUserPassword(c.Request.Context(), userID, derivedKey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete change user password request"))
		return
	}

	// Create template scene for rendering information about the user
	data := scene.New(c)
	data["Success"] = true
	data["UserID"] = userID

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLData: data,
		JSONData: api.Reply{Success: true},
		HTMLName: "password_changed.html",
	})
}

func (s *Server) ResetPassword(c *gin.Context) {
	var (
		err error
		in  *api.ResetPasswordRequest
	)

	// We do not allow JSON API requests to this endpoint.
	// Technically someone could automate requests with an Accept: text/html header
	// so it's also important to rate limit reset password requests. But returning a
	// 406 error here is for the legitimate API users.
	if IsAPIRequest(c) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, api.Error("endpoint unavailable for API calls"))
		return
	}

	in = &api.ResetPasswordRequest{}
	if err = c.BindJSON(in); err != nil {
		c.JSON(http.StatusBadRequest, api.Error("could not parse reset password request"))
		return
	}

	// TODO: lookup user and send reset email with rate limiting.

	// Make sure the user is logged out to prevent session hijacking
	auth.ClearAuthCookies(c, s.conf.Web.Auth.CookieDomain)

	// Redirect to reset-password success page
	htmx.Redirect(c, http.StatusFound, "/reset-password/success")
}
