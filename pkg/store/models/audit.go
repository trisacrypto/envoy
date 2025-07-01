package models

import (
	"database/sql"
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	"go.rtnl.ai/ulid"
)

// ###########################################################################
// ComplianceAuditLog
// ###########################################################################

// ComplianceAuditLog stores the information necessary to track changes to
// specific database tables and entries, such as transactions, users, api keys,
// etc. A ComplianceAuditLog entry in the database is meant to be immutable, and
// to prove this each ComplianceAuditLog entry contains a cryptographic Signature
// that can be verified against the other field data.
type ComplianceAuditLog struct {
	// ID is a unique indentifier for a ComplianceAuditLog
	ID ulid.ULID
	// ActorID can be a ULID or UUID, depending on the ActorType
	ActorID []byte
	// ActorType allows us to decode the ActorID and is human-readable
	ActorType enum.Actor
	// ResourceID can be a ULID or UUID, depending on the ResourceType
	ResourceID []byte
	// ResourceType allows us to decode the ResourceID and is human-readable
	ResourceType enum.Resource
	// ResourceModified is the time the resource was modified at
	ResourceModified time.Time
	// Action is the type of change made in the database
	Action enum.Action
	// ResourceActionMeta is an optional string specific to the ResourceType and
	// Action that can include further details, such as a JSON changeset or a note
	ResourceActionMeta sql.NullString
	// Signature is a cryptographic Signature that can be used to verify that an
	// instance of a ComplianceAuditLog was not modified
	Signature []byte
	// KeyID is the identification for the public key that can verify this log
	KeyID string
}

// Adds a signature value to the ComplianceAuditLog, replacing any value present.
func (l *ComplianceAuditLog) Sign() error {
	l.Signature = ulid.MakeSecure().Bytes() //TODO (sc-32721): this is a placeholder; sign using the private cert
	l.KeyID = ulid.MakeSecure().String()    //TODO (sc-32721): this is a placeholder; put the public cert's ID here
	return nil
}

// Returns true if the signature on the ComplianceAuditLog is valid for the
// data in the other fields.
func (l *ComplianceAuditLog) Verify() bool {
	return false //TODO(sc-32721): this is a placeholder; validate using the public cert
}

// ###########################################################################
// ComplianceAuditLog Scan/Params
// ###########################################################################

func (l *ComplianceAuditLog) Scan(scanner Scanner) error {
	return scanner.Scan(
		&l.ID,
		&l.ActorID,
		&l.ActorType,
		&l.ResourceID,
		&l.ResourceType,
		&l.ResourceModified,
		&l.Action,
		&l.ResourceActionMeta,
		&l.Signature,
		&l.KeyID,
	)
}

func (l *ComplianceAuditLog) Params() []any {
	return []any{
		sql.Named("id", l.ID),
		sql.Named("actorId", l.ActorID),
		sql.Named("actorType", l.ActorType),
		sql.Named("resourceId", l.ResourceID),
		sql.Named("resourceType", l.ResourceType),
		sql.Named("resourceModified", l.ResourceModified),
		sql.Named("action", l.Action),
		sql.Named("resourceActionMeta", l.ResourceActionMeta),
		sql.Named("signature", l.Signature),
		sql.Named("keyId", l.KeyID),
	}
}

// ###########################################################################
// ComplianceAuditLogPageInfo
// ###########################################################################

// Options for listing ComplianceAuditLog objects from the store interface.
// ResourceTypes and ResourceID are mutually exclusive, as well as ActorTypes
// and ActorID; if both of either pair are provided in an object, then only the
// ID field(s) will be used to filter the result. After and Before may be used
// with any other combination. These options will be concatenated into the SQL
// query using 'AND' logic.
type ComplianceAuditLogPageInfo struct {
	PageInfo
	// ResourceTypes filters results to include only these enum.Resource values
	ResourceTypes []string `json:"resource_types,omitempty"`
	// ResourceID filters results by a specific resource ID
	ResourceID string `json:"resource_id,omitempty"`
	// ResourceTypes filters results to include only these enum.Actor values
	ActorTypes []string `json:"actor_types,omitempty"`
	// ActorID filters results by a specific actor ID
	ActorID string `json:"actor_id,omitempty"`
	// After filters results to include logs with ResourceModified on or after this time (inclusive)
	After time.Time `json:"after,omitempty"`
	// Before filters results to include logs with ResourceModified before this time (exclusive)
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
