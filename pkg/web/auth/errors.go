package auth

import "errors"

var (
	ErrUnknownSigningKey = errors.New("unknown signing key")
	ErrNoKeyID           = errors.New("token does not have kid in header")
	ErrInvalidKeyID      = errors.New("invalid key id")
	ErrUnparsableClaims  = errors.New("could not parse or verify claims")
	ErrInvalidAudience   = errors.New("invalid audience")
	ErrInvalidIssuer     = errors.New("invalid issuer")
	ErrUnauthenticated   = errors.New("request is unauthenticated")
	ErrNoClaims          = errors.New("no claims found on the request context")
	ErrNoUserInfo        = errors.New("no user info found on the request context")
	ErrInvalidAuthToken  = errors.New("invalid authorization token")
	ErrAuthRequired      = errors.New("this endpoint requires authentication")
	ErrNotAuthorized     = errors.New("user does not have permission to perform this operation")
	ErrNoAuthUser        = errors.New("could not identify authenticated user in request")
	ErrParseBearer       = errors.New("could not parse Bearer token from Authorization header")
	ErrNoAuthorization   = errors.New("no authorization header in request")
	ErrNoRefreshToken    = errors.New("cannot reauthenticate no refresh token in request")
)
