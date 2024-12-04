package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/sunrise"
)

type Sunrise struct {
	Model
	EnvelopeID uuid.UUID            // A foreign key reference to the Transaction
	Email      string               // Email address of recipients the token is sent to (might be a comma separated list)
	Expiration time.Time            // The timestamp that the sunrise verification token is no longer valid
	Signature  *sunrise.SignedToken // The signed token produced by the sunrise package for verification purposes
	Status     string               // The status of the sunrise message (should be similar to the status of the transaction)
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
