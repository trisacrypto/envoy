package web

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/logger"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/verification"
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
			htmx.Redirect(c, http.StatusSeeOther, in.Next)
			return
		}
		htmx.Redirect(c, http.StatusSeeOther, "/")
	default:
		c.AbortWithError(http.StatusNotAcceptable, ErrNotAccepted)
	}
}

func (s *Server) Logout(c *gin.Context) {
	// Clear the client cookies
	auth.ClearAuthCookies(c, s.conf.Web.Auth.CookieDomain)

	// Send the user to the login page
	htmx.Redirect(c, http.StatusSeeOther, "/login")
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
		htmx.Redirect(c, http.StatusSeeOther, "/")
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

//////////////////////////////////////////////////////////////////////////////
// Reset Password Workflow Endpoints
//////////////////////////////////////////////////////////////////////////////

// Looks up a user by email and sends that user a link/token to reset their password.
func (s *Server) ForgotPassword(c *gin.Context) {
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
		s.Error(c, errors.New("could not parse reset password request"))
		return
	}

	// Send the email, also creating a verification token; if no email was provided
	// simply redirect them to the success page to avoid leaking information.
	if in.Email != "" {
		ctx := c.Request.Context()
		if err = s.sendResetPasswordEmail(ctx, in.Email); err != nil {
			// If the user is not found, then still redirect to the success page because
			// we don't want to leak information about whether the email address is valid.
			// If the error is ErrTooSoon, then we want to rate limit the user without
			// leaking information so also redirect to the success page.
			if !errors.Is(err, dberr.ErrNotFound) && !errors.Is(err, dberr.ErrTooSoon) {
				c.Error(err)
				s.Error(c, errors.New("could not complete reset password request"))
				return
			}

			log.Warn().Err(err).Str("email", in.Email).Msg("non-user email address provided for reset password request")
		}
	}

	// Make sure the user is logged out to prevent session hijacking
	auth.ClearAuthCookies(c, s.conf.Web.Auth.CookieDomain)

	// Redirect to reset-password success page (note do not use an HTMX partial here
	// because the forgot password request can come from a logged in user on their
	// profile page or a non-logged in user on the login page); a full redirect is
	// necessary so they can close this window and follow the flow from their email.
	htmx.Redirect(c, http.StatusSeeOther, "/forgot-password/sent")
}

// Verifies an incoming password change requested via a verification link, then changes
// the user's password according to the password form submitted.
func (s *Server) ResetPassword(c *gin.Context) {
	var (
		derivedKey string
		err        error
		in         *api.ResetPasswordChangeRequest
		token      verification.VerificationToken
		link       *models.ResetPasswordLink
	)
	const (
		failureStatus   = http.StatusSeeOther
		failureTemplate = "auth/reset/failure.html"
	)

	// Prepare general
	in = &api.ResetPasswordChangeRequest{}
	ctx := c.Request.Context()
	log := logger.Tracing(ctx)

	// Prepare for possible error
	errScene := scene.New(c)
	errScene["SupportEmail"] = s.conf.Email.SupportEmail

	// We do not allow JSON API requests to this endpoint. Returning a 406 error
	// here is for the legitimate API users.
	if IsAPIRequest(c) {
		c.AbortWithStatusJSON(http.StatusNotAcceptable, api.Error("endpoint unavailable for API calls"))
		return
	}

	// Read the token string
	if err = c.BindJSON(in); err != nil {
		log.Warn().Err(err).Msg("could not parse query string")
		c.HTML(failureStatus, failureTemplate, errScene)
		return
	}

	// Validate the token string
	if err = in.URLVerification.Validate(); err != nil {
		// If the token is invalid or missing, return a 404.
		// NOTE: do not log an error as this is very verbose, instead just a debug message
		log.Debug().Err(err).Msg("reset password change with invalid token")
		c.HTML(failureStatus, failureTemplate, errScene)
		return
	}

	// Get the verification token from the token string
	token = in.URLVerification.VerificationToken()

	// Verify the ResetPasswordLink token
	if link, err = s.verifyResetPasswordLinkToken(c, token); err != nil {
		// NOTE: `verifyResetPasswordLinkToken()` handles setting the response and logging the error
		return
	}

	// Confirm the two entered passwords are valid and match
	password := api.ProfilePassword{
		Current:  "ignored",
		Password: in.Password,
		Confirm:  in.Confirm,
	}
	if err = password.Validate(); err != nil {
		log.Warn().Err(err).Str("link_id", link.ID.String()).Msg("reset password change invalid ProfilePassword")
		c.HTML(failureStatus, failureTemplate, errScene)
		return
	}

	// Create derived key from requested password reset
	if derivedKey, err = passwords.CreateDerivedKey(in.Password); err != nil {
		log.Warn().Err(err).Str("link_id", link.ID.String()).Msg("reset password change could not create derived key")
		c.HTML(failureStatus, failureTemplate, errScene)
		return
	}

	// Set the password for the specified user
	if err = s.store.SetUserPassword(c.Request.Context(), link.UserID, derivedKey); err != nil {
		log.Warn().Err(err).Str("link_id", link.ID.String()).Msg("reset password change could not set password for user")
		c.HTML(failureStatus, failureTemplate, errScene)
		return
	}

	// Redirect to reset-password success page
	c.HTML(http.StatusOK, "auth/reset/success_changed.html", scene.New(c))
}

//////////////////////////////////////////////////////////////////////////////
// Reset Password Workflow internal functions
//////////////////////////////////////////////////////////////////////////////

// The default amount of time that a ResetPasswordLink will expire after
const resetPasswordLinkTTL = 15 * time.Minute

// Send a reset password email to the user, also creating a verification token.
func (s *Server) sendResetPasswordEmail(ctx context.Context, emailAddr string) (err error) {
	// Lookup the user
	// TODO: add this all in a single transaction so we don't have a partial
	// reset password link and no email sent.
	var user *models.User
	if user, err = s.store.RetrieveUser(ctx, emailAddr); err != nil {
		return err
	}

	// Create a ResetPasswordLink record for database storage
	record := &models.ResetPasswordLink{
		UserID:     user.ID,
		Email:      user.Email,
		Expiration: time.Now().Add(resetPasswordLinkTTL),
	}

	// Create the ID in the database of the ResetPasswordLink record.
	// NOTE: the create reset password link database function will return ErrTooSoon
	// if the record already exists and is not expired; otherwise it will delete any
	// existing (expired) record for the user and create a new one. ErrTooSoon will
	// enable rate limiting to make sure the user cannot spam reset password requests.
	if err = s.store.CreateResetPasswordLink(ctx, record); err != nil {
		return err
	}

	// Create the ResetPasswordEmailData for the email builder
	emailData := emails.ResetPasswordEmailData{
		ContactName:  user.Name.String,
		BaseURL:      s.url.ResolveReference(&url.URL{Path: "/reset-password"}),
		SupportEmail: s.conf.Email.SupportEmail,
	}

	// Create the HMAC verification token for the ResetPasswordLink
	verification := verification.NewToken(record.ID, record.Expiration)

	// Sign the verification token
	if emailData.Token, record.Signature, err = verification.Sign(); err != nil {
		return err
	}

	// Update the ResetPasswordLink record in the database with the token
	if err = s.store.UpdateResetPasswordLink(ctx, record); err != nil {
		return err
	}

	// Build the email
	var email *emails.Email
	if email, err = emails.NewResetPasswordEmail(user.Email, emailData); err != nil {
		return err
	}

	// Send the email to the user
	if err = email.Send(); err != nil {
		return err
	}

	// Update the ResetPasswordLink record in the database with a SentOn timestamp
	record.SentOn = sql.NullTime{Valid: true, Time: time.Now()}
	if err = s.store.UpdateResetPasswordLink(ctx, record); err != nil {
		return err
	}

	return nil
}

// Verifies a ResetPasswordLink token and returns the ResetPasswordLink object.
// NOTE: This function will handle logging and setting the HTML response for errors.
func (s *Server) verifyResetPasswordLinkToken(c *gin.Context, token verification.VerificationToken) (link *models.ResetPasswordLink, err error) {
	// Prepare for possible error
	const (
		failureStatus   = http.StatusSeeOther
		failureTemplate = "auth/reset/failure.html"
	)
	errScene := scene.New(c)
	errScene["SupportEmail"] = s.conf.Email.SupportEmail

	// Prepare context and logging
	ctx := c.Request.Context()
	log := logger.Tracing(ctx)

	// Get the ResetPasswordLink record from the database
	if link, err = s.store.RetrieveResetPasswordLink(ctx, token.RecordID()); err != nil {
		log.Warn().Err(err).Str("token_record_id", token.RecordID().String()).Msg("could not retrieve reset password record")
		c.HTML(failureStatus, failureTemplate, errScene)
		return nil, err
	}

	// Check that the token is valid
	if secure, err := link.Signature.Verify(token); err != nil || !secure {
		// If the token is not secure or verifiable, return a 404 but be freaked out
		log.Warn().Err(err).Str("link_id", link.ID.String()).Bool("secure", secure).Msg("a reset password request hmac verification failed")
		c.HTML(failureStatus, failureTemplate, errScene)
		return nil, err
	}

	// Check that the token and link have both not expired
	if link.Signature.Token.IsExpired() || link.IsExpired() {
		// The token is expired or has already been verified/completed.
		log.Debug().Str("link_id", link.ID.String()).Msg("received a request with an expired verification token")
		c.HTML(failureStatus, failureTemplate, errScene)
		return nil, err
	}

	// Delete the ResetPasswordLink record from the database now that record is verified.
	if err = s.store.DeleteResetPasswordLink(ctx, link.ID); err != nil {
		log.Warn().Err(err).Str("link_id", link.ID.String()).Msg("failed to update reset password verified time")
		c.HTML(failureStatus, failureTemplate, errScene)
		return nil, err
	}

	return link, nil
}
