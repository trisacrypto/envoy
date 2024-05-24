package api

import (
	"context"
	"io"

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
	Login(context.Context, *LoginRequest) (*LoginReply, error)
	Authenticate(context.Context, *APIAuthentication) (*LoginReply, error)
	Reauthenticate(context.Context, *ReauthenticateRequest) (*LoginReply, error)

	// Transactions Resource
	ListTransactions(context.Context, *PageQuery) (*TransactionsList, error)
	CreateTransaction(context.Context, *Transaction) (*Transaction, error)
	TransactionDetail(context.Context, uuid.UUID) (*Transaction, error)
	UpdateTransaction(context.Context, *Transaction) (*Transaction, error)
	DeleteTransaction(context.Context, uuid.UUID) error

	// Transaction Actions
	Prepare(context.Context, *Prepare) (*Prepared, error)
	SendPrepared(context.Context, *Prepared) (*Transaction, error)
	Export(context.Context, io.Writer) error

	// Transaction Detail Actions
	Preview(ctx context.Context, transactionID uuid.UUID) (*Envelope, error)
	SendEnvelope(ctx context.Context, transactionID uuid.UUID, in *Envelope) (*Envelope, error)
	Accept(ctx context.Context, transactionID uuid.UUID, in *Envelope) (*Envelope, error)
	Reject(ctx context.Context, transactionID uuid.UUID, in *Rejection) (*Envelope, error)

	// SecureEnvelopes Resource
	ListSecureEnvelopes(ctx context.Context, transactionID uuid.UUID, in *EnvelopeListQuery) (*EnvelopesList, error)
	SecureEnvelopeDetail(ctx context.Context, transactionID uuid.UUID, envID ulid.ULID) (*SecureEnvelope, error)
	DecryptedEnvelopeDetail(ctx context.Context, transactionID uuid.UUID, envID ulid.ULID) (*Envelope, error)
	DeleteSecureEnvelope(ctx context.Context, transactionID uuid.UUID, envID ulid.ULID) error

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

	// Utilities
	EncodeTravelAddress(context.Context, *TravelAddress) (*TravelAddress, error)
	DecodeTravelAddress(context.Context, *TravelAddress) (*TravelAddress, error)
}

//===========================================================================
// Top Level Requests and Responses
//===========================================================================

// Reply contains standard fields that are used for generic API responses and errors.
type Reply struct {
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	Version     string      `json:"version,omitempty"`
	ErrorDetail ErrorDetail `json:"errors,omitempty"`
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
