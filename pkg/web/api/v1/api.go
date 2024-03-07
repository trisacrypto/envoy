package api

import (
	"context"
	"time"

	"github.com/oklog/ulid/v2"
)

//===========================================================================
// Service Interface
//===========================================================================

// Client defines the service interface for interacting with the TRISA self-hosted node
// internal API (e.g. the API that users can integrate with).
type Client interface {
	Status(context.Context) (*StatusReply, error)

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
	PageSize      int    `json:"page_size" url:"page_size,omitempty" form:"page_size"`
	NextPageToken string `json:"next_page_token" url:"next_page_token,omitempty" form:"next_page_token"`
	PrevPageToken string `json:"prev_page_token" url:"prev_page_token,omitempty" form:"prev_page_token"`
}

//===========================================================================
// Accounts Resource
//===========================================================================

type Account struct {
	ID               ulid.ULID        `json:"id,omitempty"`
	CustomerID       string           `json:"customer_id"`
	FirstName        string           `json:"first_name"`
	LastName         string           `json:"last_name"`
	TravelAddress    string           `json:"travel_address,omitempty"`
	IVMSRecord       string           `json:"ivms101,omitempty"`
	CryptoAdddresses []*CryptoAddress `json:"crypto_addresses"`
	Created          time.Time        `json:"created,omitempty"`
	Modified         time.Time        `json:"modified,omitempty"`
}

type CryptoAddress struct {
	ID            ulid.ULID `json:"id,omitempty"`
	CryptoAddress string    `json:"crypto_address"`
	Network       string    `json:"network"`
	AssetType     string    `json:"asset_type"`
	Tag           string    `json:"tag"`
	Created       time.Time `json:"created,omitempty"`
	Modified      time.Time `json:"modified,omitempty"`
}

type AccountsList struct {
	Page     *PageQuery `json:"page"`
	Accounts []*Account `json:"accounts"`
}

type CryptoAddressList struct {
	Page             *PageQuery       `json:"page"`
	CryptoAdddresses []*CryptoAddress `json:"crypto_addresses"`
}
