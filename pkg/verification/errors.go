package verification

import "errors"

var (
	ErrDecode            = errors.New("verification: could not decode token")
	ErrSize              = errors.New("verification: invalid size for token")
	ErrInvalidRecordID   = errors.New("invalid verification token: no record id")
	ErrInvalidExpiration = errors.New("invalid verification token: no expiration timestamp")
	ErrInvalidNonce      = errors.New("invalid verification token: incorrect nonce")
	ErrInvalidSignature  = errors.New("invalid verification token: incorrect hmac signature")
	ErrUnexpectedType    = errors.New("verification: could not scan non-bytes type")
)
