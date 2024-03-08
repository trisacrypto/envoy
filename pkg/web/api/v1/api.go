package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"self-hosted-node/pkg/store/models"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
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

func NewAccount(model *models.Account) (out *Account, err error) {
	out = &Account{
		ID:            model.ID,
		CustomerID:    model.CustomerID.String,
		FirstName:     model.FirstName.String,
		LastName:      model.LastName.String,
		TravelAddress: model.TravelAddress,
		Created:       model.Created,
		Modified:      model.Modified,
	}

	// Render the IVMS101 data as as JSON string
	if model.IVMSRecord != nil {
		if data, err := json.Marshal(model.IVMSRecord); err != nil {
			// Log the error but do not stop processing
			log.Error().Err(err).Str("account_id", model.ID.String()).Msg("could not marshal IVMS101 record to JSON")
		} else {
			out.IVMSRecord = string(data)
		}
	}

	// Collect the crypto address associations
	var addresses []*models.CryptoAddress
	if addresses, err = model.CryptoAddresses(); err != nil {
		return nil, err
	}

	// Add the crypto addresses to the response
	out.CryptoAdddresses = make([]*CryptoAddress, 0, len(addresses))
	for _, address := range addresses {
		addr, _ := NewCryptoAddress(address)
		out.CryptoAdddresses = append(out.CryptoAdddresses, addr)
	}

	return out, nil
}

func NewAccountList(page *models.AccountsPage) (out *AccountsList, err error) {
	// TODO: convert PageInfo to PageQuery
	out = &AccountsList{
		Page:     &PageQuery{},
		Accounts: make([]*Account, 0, len(page.Accounts)),
	}

	for _, model := range page.Accounts {
		var account *Account
		if account, err = NewAccount(model); err != nil {
			return nil, err
		}

		out.Accounts = append(out.Accounts, account)
	}

	return out, nil
}

func (a *Account) Model() (model *models.Account, err error) {
	model = &models.Account{
		Model: models.Model{
			ID:       a.ID,
			Created:  a.Created,
			Modified: a.Modified,
		},
		CustomerID:    sql.NullString{String: a.CustomerID, Valid: a.CustomerID != ""},
		FirstName:     sql.NullString{String: a.FirstName, Valid: a.FirstName != ""},
		LastName:      sql.NullString{String: a.LastName, Valid: a.LastName != ""},
		TravelAddress: a.TravelAddress,
		IVMSRecord:    nil,
	}

	if a.IVMSRecord != "" {
		model.IVMSRecord = &ivms101.Person{}
		if err = json.Unmarshal([]byte(a.IVMSRecord), model.IVMSRecord); err != nil {
			return nil, err
		}
	}

	if len(a.CryptoAdddresses) > 0 {
		addresses := make([]*models.CryptoAddress, 0, len(a.CryptoAdddresses))
		for _, address := range a.CryptoAdddresses {
			addr, _ := address.Model()
			addresses = append(addresses, addr)
		}

		model.SetCryptoAddresses(addresses)
	}

	return model, nil
}

func NewCryptoAddress(model *models.CryptoAddress) (*CryptoAddress, error) {
	return &CryptoAddress{
		ID:            model.ID,
		CryptoAddress: model.CryptoAddress,
		Network:       model.Network,
		AssetType:     model.AssetType.String,
		Tag:           model.Tag.String,
		Created:       model.Created,
		Modified:      model.Modified,
	}, nil
}

func NewCryptoAddressList(page *models.CryptoAddressPage) (out *CryptoAddressList, err error) {
	// TODO: convert PageInfo to PageQuery
	out = &CryptoAddressList{
		Page:             &PageQuery{},
		CryptoAdddresses: make([]*CryptoAddress, 0, len(page.CryptoAddresses)),
	}

	for _, model := range page.CryptoAddresses {
		var addr *CryptoAddress
		if addr, err = NewCryptoAddress(model); err != nil {
			return nil, err
		}

		out.CryptoAdddresses = append(out.CryptoAdddresses, addr)
	}

	return out, nil
}

func (c *CryptoAddress) Model() (*models.CryptoAddress, error) {
	return &models.CryptoAddress{
		Model: models.Model{
			ID:       c.ID,
			Created:  c.Created,
			Modified: c.Modified,
		},
		CryptoAddress: c.CryptoAddress,
		Network:       c.Network,
		AssetType:     sql.NullString{String: c.AssetType, Valid: c.AssetType != ""},
		Tag:           sql.NullString{String: c.Tag, Valid: c.Tag != ""},
	}, nil
}
