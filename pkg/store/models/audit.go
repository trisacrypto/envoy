package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"go.rtnl.ai/ulid"
)

// ComplianceAuditLog stores the information necessary to track changes to
// specific database tables and entries, such as transactions, users, api keys,
// etc. A ComplianceAuditLog entry in the database is meant to be immutable, and
// to prove this each ComplianceAuditLog entry contains a cryptographic Signature
// that can be verified against the other field data.
type ComplianceAuditLog struct {
	// ID is a unique indentifier for a ComplianceAuditLog
	ID uuid.UUID
	// Timestamp is the time the log was created and signed
	Timestamp time.Time
	// ActorID can be a ULID or UUID, depending on the ActorType
	ActorID []byte
	// ActorType allows us to decode the ActorID and is human-readable
	ActorType enum.Actor
	// ResourceID can be a ULID or UUID, depending on the ResourceType
	ResourceID []byte
	// ResourceType allows us to decode the ResourceID and is human-readable
	ResourceType enum.Resource
	// Action is the type of change made in the database
	Action enum.Action
	// ResourceActionMeta is an optional string specific to the ResourceType and
	// Action that can include further details, such as a JSON changeset or a note
	ResourceActionMeta sql.NullString
	// Signature is a cryptographic signature that can be used to verify that an
	// instance of a ComplianceAuditLog was not modified
	Signature []byte
}

func (l *ComplianceAuditLog) Scan(scanner Scanner) error {
	return scanner.Scan(
		&l.ID,
		&l.Timestamp,
		&l.ActorID,
		&l.ActorType,
		&l.ResourceID,
		&l.ResourceType,
		&l.Action,
		&l.ResourceActionMeta,
		&l.Signature,
	)
}

func (l *ComplianceAuditLog) Params() []any {
	return []any{
		sql.Named("id", l.ID),
		sql.Named("timestamp", l.Timestamp),
		sql.Named("actorId", l.ActorID),
		sql.Named("actorType", l.ActorType),
		sql.Named("resourceId", l.ResourceID),
		sql.Named("resourceType", l.ResourceType),
		sql.Named("action", l.Action),
		sql.Named("resourceActionMeta", l.ResourceActionMeta),
		sql.Named("signature", l.Signature),
	}
}

// Adds a Signature value to the ComplianceAuditLog, replacing any value present.
func (l *ComplianceAuditLog) Sign() error {
	l.Signature = ulid.MakeSecure().Bytes() //FIXME: this is a placeholder; sign using the private cert
	return nil
}

// Returns true if the Signature on the ComplianceAuditLog is valid for the
// data in the other fields.
func (l *ComplianceAuditLog) Verify() (bool, error) {
	valid := false //FIXME: this is a placeholder; validate using the public cert
	return valid, nil
}

// Options for listing ComplianceAuditLog objects from the store interface.
type ComplianceAuditLogPageInfo struct {
	PageInfo
	// ResourceTypes filters results to include only these resource types
	ResourceTypes []enum.Resource `json:"resource_types,omitempty"`
	// ResourceID filters results by a specific resource ID
	ResourceID []byte `json:"resource_id,omitempty"`
	// ResourceTypes filters results to include only these actor types
	ActorTypes []enum.Actor `json:"actor_types,omitempty"`
	// ActorID filters results by a specific actor ID
	ActorID []byte `json:"actor_id,omitempty"`
	// After filters results to include logs created on or after this time (inclusive)
	After time.Time `json:"after,omitempty"`
	// Before filters results to include logs created before this time (exclusive)
	Before time.Time `json:"before,omitempty"`
}

// Copies the page info from the input into a new object, or creates a new
// zero'ed object if the input is nil.
func ComplianceAuditLogPageInfoFrom(in *ComplianceAuditLogPageInfo) (out *ComplianceAuditLogPageInfo) {
	out = &ComplianceAuditLogPageInfo{}
	if in != nil {
		out.PageInfo = *PageInfoFrom(&in.PageInfo)
		out.ResourceTypes = in.ResourceTypes
		out.ResourceID = in.ResourceID
		out.ActorTypes = in.ActorTypes
		out.ActorID = in.ActorID
		out.After = in.After
		out.Before = in.Before
	}
	return out
}
