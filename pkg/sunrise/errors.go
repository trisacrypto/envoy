package sunrise

import "errors"

var (
	ErrDecode            = errors.New("sunrise: could not decode token")
	ErrSize              = errors.New("sunrise: invalid size for token")
	ErrInvalidSunriseID  = errors.New("invalid sunrise token: no sunrise id")
	ErrInvalidExpiration = errors.New("invalid sunrise token: no expiration timestamp")
	ErrInvalidNonce      = errors.New("invalid sunrise token: incorrect nonce")
	ErrInvalidSignature  = errors.New("invalid sunrise token: incorrect hmac signature")
	ErrUnexpectedType    = errors.New("sunrise: could not scan non-bytes type")
)
