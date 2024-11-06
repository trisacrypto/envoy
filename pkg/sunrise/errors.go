package sunrise

import "errors"

var (
	ErrDecode            = errors.New("sunrise: could not decode token")
	ErrSize              = errors.New("sunrise: invalid size for token")
	ErrInvalidEnvelopeID = errors.New("invalid sunrise token: no envelope id")
	ErrInvalidExpiration = errors.New("invalid sunrise token: no expiration timestamp")
	ErrInvalidNonce      = errors.New("invalid sunrise token: incorrect nonce")
)
