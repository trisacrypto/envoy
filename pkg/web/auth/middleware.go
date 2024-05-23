package auth

import (
	"net/http"
	"net/url"
	"regexp"
	"time"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/htmx"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/rs/zerolog/log"
)

const (
	Authorization      = "Authorization"
	AccessTokenCookie  = "access_token"
	RefreshTokenCookie = "refresh_token"
	ContextUserClaims  = "user_claims"
	CookieMaxAgeBuffer = 600 * time.Second
)

// used to extract the access token from the authorization header
var bearer = regexp.MustCompile(`^\s*[Bb]earer\s+([a-zA-Z0-9_\-\.]+)\s*$`)

func Authenticate(issuer *ClaimsIssuer) gin.HandlerFunc {
	innerAuthenticate := func(c *gin.Context) (claims *Claims, err error) {
		// Fetch access token from the request, if no access token is available, reject.
		var accessToken string
		if accessToken, err = GetAccessToken(c); err != nil {
			log.Debug().Err(err).Msg("no access token in authenticated request")
			return nil, ErrAuthRequired
		}

		if claims, err = issuer.Verify(accessToken); err != nil {
			// TODO: attempt to refresh the claims if the accessToken is expired.
			log.Debug().Err(err).Msg("invalid access token in request")
			return nil, ErrAuthRequired
		}

		return claims, nil
	}

	return func(c *gin.Context) {
		var (
			err    error
			claims *Claims
		)

		if claims, err = innerAuthenticate(c); err != nil {
			// Content Negotiation
			switch c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) {
			case binding.MIMEJSON:
				// Return a 401 with the error to the API client
				c.AbortWithStatusJSON(http.StatusUnauthorized, api.Error(err))
				return
			case binding.MIMEHTML:
				// Redirect the user to the login page
				params := make(url.Values)
				params.Set("next", c.Request.URL.Path)
				redirect := &url.URL{Path: "/login", RawQuery: params.Encode()}

				c.Abort()
				htmx.Redirect(c, http.StatusFound, redirect.String())
				return
			default:
				c.AbortWithError(http.StatusNotAcceptable, ErrNotAccepted)
				return
			}
		}

		// Add claims to context fo ruse in downstream processing
		c.Set(ContextUserClaims, claims)
		c.Next()
	}
}

func Authorize(permissions ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, err := GetClaims(c)
		if err != nil {
			log.Warn().Err(err).Msg("no claims in request")
			c.AbortWithStatusJSON(http.StatusUnauthorized, api.Error(ErrNotAuthorized))
			return
		}

		if !claims.HasAllPermissions(permissions...) {
			log.Debug().Err(err).Msg("user does not have required permissions")
			c.AbortWithStatusJSON(http.StatusForbidden, api.Error(ErrNotAuthorized))
			return
		}

		c.Next()
	}
}

// GetAccessToken retrieves the bearer token from the authorization header and parses it
// to return only the JWT access token component of the header. Alternatively, if the
// authorization header is not present, then the token is fetched from cookies. If the
// header is missing or the token is not available, an error is returned.
//
// NOTE: the authorization header takes precedence over access tokens in cookies.
func GetAccessToken(c *gin.Context) (tks string, err error) {
	// Attempt to get the access token from the header.
	if header := c.GetHeader(Authorization); header != "" {
		match := bearer.FindStringSubmatch(header)
		if len(match) == 2 {
			return match[1], nil
		}
		return "", ErrParseBearer
	}

	// Attempt to get the access token from cookies.
	var cookie string
	if cookie, err = c.Cookie(AccessTokenCookie); err == nil {
		// If the error is nil, that means we were able to retrieve the access token cookie
		return cookie, nil
	}

	// If we could find the access token, return an error.
	return "", ErrNoAuthorization
}

// GetRefreshToken retrieves the refresh token from the cookies in the request. If the
// cookie is not present or expired then an error is returned.
func GetRefreshToken(c *gin.Context) (tks string, err error) {
	if tks, err = c.Cookie(RefreshTokenCookie); err != nil {
		return "", ErrNoRefreshToken
	}
	return tks, nil
}

func GetClaims(c *gin.Context) (*Claims, error) {
	claims, exists := c.Get(ContextUserClaims)
	if !exists {
		return nil, ErrNoClaims
	}
	return claims.(*Claims), nil
}

// SetAuthCookies is a helper function to set authentication cookies on a gin request.
// The access token cookie (access_token) is an http only cookie that expires when the
// access token expires. The refresh token cookie is not an http only cookie (it can be
// accessed by client-side scripts) and it expires when the refresh token expires. Both
// cookies require https and will not be set (silently) over http connections.
func SetAuthCookies(c *gin.Context, accessToken, refreshToken, domain string) (err error) {
	// Parse access token to get expiration time
	var accessExpires time.Time
	if accessExpires, err = ExpiresAt(accessToken); err != nil {
		return err
	}

	// Secure is true unless the domain is localhost
	secure := true
	if domain == "localhost" {
		secure = false
	}

	// Set the access token cookie: httpOnly is true; cannot be accessed by Javascript
	accessMaxAge := int((time.Until(accessExpires.Add(CookieMaxAgeBuffer))).Seconds())
	c.SetCookie(AccessTokenCookie, accessToken, accessMaxAge, "/", domain, secure, true)

	// Parse refresh token to get expiration time
	var refreshExpires time.Time
	if refreshExpires, err = ExpiresAt(refreshToken); err != nil {
		return err
	}

	// Set the refresh token cookie: httpOnly is false; can be accessed by Javascript
	refreshMaxAge := int((time.Until(refreshExpires.Add(CookieMaxAgeBuffer))).Seconds())
	c.SetCookie(RefreshTokenCookie, refreshToken, refreshMaxAge, "/", domain, secure, false)
	return nil
}

// ClearAuthCookies is a helper function to clear authentication cookies on a gin
// request to effectively log out a user.
func ClearAuthCookies(c *gin.Context, domain string) {
	c.SetCookie(AccessTokenCookie, "", -1, "/", domain, true, true)
	c.SetCookie(RefreshTokenCookie, "", -1, "/", domain, true, false)
}
