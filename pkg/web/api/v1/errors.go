package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/trisacrypto/trisa/pkg/ivms101"
)

//===========================================================================
// Standard Error Handling
//===========================================================================

var (
	Unsuccessful  = Reply{Success: false}
	NotFound      = Reply{Success: false, Error: "resource not found"}
	NotAllowed    = Reply{Success: false, Error: "method not allowed"}
	InternalError = Reply{Success: false, Error: "an internal error occurred"}
)

var (
	ErrInvalidTimestamp     = errors.New("payload timestamp has invalid string format")
	ErrInvalidRejection     = errors.New("envelope does not contain a rejection/repair error")
	ErrParsingIVMS101Person = errors.New("unable to parse IVMS101 person record as json or protocol buffers")
	ErrUnknownEncoding      = errors.New("could not detect data encoding please specify and try again")
)

// Construct a new response for an error or simply return unsuccessful.
func Error(err interface{}) Reply {
	if err == nil {
		return Unsuccessful
	}

	rep := Reply{Success: false}
	switch err := err.(type) {
	case ValidationErrors:
		if len(err) == 1 {
			rep.Error = err.Error()
		} else {
			rep.Error = fmt.Sprintf("%d validation errors occurred", len(err))
			rep.ErrorDetail = make(ErrorDetail, 0, len(err))
			for _, verr := range err {
				rep.ErrorDetail = append(rep.ErrorDetail, &DetailError{
					Field: verr.field,
					Error: verr.Error(),
				})
			}
		}
	case error:
		rep.Error = err.Error()
	case string:
		rep.Error = err
	case fmt.Stringer:
		rep.Error = err.String()
	case json.Marshaler:
		data, e := err.MarshalJSON()
		if e != nil {
			panic(err)
		}
		rep.Error = string(data)
	default:
		rep.Error = "unhandled error response"
	}

	return rep
}

//===========================================================================
// Status Errors
//===========================================================================

// StatusError decodes an error response from the TRISA API.
type StatusError struct {
	StatusCode int
	Reply      Reply
}

func (e *StatusError) Error() string {
	return fmt.Sprintf("[%d] %s", e.StatusCode, e.Reply.Error)
}

// ErrorStatus returns the HTTP status code from an error or 500 if the error is not a StatusError.
func ErrorStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	if e, ok := err.(*StatusError); !ok || e.StatusCode < 100 || e.StatusCode >= 600 {
		return http.StatusInternalServerError
	} else {
		return e.StatusCode
	}
}

//===========================================================================
// Detail Error
//===========================================================================

type ErrorDetail []*DetailError

type DetailError struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

//===========================================================================
// Field Validation Errors
//===========================================================================

func MissingField(field string) *FieldError {
	return &FieldError{verb: "missing", field: field, issue: "this field is required"}
}

func IncorrectField(field, issue string) *FieldError {
	return &FieldError{verb: "invalid field", field: field, issue: issue}
}

func ReadOnlyField(field string) *FieldError {
	return &FieldError{verb: "read-only field", field: field, issue: "this field cannot be written by the user"}
}

func InvalidIVMS101(err *ivms101.FieldError) *FieldError {
	return &FieldError{verb: "", field: err.Field(), issue: err.Error()}
}

func OneOfMissing(fields ...string) *FieldError {
	var fieldstr string
	switch len(fields) {
	case 0:
		panic("no fields specified for one of")
	case 1:
		return MissingField(fields[0])
	default:
		fieldstr = fieldList(fields...)
	}

	return &FieldError{verb: "missing one of", field: fieldstr, issue: "at most one of these fields is required"}
}

func OneOfTooMany(fields ...string) *FieldError {
	if len(fields) < 2 {
		panic("must specify at least two fields for one of too many")
	}
	return &FieldError{verb: "specify only one of", field: fieldList(fields...), issue: "at most one of these fields may be specified"}
}

func ValidationError(err error, errs ...*FieldError) error {
	var verr ValidationErrors
	if err == nil {
		verr = make(ValidationErrors, 0, len(errs))
	} else {
		var ok bool
		if verr, ok = err.(ValidationErrors); !ok {
			verr = make(ValidationErrors, 0, len(errs)+1)
			verr = append(verr, &FieldError{verb: "invalid", field: "input", issue: err.Error()})
		}
	}

	for _, e := range errs {
		if e != nil {
			verr = append(verr, e)
		}
	}

	if len(verr) == 0 {
		return nil
	}
	return verr
}

type FieldError struct {
	verb  string
	field string
	issue string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s %s: %s", e.verb, e.field, e.issue)
}

func (e *FieldError) Subfield(parent string) *FieldError {
	e.field = fmt.Sprintf("%s.%s", parent, e.field)
	return e
}

func (e *FieldError) SubfieldArray(parent string, index int) *FieldError {
	e.field = fmt.Sprintf("%s[%d].%s", parent, index, e.field)
	return e
}

type ValidationErrors []*FieldError

func (e ValidationErrors) Error() string {
	if len(e) == 1 {
		return e[0].Error()
	}

	errs := make([]string, 0, len(e))
	for _, err := range e {
		errs = append(errs, err.Error())
	}

	return fmt.Sprintf("%d validation errors occurred:\n  %s", len(e), strings.Join(errs, "\n  "))
}

func (e ValidationErrors) Map() map[string]string {
	errs := make(map[string]string, len(e))
	for _, err := range e {
		errs[err.field] = err.Error()
	}
	return errs
}

func fieldList(fields ...string) string {
	switch len(fields) {
	case 0:
		return ""
	case 1:
		return fields[0]
	case 2:
		return fmt.Sprintf("%s or %s", fields[0], fields[1])
	default:
		last := len(fields) - 1
		return fmt.Sprintf("%s, or %s", strings.Join(fields[0:last], ", "), fields[last])
	}
}

//===========================================================================
// Convert IVMS101 Validation Errors to API Errors
//===========================================================================

func ConvertIVMS101Errors(err ivms101.ValidationErrors) ValidationErrors {
	verr := make(ValidationErrors, 0, len(err))
	for _, ferr := range err {
		verr = append(verr, InvalidIVMS101(ferr))
	}
	return verr
}
