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
	// ChangeNotes is an optional string that can include further details
	ChangeNotes sql.NullString

	// Signature is a cryptographic Signature that can be used to verify that an
	// instance of a ComplianceAuditLog was not modified
	Signature []byte
	// KeyID is the identification for the public key that can verify this log
	KeyID string
	// Algorithm is the identification for the algorithm that can verify this log
	Algorithm string
}

// ###########################################################################
// ComplianceAuditLog signing/verification helpers
// ###########################################################################

// Returns the concatenated field data for an audit log, not including the
// signature metadata fields.
func (l *ComplianceAuditLog) Data() (data []byte) {
	// Concatenate all of the fields as bytes in the order they appear in the
	// ComplianceAuditLog. Do not include the signature, key id, or algorithm
	// fields, as those are considered metadata for the signature and not part
	// of the log. DO NOT change the order of these fields in the future, or
	// else verification for old logs could fail; if you add any fields, append
	// to the very end of the byte slice!
	data = append(data, l.ID.Bytes()...)
	data = append(data, l.ActorID...)
	data = append(data, []byte(l.ActorType.String())...)
	data = append(data, l.ResourceID...)
	data = append(data, []byte(l.ResourceType.String())...)
	data = append(data, []byte(l.ResourceModified.String())...)
	data = append(data, []byte(l.Action.String())...)
	if l.ChangeNotes.Valid {

		data = append(data, []byte(l.ChangeNotes.String)...)
	}

	// TODO: append any new fields immediately above this comment; do not reorder!

	return data
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
		&l.ChangeNotes,
		&l.Signature,
		&l.KeyID,
		&l.Algorithm,
	)
}

func (l *ComplianceAuditLog) ScanSummary(scanner Scanner) error {
	return scanner.Scan(
		&l.ID,
		&l.ActorID,
		&l.ActorType,
		&l.ResourceID,
		&l.ResourceType,
		&l.ResourceModified,
		&l.Action,
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
		sql.Named("changeNotes", l.ChangeNotes),
		sql.Named("signature", l.Signature),
		sql.Named("keyId", l.KeyID),
		sql.Named("algorithm", l.Algorithm),
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

	// FILTERING OPTIONS

	// ResourceTypes filters results to include only these enum.Resource values
	ResourceTypes []string
	// ResourceID filters results by a specific resource ID
	ResourceID string
	// ResourceTypes filters results to include only these enum.Actor values
	ActorTypes []string
	// ActorID filters results by a specific actor ID
	ActorID string
	// After filters results to include logs with ResourceModified on or after this time (inclusive)
	After time.Time
	// Before filters results to include logs with ResourceModified before this time (exclusive)
	Before time.Time

	// DISPLAY OPTIONS

	// DetailedLogs will return the full log entry if true (otherwise List
	// returns only summary info)
	DetailedLogs bool
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
		out.DetailedLogs = in.DetailedLogs
	}
	return out
}
