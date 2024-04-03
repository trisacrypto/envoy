package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"self-hosted-node/pkg"
)

var (
	Unsuccessful = Reply{Success: false, Version: pkg.Version()}
	NotFound     = Reply{Success: false, Error: "resource not found", Version: pkg.Version()}
	NotAllowed   = Reply{Success: false, Error: "method not allowed", Version: pkg.Version()}
)

// Construct a new response for an error or simply return unsuccessful.
func Error(err interface{}) Reply {
	if err == nil {
		return Unsuccessful
	}

	rep := Reply{Success: false}
	switch err := err.(type) {
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

func MissingField(field string) *FieldError {
	return &FieldError{verb: "missing", field: field, issue: "this field is required"}
}

func IncorrectField(field, issue string) *FieldError {
	return &FieldError{verb: "invalid field", field: field, issue: issue}
}

func ReadOnlyField(field string) *FieldError {
	return &FieldError{verb: "read-only field", field: field, issue: "this field cannot be written by the user"}
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

type FieldError struct {
	verb  string
	field string
	issue string
}

func (e *FieldError) Error() string {
	return fmt.Sprintf("%s %s: %s", e.verb, e.field, e.issue)
}
