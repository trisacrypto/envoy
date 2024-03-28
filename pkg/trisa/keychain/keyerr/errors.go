package keyerr

import "fmt"

const (
	KeyNotFound         Error = "key with specified signature not found"
	KeyExpired          Error = "key with specified signature has expired"
	KeyNotMatched       Error = "key for specified common name not found"
	KeyOverwrite        Error = "cannot overwrite key"
	KeysNotComparable   Error = "cannot compare keys using marshaled data"
	KeyStoreUnavailable Error = "key store is unavialable, cannot store keys"
	NoDefaultKeys       Error = "key not found, no default key available"
	NoCachePrivateKeys  Error = "cannot cache private keys in external store"
	NoStorePublicKeys   Error = "private key required for internal store"
	InvalidSource       Error = "key chain is not correctly configured"
)

type Error string

// Error implements the error interface.
func (e Error) Error() string {
	return string(e)
}

// New is the equivalent of errors.New creating a new Error type with the given message
func New(errs string) error {
	return Error(errs)
}

// Fmt is the equivalent of fmt.Errorf creating a new Error type from the formatting directives.
func Fmt(format string, a ...interface{}) error {
	return Error(fmt.Sprintf(format, a...))
}
