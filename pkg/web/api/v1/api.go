package api

import (
	"context"
	"io"
	"strings"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

func init() {
	ivms101.AllowRekeying()
	ivms101.DisallowUnknownFields()
}

//===========================================================================
// Service Interface
//===========================================================================

// Client defines the service interface for interacting with the TRISA self-hosted node
// internal API (e.g. the API that users can integrate with).
type Client interface {
	Status(context.Context) (*StatusReply, error)
	DBInfo(context.Context) (*DBInfo, error)
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

	// Temporary Sunrise Action
	SendSunrise(context.Context, *Sunrise) (*Transaction, error)

	// Transaction Detail Actions
	SendEnvelope(ctx context.Context, transactionID uuid.UUID, in *Envelope) (*Envelope, error)
	LatestPayloadEnvelope(ctx context.Context, transactionID uuid.UUID) (*Envelope, error)
	AcceptPreview(ctx context.Context, transactionID uuid.UUID) (*Envelope, error)
	Accept(ctx context.Context, transactionID uuid.UUID, in *Envelope) (*Envelope, error)
	Reject(ctx context.Context, transactionID uuid.UUID, in *Rejection) (*Envelope, error)
	RepairPreview(ctx context.Context, transactionID uuid.UUID) (*Repair, error)
	Repair(ctx context.Context, transactionID uuid.UUID, in *Envelope) (*Envelope, error)
	ArchiveTransaction(context.Context, uuid.UUID) error

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
	SearchCounterparties(context.Context, *SearchQuery) (*CounterpartyList, error)
	ListCounterparties(context.Context, *PageQuery) (*CounterpartyList, error)
	CreateCounterparty(context.Context, *Counterparty) (*Counterparty, error)
	CounterpartyDetail(context.Context, ulid.ULID) (*Counterparty, error)
	UpdateCounterparty(context.Context, *Counterparty) (*Counterparty, error)
	DeleteCounterparty(context.Context, ulid.ULID) error

	// Contacts Resource
	ListContacts(ctx context.Context, counterpartyID ulid.ULID, in *PageQuery) (*ContactList, error)
	CreateContact(ctx context.Context, counterpartyID ulid.ULID, in *Counterparty) (*Contact, error)
	ContactDetail(ctx context.Context, counterpartyID, contactID ulid.ULID) (*Contact, error)
	UpdateContact(ctx context.Context, counterpartyID ulid.ULID, in *Counterparty) (*Contact, error)
	DeleteContact(ctx context.Context, counterpartyID, contactID ulid.ULID) error

	// Users Resource
	ListUsers(context.Context, *PageQuery) (*UserList, error)
	CreateUser(context.Context, *User) (*User, error)
	UserDetail(context.Context, ulid.ULID) (*User, error)
	UpdateUser(context.Context, *User) (*User, error)
	DeleteUser(context.Context, ulid.ULID) error

	// APIKey Resource
	ListAPIKeys(context.Context, *PageQuery) (*APIKeyList, error)
	CreateAPIKey(context.Context, *APIKey) (*APIKey, error)
	APIKeyDetail(context.Context, ulid.ULID) (*APIKey, error)
	UpdateAPIKey(context.Context, *APIKey) (*APIKey, error)
	DeleteAPIKey(context.Context, ulid.ULID) error

	// Utilities
	EncodeTravelAddress(context.Context, *TravelAddress) (*TravelAddress, error)
	DecodeTravelAddress(context.Context, *TravelAddress) (*TravelAddress, error)
	ValidateIVMS101(context.Context, []byte) (*ivms101.IdentityPayload, error)
}

//===========================================================================
// Top Level Requests and Responses
//===========================================================================

// Reply contains standard fields that are used for generic API responses and errors.
type Reply struct {
	Success     bool        `json:"success"`
	Error       string      `json:"error,omitempty"`
	ErrorDetail ErrorDetail `json:"errors,omitempty"`
}

// Returned on status requests.
type StatusReply struct {
	Status  string `json:"status"`
	Uptime  string `json:"uptime,omitempty"`
	Version string `json:"version,omitempty"`
}

// A copy of the sql.DBStats struct that implements JSON serialization.
type DBInfo struct {
	MaxOpenConnections int    `json:"max_open_connections"` // Maximum number of open connections to the database.
	OpenConnections    int    `json:"open_connections"`     // The number of established connections both in use and idle.
	InUse              int    `json:"in_use"`               // The number of connections currently in use.
	Idle               int    `json:"idle"`                 // The number of idle connections.
	WaitCount          int64  `json:"wait_count"`           // The total number of connections waited for.
	WaitDuration       string `json:"wait_duration"`        // The total time blocked waiting for a new connection.
	MaxIdleClosed      int64  `json:"max_idle_closed"`      // The total number of connections closed due to SetMaxIdleConns.
	MaxIdleTimeClosed  int64  `json:"max_idle_time_closed"` // The total number of connections closed due to SetConnMaxIdleTime.
	MaxLifetimeClosed  int64  `json:"max_lifetime_closed"`  // The total number of connections closed due to SetConnMaxLifetime.
}

// PageQuery manages paginated list requests.
type PageQuery struct {
	PageSize      int    `json:"page_size,omitempty" url:"page_size,omitempty" form:"page_size"`
	NextPageToken string `json:"next_page_token,omitempty" url:"next_page_token,omitempty" form:"next_page_token"`
	PrevPageToken string `json:"prev_page_token,omitempty" url:"prev_page_token,omitempty" form:"prev_page_token"`
}

// SearchQuery manages fuzzy string searches.
type SearchQuery struct {
	Query string `json:"query,omitempty" url:"query,omitempty" form:"query"`
	Limit int    `json:"limit,omitempty" url:"limit,omitempty" form:"limit"`
}

func (q *SearchQuery) Validate() (err error) {
	q.Query = strings.TrimSpace(q.Query)
	if q.Query == "" {
		err = ValidationError(err, MissingField("query"))
	}

	if q.Limit < 0 {
		err = ValidationError(err, IncorrectField("limit", "limit cannot be less than zero"))
	}

	if q.Limit > 50 {
		err = ValidationError(err, IncorrectField("limit", "maximum number of search results that can be returned is 50"))
	}

	return err
}

func (q *SearchQuery) Model() *models.SearchQuery {
	return &models.SearchQuery{Query: q.Query, Limit: q.Limit}
}
