package errors

import "errors"

var (
	ErrDSNParse           = errors.New("could not parse dsn")
	ErrInvalidDSN         = errors.New("could not parse DSN, critical component missing")
	ErrUnknownScheme      = errors.New("database scheme not handled by this package")
	ErrPathRequired       = errors.New("a path is required for this database scheme")
	ErrReadOnly           = errors.New("cannot perform operation in read-only mode")
	ErrMissingAssociation = errors.New("associated record(s) not cached on model")
	ErrMissingReference   = errors.New("missing id of foreign key reference")
	ErrNotFound           = errors.New("record not found in database")
	ErrAlreadyExists      = errors.New("record already exists in database")
	ErrNotImplemented     = errors.New("method not implemented for this storage backend")
	ErrNoIDOnCreate       = errors.New("cannot create a resource with an established id")
	ErrMissingID          = errors.New("missing id of resource")
	ErrIDMismatch         = errors.New("id does not match id of prepared resource")
	ErrNoEndpoint         = errors.New("cannot create travel address: no endpoint defined")
	ErrAmbiguous          = errors.New("ambiguous query: more than one result returned")
)
