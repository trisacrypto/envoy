package api

import (
	"context"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

//===========================================================================
// Service Interface
//===========================================================================

// Client defines the service interface for interacting with the TRISA self-hosted node
// internal API (e.g. the API that users can integrate with).
type Client interface {
	Status(context.Context) (*StatusReply, error)

	// Transactions Resource
	ListTransactions(context.Context, *PageQuery) (*TransactionsList, error)
	CreateTransaction(context.Context, *Transaction) (*Transaction, error)
	TransactionDetail(context.Context, uuid.UUID) (*Transaction, error)
	UpdateTransaction(context.Context, *Transaction) (*Transaction, error)
	DeleteTransaction(context.Context, uuid.UUID) error

	// Accounts Resource
	ListAccounts(context.Context, *PageQuery) (*AccountsList, error)
	CreateAccount(context.Context, *Account) (*Account, error)
	AccountDetail(context.Context, ulid.ULID) (*Account, error)
	UpdateAccount(context.Context, *Account) (*Account, error)
	DeleteAccount(context.Context, ulid.ULID) error

	// CryptoAddress Resource
	ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, in *PageQuery) (*CryptoAddressList, error)
	CreateCryptoAddress(ctx context.Context, accountID ulid.ULID, in *CryptoAddress) (*CryptoAddress, error)
	CryptoAddressDetail(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*CryptoAddress, error)
	UpdateCryptoAddress(ctx context.Context, accountID ulid.ULID, in *CryptoAddress) (*CryptoAddress, error)
	DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error

	// Counterparty Resource
	ListCounterparties(context.Context, *PageQuery) (*CounterpartyList, error)
	CreateCounterparty(context.Context, *Counterparty) (*Counterparty, error)
	CounterpartyDetail(context.Context, ulid.ULID) (*Counterparty, error)
	UpdateCounterparty(context.Context, *Counterparty) (*Counterparty, error)
	DeleteCounterparty(context.Context, ulid.ULID) error
}

//===========================================================================
// Top Level Requests and Responses
//===========================================================================

// Reply contains standard fields that are used for generic API responses and errors.
type Reply struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Version string `json:"version,omitempty"`
}

// Returned on status requests.
type StatusReply struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime,omitempty"`
	Version string `json:"version,omitempty"`
}

// PageQuery manages paginated list requests.
type PageQuery struct {
	PageSize      int    `json:"page_size,omitempty" url:"page_size,omitempty" form:"page_size"`
	NextPageToken string `json:"next_page_token" url:"next_page_token,omitempty" form:"next_page_token"`
	PrevPageToken string `json:"prev_page_token" url:"prev_page_token,omitempty" form:"prev_page_token"`
}
