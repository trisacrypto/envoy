package api

import (
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

//===========================================================================
// ComplianceAuditLog
//===========================================================================

// ComplianceAuditLog stores the information necessary to track changes to
// specific database tables and entries, such as transactions, users, api keys,
// etc. A ComplianceAuditLog entry in the database is meant to be immutable, and
// to prove this each ComplianceAuditLog entry contains a cryptographic Signature
// that can be verified against the other field data.
type ComplianceAuditLog struct {

	// LOG CONTENT FIELDS:

	// ID is a unique indentifier for a ComplianceAuditLog.
	ID ulid.ULID
	// ActorID can be a ULID or UUID, depending on the ActorType.
	ActorID string
	// ActorType allows us to decode the ActorID and is human-readable (an
	// .enum.Actor).
	ActorType string
	// ResourceID can be a ULID or UUID, depending on the ResourceType.
	ResourceID string
	// ResourceType allows us to decode the ResourceID and is human-readable
	// (an enum.Resource).
	ResourceType string
	// ResourceModified is the time the resource was modified at.
	ResourceModified time.Time
	// Action is the type of change made in the database (enum.Action).
	Action string
	// ChangeNotes is an optional string that can include further details. This
	// field may be returned as the empty string depending on the API request
	// options.
	ChangeNotes string

	// SIGNATURE METADATA FIELDS:

	// Signature is a cryptographic Signature that can be used to verify that an
	// instance of a ComplianceAuditLog was not modified. This field may be
	// returned as the empty string depending on the API request options.
	Signature string
	// KeyID is the identification for the public key that can verify this log.
	// This field may be returned as the empty string depending on the API
	// request options.
	KeyID string
	// Algorithm is the identification for the algorithm that can verify this
	// log. This field may be returned as the empty string depending on the API
	// request options.
	Algorithm string
}

// Create a new api.ComplianceAuditLog from a database model.ComplianceAuditLog
func NewComplianceAuditLog(model *models.ComplianceAuditLog) (out *ComplianceAuditLog) {
	out = &ComplianceAuditLog{
		ID:               model.ID,
		ActorID:          string(model.ActorID),
		ActorType:        model.ActorType.String(),
		ResourceID:       string(model.ResourceID),
		ResourceType:     model.ResourceType.String(),
		ResourceModified: model.ResourceModified,
		Action:           model.Action.String(),
	}

	if model.ChangeNotes.Valid {
		out.ChangeNotes = model.ChangeNotes.String
	}

	if model.Signature != nil {
		out.Signature = string(model.Signature)
		out.KeyID = model.KeyID
		out.Algorithm = model.Algorithm
	}

	return out
}

// NOTE: There will be no API endpoint to accept a ComplianceAuditLog, so there
// is no need for a Validate() or a Model() function for it.

//===========================================================================
// ComplianceAuditLogQuery
//===========================================================================

// ComplianceAuditLogQuery stores the input from an API request to the
// ComplianceAuditLog List endpoint
type ComplianceAuditLogQuery struct {
	PageQuery

	// FILTERING OPTIONS

	// ResourceTypes filters results to include only these enum.Resource values
	ResourceTypes []string `json:"resource_types,omitempty"`
	// ResourceID filters results by a specific resource ID
	ResourceID string `json:"resource_id,omitempty"`
	// ResourceTypes filters results to include only these enum.Actor values
	ActorTypes []string `json:"actor_types,omitempty"`
	// ActorID filters results by a specific actor ID
	ActorID string `json:"actor_id,omitempty"`
	// After filters results to include logs with ResourceModified on or after this time (inclusive)
	After *time.Time `json:"after,omitempty"`
	// Before filters results to include logs with ResourceModified before this time (exclusive)
	Before *time.Time `json:"before,omitempty"`

	// DISPLAY OPTIONS

	// DetailedLogs will return the full log entry if true (otherwise List
	// returns only summary info)
	DetailedLogs bool `json:"detailed_logs,omitempty"`
}

// Validates an API ComplianceAuditLogQuery
func (q *ComplianceAuditLogQuery) Validate() (err error) {
	// Check ResourceTypes are valid for the enum values
	if len(q.ResourceTypes) != 0 {
		for _, t := range q.ResourceTypes {
			if ok, _ := enum.CheckResource(t, enum.ResourceTransaction, enum.ResourceUser, enum.ResourceAPIKey, enum.ResourceCounterparty, enum.ResourceAccount, enum.ResourceSunrise); !ok {
				err = ValidationError(err, IncorrectField("resource_types", "must be one of transaction, user, api_key, counterparty, account, or sunrise"))
			}
		}
	}

	// Check ActorTypes are valid for the enum values
	if len(q.ActorTypes) != 0 {
		for _, t := range q.ActorTypes {
			if ok, _ := enum.CheckActor(t, enum.ActorUser, enum.ActorAPIKey, enum.ActorSunrise); !ok {
				err = ValidationError(err, IncorrectField("actor_types", "must be one of user, api_key, or sunrise"))
			}
		}
	}

	// Check After and Before times
	if q.After != nil && !q.After.IsZero() {
		// Check After is not in the future (timeline example: '...(After)->......(Now)...')
		if ok := time.Now().After(*q.After); !ok {
			err = ValidationError(err, IncorrectField("after", "cannot be in the future"))
		}

		// Check Before is after After (timeline example: '...(After)->......<-(Before)...')
		if q.Before != nil && !q.Before.IsZero() {
			if ok := q.Before.After(*q.After); !ok {
				err = ValidationError(err, IncorrectField("before", "before must come before after"))
				err = ValidationError(err, IncorrectField("after", "after must come after before"))
			}
		}
	}

	// NOTE: ResourceID, ActorID, and DetailedLogs require no checks

	return err
}

// Creates a models.ComplianceAuditLogPageInfo from an api.ComplianceAuditLogQuery
func (q *ComplianceAuditLogQuery) Query() (query *models.ComplianceAuditLogPageInfo) {
	query = &models.ComplianceAuditLogPageInfo{
		PageInfo: models.PageInfo{
			PageSize: uint32(q.PageSize),
			// TODO: pagination tokens->IDs
		},
		ResourceTypes: q.ResourceTypes,
		ResourceID:    q.ResourceID,
		ActorTypes:    q.ActorTypes,
		ActorID:       q.ActorID,
		DetailedLogs:  q.DetailedLogs,
	}

	if q.After != nil && !q.After.IsZero() {
		query.After = *q.After
	}

	if q.Before != nil && !q.Before.IsZero() {
		query.Before = *q.Before
	}

	return query
}

//===========================================================================
// ComplianceAuditLogList
//===========================================================================

// ComplianceAuditLogList is the format for outputting a list of logs
type ComplianceAuditLogList struct {
	Page *ComplianceAuditLogQuery `json:"page"`
	Logs []*ComplianceAuditLog    `json:"logs"`
}

// Creates an api.ComplianceAuditLogList from a models.ComplianceAuditLogPage
func NewComplianceAuditLogList(page *models.ComplianceAuditLogPage) (out *ComplianceAuditLogList, err error) {
	out = &ComplianceAuditLogList{
		Page: &ComplianceAuditLogQuery{
			PageQuery: PageQuery{
				PageSize: int(page.Page.PageSize),
			},
			ResourceTypes: page.Page.ResourceTypes,
			ResourceID:    page.Page.ResourceID,
			ActorTypes:    page.Page.ActorTypes,
			ActorID:       page.Page.ActorID,
			After:         &page.Page.After,
			Before:        &page.Page.Before,
			DetailedLogs:  page.Page.DetailedLogs,
		},
		Logs: make([]*ComplianceAuditLog, 0, len(page.Logs)),
	}

	for _, model := range page.Logs {
		out.Logs = append(out.Logs, NewComplianceAuditLog(model))
	}

	return out, nil
}
