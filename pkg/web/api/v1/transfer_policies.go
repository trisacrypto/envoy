package api

import (
	"time"

	"go.rtnl.ai/ulid"
)

type TransferPolicyQuery struct {
	PageQuery
	Action   string `json:"action,omitempty" url:"action,omitempty" form:"action"`
	Protocol string `json:"protocol,omitempty" url:"protocol,omitempty" form:"protocol"`
}

type TransferPolicyList struct {
	Page        *PageQuery                 `json:"page,omitempty"`
	Policies    []*TransferPolicy          `json:"policies,omitempty"`
	TotalCounts *TransferPolicyMatchCounts `json:"counts,omitempty"`       // total count of transfers that have matched all policies, can be filtered by protocol
	PolicyCount int64                      `json:"policy_count,omitempty"` // total number of policies, can be filtered by protocol
}

type TransferPolicy struct {
	ID         ulid.ULID    `json:"id,omitempty"`
	Name       string       `json:"name,omitempty"`
	Protocol   string       `json:"protocol,omitempty"`
	Condition  []*Condition `json:"conditions,omitempty"` // list of conditions that must be met for the policy to match a transfer
	Action     *Action      `json:"action,omitempty"`     // action to take when the policy matches a transfer
	MatchCount int64        `json:"count,omitempty"`      // number of transfers that match this policy
	Enabled    bool         `json:"enabled"`
	Created    time.Time    `json:"created,omitempty"`
	Modified   time.Time    `json:"modified,omitempty"`
}

type Condition struct {
	Counterparties []ulid.ULID `json:"counterparties,omitempty"` // list of counterparty IDs that match the policy
	Jurisdictions  []string    `json:"jurisdictions,omitempty"`  // list of jurisdictions that match the policy
	IVMSCheck      *IVMSCheck  `json:"ivms_check,omitempty"`     // IVMS101 fields that are required or missing from a transfer in order for the policy to match
	Limit          *Limit      `json:"amount_limit,omitempty"`   // transfer amount limit
}

// IVMSCheck defines optional criteria for IVMS101 fields in a transfer.
// An empty field list leaves its criterion unset. If both lists are empty or the
// check is omitted, the policy skips IVMS evaluation and evaluates its other criteria.
type IVMSCheck struct {
	PresentFields []string `json:"present_fields,omitempty"` // criterion matches if every listed field is present in the transfer
	MissingFields []string `json:"missing_fields,omitempty"` // criterion matches if any listed field is missing from the transfer
}

type Limit struct {
	Networks  []string `json:"networks,omitempty"`
	Condition string   `json:"condition,omitempty"` // condition: below, at_or_above
	Amount    float64  `json:"amount,omitempty"`
	Enabled   bool     `json:"enabled,omitempty"`
}

type Action struct {
	Type       string `json:"type,omitempty"`        // type: auto, label, or archive
	Outcome    string `json:"outcome,omitempty"`     // outcome for auto or label type: accept, reject, repair, needs_review, follow_up, high_risk
	ReasonCode string `json:"reason_code,omitempty"` // reason code provided when auto decision is reject or repair
	Message    string `json:"message,omitempty"`     // message provided when auto decision is reject or repair
}

type TransferPolicyMatchCounts struct {
	Total   int64 `json:"total,omitempty"`
	Auto    int64 `json:"auto,omitempty"`
	Label   int64 `json:"label,omitempty"`
	Archive int64 `json:"archive,omitempty"`
}
