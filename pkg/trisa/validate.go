package trisa

import (
	"fmt"
	"time"

	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

var (
	ErrMissingIdentity = &api.Error{
		Code:    api.Error_MISSING_FIELDS,
		Message: "identity payload is required",
		Retry:   true,
	}

	ErrMissingTransaction = &api.Error{
		Code:    api.Error_MISSING_FIELDS,
		Message: "transaction payload is required",
		Retry:   true,
	}

	ErrMissingSentAt = &api.Error{
		Code:    api.Error_MISSING_FIELDS,
		Message: "sent at payload field is required for non-repudiation",
		Retry:   true,
	}

	ErrInvalidTimestamp = &api.Error{
		Code:    api.Error_VALIDATION_ERROR,
		Message: "could not parse payload timestamp as RFC3339 timestamp",
		Retry:   true,
	}
)

var validIdentityTypes = map[string]struct{}{
	"type.googleapis.com/ivms101.IdentityPayload": {},
}

var validTransactionTypes = map[string]struct{}{
	"type.googleapis.com/trisa.data.generic.v1beta1.Transaction": {},
	"type.googleapis.com/trisa.data.generic.v1beta1.Pending":     {},
	"type.googleapis.com/trisa.data.generic.v1beta1.Sunrise":     {},
}

// Validates an incoming TRISA payload, ensuring that it has all required fields and
// handled types for the specified node. If not, a TRISA error is returned.
func Validate(payload *api.Payload) *api.Error {
	if payload.Identity == nil {
		return ErrMissingIdentity
	}

	if payload.Transaction == nil {
		return ErrMissingTransaction
	}

	if payload.SentAt == "" {
		return ErrMissingSentAt
	}

	if _, ok := validIdentityTypes[payload.Identity.TypeUrl]; !ok {
		return &api.Error{
			Code:    api.Error_UNPARSEABLE_IDENTITY,
			Message: fmt.Sprintf("unknown identity payload type %q", payload.Identity.TypeUrl),
			Retry:   true,
		}
	}

	if _, ok := validTransactionTypes[payload.Transaction.TypeUrl]; !ok {
		return &api.Error{
			Code:    api.Error_UNPARSEABLE_TRANSACTION,
			Message: fmt.Sprintf("unknown transaction payload type %q", payload.Transaction.TypeUrl),
			Retry:   true,
		}
	}

	if _, err := time.Parse(time.RFC3339, payload.SentAt); err != nil {
		return ErrInvalidTimestamp
	}

	if payload.ReceivedAt != "" {
		if _, err := time.Parse(time.RFC3339, payload.ReceivedAt); err != nil {
			return ErrInvalidTimestamp
		}
	}

	return nil
}
