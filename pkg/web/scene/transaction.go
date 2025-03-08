package scene

import (
	"fmt"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Converted from an *api.TransactionList to provide additional UI-specific functionality.
type TransactionList struct {
	Page         *api.PageQuery
	Transactions []*Transaction
}

// Wraps an *api.Transaction to provide additional UI-specific functionality.
type Transaction struct {
	api.Transaction
	Status Status
}

//===========================================================================
// Status Helpers
//===========================================================================

// Status wraps a Transaction status to provide additional information such as class,
// tooltip, color, icons, etc for the UI.
type Status string

const (
	colorUnspecified   = "secondary"
	tooltipUnspecified = "The transfer state is unknown or purposefully not specified."

	colorDraft   = "light"
	tooltipDraft = "The TRISA exchange is in a draft state and has not been sent."

	colorPending   = "info"
	tooltipPending = "Action is required by the sending party, await a following RPC."

	colorReview   = "primary"
	tooltipReview = "Action is required by the receiving party."

	colorRepair   = "warning"
	tooltipRepair = "Some part of the payload of the TRISA exchange requires repair."

	colorAccepted   = "success"
	tooltipAccepted = "The TRISA exchange is accepted and the counterparty is awaiting the on-chain transaction."

	colorCompleted   = "success"
	tooltipCompleted = "The TRISA exchange and the on-chain transaction have been completed."

	colorRejected   = "danger"
	tooltipRejected = "The TRISA exchange is rejected and no on-chain transaction should proceed."
)

func (s Status) String() string {
	return cases.Title(language.English).String(string(s))
}

func (s Status) Color() string {
	switch s {
	case models.StatusUnspecified, "":
		return colorUnspecified
	case models.StatusDraft:
		return colorDraft
	case models.StatusPending:
		return colorPending
	case models.StatusReview:
		return colorReview
	case models.StatusRepair:
		return colorRepair
	case models.StatusAccepted:
		return colorAccepted
	case models.StatusCompleted:
		return colorCompleted
	case models.StatusRejected:
		return colorRejected
	default:
		panic(fmt.Errorf("unhandled color for status %q", s))
	}
}

func (s Status) Tooltip() string {
	switch s {
	case models.StatusUnspecified, "":
		return tooltipUnspecified
	case models.StatusDraft:
		return tooltipDraft
	case models.StatusPending:
		return tooltipPending
	case models.StatusReview:
		return tooltipReview
	case models.StatusRepair:
		return tooltipRepair
	case models.StatusAccepted:
		return tooltipAccepted
	case models.StatusCompleted:
		return tooltipCompleted
	case models.StatusRejected:
		return tooltipRejected
	default:
		panic(fmt.Errorf("unhandled tooltip for status %q", s))
	}
}
