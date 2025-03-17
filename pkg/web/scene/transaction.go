package scene

import (
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// Converted from an *api.TransactionList to provide additional UI-specific functionality.
type TransactionList struct {
	Page         *api.TransactionListQuery
	Transactions []*Transaction
}

// Wraps an *api.Transaction to provide additional UI-specific functionality.
type Transaction struct {
	api.Transaction
	Status Status
}

//===========================================================================
// Scene Transaction Helpers
//===========================================================================

func (s Scene) TransactionsList() *TransactionList {
	if data, ok := s[APIData]; ok {
		if txns, ok := data.(*api.TransactionsList); ok {
			out := &TransactionList{
				Page:         txns.Page,
				Transactions: make([]*Transaction, len(txns.Transactions)),
			}

			for i, txn := range txns.Transactions {
				out.Transactions[i] = &Transaction{
					Transaction: *txn,
					Status:      NewStatus(txn.Status),
				}
			}

			return out
		}
	}
	return nil
}

func (s Scene) TransactionDetail() *Transaction {
	if data, ok := s[APIData]; ok {
		if tx, ok := data.(*api.Transaction); ok {
			return &Transaction{
				Transaction: *tx,
				Status:      NewStatus(tx.Status),
			}
		}
	}
	return nil
}

func (s Scene) TransactionCounts() *models.TransactionCounts {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*models.TransactionCounts); ok {
			return out
		}
	}
	return nil
}

//===========================================================================
// Status Helpers
//===========================================================================

// Status wraps a Transaction status to provide additional information such as class,
// tooltip, color, icons, etc for the UI.
type Status struct {
	text    string
	value   enum.Status
	Color   string
	Tooltip string
}

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

func NewStatus(text string) Status {
	status := Status{
		text: text,
	}

	var err error
	if status.value, err = enum.ParseStatus(text); err != nil {
		panic(err)
	}

	switch status.value {
	case enum.StatusUnspecified:
		status.Color = colorUnspecified
		status.Tooltip = tooltipUnspecified
	case enum.StatusDraft:
		status.Color = colorDraft
		status.Tooltip = tooltipDraft
	case enum.StatusPending:
		status.Color = colorPending
		status.Tooltip = tooltipPending
	case enum.StatusReview:
		status.Color = colorReview
		status.Tooltip = tooltipReview
	case enum.StatusRepair:
		status.Color = colorRepair
		status.Tooltip = tooltipRepair
	case enum.StatusAccepted:
		status.Color = colorAccepted
		status.Tooltip = tooltipAccepted
	case enum.StatusCompleted:
		status.Color = colorCompleted
		status.Tooltip = tooltipCompleted
	case enum.StatusRejected:
		status.Color = colorRejected
		status.Tooltip = tooltipRejected
	}

	return status
}

func (s Status) String() string {
	return cases.Title(language.English).String(s.text)
}

func (s Status) Opacity() string {
	if s.value == enum.StatusAccepted {
		return "bg-opacity-75"
	}
	return ""
}

func (s Status) Review() bool {
	return s.value == enum.StatusReview
}

func (s Status) Repair() bool {
	return s.value == enum.StatusRepair
}

func (s Status) ActionRequired() bool {
	ok, _ := enum.CheckStatus(s, enum.StatusReview, enum.StatusRepair, enum.StatusDraft)
	return ok
}

func (s Status) Wait() bool {
	return s.value == enum.StatusPending
}
