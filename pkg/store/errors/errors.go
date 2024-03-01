package errors

import "errors"

var (
	ErrDSNParse           = errors.New("could not parse dsn")
	ErrInvalidDSN         = errors.New("could not parse DSN, critical component missing")
	ErrUnknownScheme      = errors.New("database scheme not handled by this package")
	ErrPathRequired       = errors.New("a path is required for this database scheme")
	ErrReadOnly           = errors.New("cannot perform operation in read-only mode")
	ErrMissingAssociation = errors.New("associated record(s) not cached on model")
	ErrNotFound           = errors.New("record not found in database")
)
