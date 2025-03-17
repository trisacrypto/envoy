package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/sunrise"
)

type Sunrise struct {
	Model
	EnvelopeID uuid.UUID            // A foreign key reference to the Transaction
	Email      string               // Email address of recipients the token is sent to (might be a comma separated list)
	Expiration time.Time            // The timestamp that the sunrise verification token is no longer valid
	Signature  *sunrise.SignedToken // The signed token produced by the sunrise package for verification purposes
	Status     enum.Status          // The status of the sunrise message (should be similar to the status of the transaction)
	SentOn     sql.NullTime         // The timestamp that the email message was sent
	VerifiedOn sql.NullTime         // The last timestamp that the user verified the token
}

// Scans a complete SELECT into the Sunrise model
func (s *Sunrise) Scan(scanner Scanner) error {
	return scanner.Scan(
		&s.ID,
		&s.EnvelopeID,
		&s.Email,
		&s.Expiration,
		&s.Signature,
		&s.Status,
		&s.SentOn,
		&s.VerifiedOn,
		&s.Created,
		&s.Modified,
	)
}

// Scans a partial SELECT into the Sunrise model for listing the sunrise model
func (s *Sunrise) ScanSummary(scanner Scanner) error {
	return scanner.Scan(
		&s.ID,
		&s.EnvelopeID,
		&s.Expiration,
		&s.Status,
		&s.SentOn,
		&s.VerifiedOn,
	)
}

// Get the complete named params of the sunrise message from the model.
func (s *Sunrise) Params() []any {
	return []any{
		sql.Named("id", s.ID),
		sql.Named("envelopeID", s.EnvelopeID),
		sql.Named("email", s.Email),
		sql.Named("expiration", s.Expiration),
		sql.Named("signature", s.Signature),
		sql.Named("status", s.Status),
		sql.Named("sentOn", s.SentOn),
		sql.Named("verifiedOn", s.VerifiedOn),
		sql.Named("created", s.Created),
		sql.Named("modified", s.Modified),
	}
}

// IsExpired returns true if the message expiration is before the current time and if
// the status is not in a final state (e.g. completed or rejected). If the status is
// in a final state, then the message is not considered expired no matter the exiration
// timestamp.
func (s *Sunrise) IsExpired() bool {
	return time.Now().After(s.Expiration) && s.Status != enum.StatusCompleted && s.Status != enum.StatusRejected
}
