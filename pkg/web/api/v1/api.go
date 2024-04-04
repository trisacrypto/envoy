package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/url"
	dberr "self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"
	"strings"
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
	PageSize      int    `json:"page_size" url:"page_size,omitempty" form:"page_size"`
	NextPageToken string `json:"next_page_token" url:"next_page_token,omitempty" form:"next_page_token"`
	PrevPageToken string `json:"prev_page_token" url:"prev_page_token,omitempty" form:"prev_page_token"`
}

//===========================================================================
// Accounts Resource
//===========================================================================

type Account struct {
	ID              ulid.ULID        `json:"id,omitempty"`
	CustomerID      string           `json:"customer_id"`
	FirstName       string           `json:"first_name"`
	LastName        string           `json:"last_name"`
	TravelAddress   string           `json:"travel_address,omitempty"`
	IVMSRecord      string           `json:"ivms101,omitempty"`
	CryptoAddresses []*CryptoAddress `json:"crypto_addresses"`
	Created         time.Time        `json:"created,omitempty"`
	Modified        time.Time        `json:"modified,omitempty"`
}

type CryptoAddress struct {
	ID            ulid.ULID `json:"id,omitempty"`
	CryptoAddress string    `json:"crypto_address"`
	Network       string    `json:"network"`
	AssetType     string    `json:"asset_type"`
	Tag           string    `json:"tag"`
	TravelAddress string    `json:"travel_address,omitempty"`
	Created       time.Time `json:"created,omitempty"`
	Modified      time.Time `json:"modified,omitempty"`
}

type AccountsList struct {
	Page     *PageQuery `json:"page"`
	Accounts []*Account `json:"accounts"`
}

type CryptoAddressList struct {
	Page            *PageQuery       `json:"page"`
	CryptoAddresses []*CryptoAddress `json:"crypto_addresses"`
}

func NewAccount(model *models.Account) (out *Account, err error) {
	out = &Account{
		ID:            model.ID,
		CustomerID:    model.CustomerID.String,
		FirstName:     model.FirstName.String,
		LastName:      model.LastName.String,
		TravelAddress: model.TravelAddress.String,
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
		if !errors.Is(err, dberr.ErrMissingAssociation) {
			return nil, err
		}
	}

	// Add the crypto addresses to the response
	out.CryptoAddresses = make([]*CryptoAddress, 0, len(addresses))
	for _, address := range addresses {
		addr, _ := NewCryptoAddress(address)
		out.CryptoAddresses = append(out.CryptoAddresses, addr)
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
		TravelAddress: sql.NullString{String: a.TravelAddress, Valid: a.TravelAddress != ""},
		IVMSRecord:    nil,
	}

	if a.IVMSRecord != "" {
		model.IVMSRecord = &ivms101.Person{}
		if err = json.Unmarshal([]byte(a.IVMSRecord), model.IVMSRecord); err != nil {
			return nil, err
		}
	}

	if len(a.CryptoAddresses) > 0 {
		addresses := make([]*models.CryptoAddress, 0, len(a.CryptoAddresses))
		for _, address := range a.CryptoAddresses {
			addr, _ := address.Model(model)
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
		TravelAddress: model.TravelAddress.String,
		Created:       model.Created,
		Modified:      model.Modified,
	}, nil
}

func NewCryptoAddressList(page *models.CryptoAddressPage) (out *CryptoAddressList, err error) {
	// TODO: convert PageInfo to PageQuery
	out = &CryptoAddressList{
		Page:            &PageQuery{},
		CryptoAddresses: make([]*CryptoAddress, 0, len(page.CryptoAddresses)),
	}

	for _, model := range page.CryptoAddresses {
		var addr *CryptoAddress
		if addr, err = NewCryptoAddress(model); err != nil {
			return nil, err
		}

		out.CryptoAddresses = append(out.CryptoAddresses, addr)
	}

	return out, nil
}

func (c *CryptoAddress) Model(acct *models.Account) (*models.CryptoAddress, error) {
	addr := &models.CryptoAddress{
		Model: models.Model{
			ID:       c.ID,
			Created:  c.Created,
			Modified: c.Modified,
		},
		CryptoAddress: c.CryptoAddress,
		Network:       c.Network,
		AssetType:     sql.NullString{String: c.AssetType, Valid: c.AssetType != ""},
		Tag:           sql.NullString{String: c.Tag, Valid: c.Tag != ""},
		TravelAddress: sql.NullString{String: c.TravelAddress, Valid: c.TravelAddress != ""},
	}

	if acct != nil {
		addr.SetAccount(acct)
	}
	return addr, nil
}

//===========================================================================
// Counterparty Resource
//===========================================================================

type Counterparty struct {
	ID                  ulid.ULID `json:"id,omitempty"`
	Source              string    `json:"source,omitempty"`
	DirectoryID         string    `json:"directory_id,omitempty"`
	RegisteredDirectory string    `json:"registered_directory,omitempty"`
	Protocol            string    `json:"protocol"`
	CommonName          string    `json:"common_name"`
	Endpoint            string    `json:"endpoint"`
	Name                string    `json:"name"`
	Website             string    `json:"website"`
	Country             string    `json:"country"`
	BusinessCategory    string    `json:"business_category"`
	VASPCategories      []string  `json:"vasp_categories"`
	VerifiedOn          time.Time `json:"verified_on"`
	IVMSRecord          string    `json:"ivms101,omitempty"`
	Created             time.Time `json:"created,omitempty"`
	Modified            time.Time `json:"modified,omitempty"`
}

type CounterpartyList struct {
	Page           *PageQuery      `json:"page"`
	Counterparties []*Counterparty `json:"counterparties"`
}

func NewCounterparty(model *models.Counterparty) (out *Counterparty, err error) {
	out = &Counterparty{
		ID:                  model.ID,
		Source:              model.Source,
		DirectoryID:         model.DirectoryID.String,
		RegisteredDirectory: model.RegisteredDirectory.String,
		Protocol:            model.Protocol,
		CommonName:          model.CommonName,
		Endpoint:            model.Endpoint,
		Name:                model.Name,
		Website:             model.Website.String,
		Country:             model.Country,
		BusinessCategory:    model.BusinessCategory.String,
		VASPCategories:      model.VASPCategories,
		VerifiedOn:          model.VerifiedOn.Time,
		Created:             model.Created,
		Modified:            model.Modified,
	}

	if model.IVMSRecord != nil {
		if data, err := json.Marshal(model.IVMSRecord); err != nil {
			// Log the error but do not stop processing
			log.Error().Err(err).Str("counterparty_id", model.ID.String()).Msg("could not marshal IVMS101 record to JSON")
		} else {
			out.IVMSRecord = string(data)
		}
	}

	return out, nil
}

func NewCounterpartyList(page *models.CounterpartyPage) (out *CounterpartyList, err error) {
	out = &CounterpartyList{
		Page:           &PageQuery{},
		Counterparties: make([]*Counterparty, 0, len(page.Counterparties)),
	}

	for _, model := range page.Counterparties {
		var counterparty *Counterparty
		if counterparty, err = NewCounterparty(model); err != nil {
			return nil, err
		}
		out.Counterparties = append(out.Counterparties, counterparty)
	}

	return out, nil
}

func (c *Counterparty) Validate() (err error) {
	if c.Source != "" {
		err = ValidationError(err, ReadOnlyField("source"))
	}

	if c.DirectoryID != "" {
		err = ValidationError(err, ReadOnlyField("directory_id"))
	}

	if c.RegisteredDirectory != "" {
		err = ValidationError(err, ReadOnlyField("registered_directory"))
	}

	c.Protocol = strings.TrimSpace(strings.ToLower(c.Protocol))
	if c.Protocol == "" {
		err = ValidationError(err, MissingField("protocol"))
	} else {
		if c.Protocol != "trisa" && c.Protocol != "trp" {
			err = ValidationError(err, IncorrectField("protocol", "protocol must be either trisa or trp"))
		}
	}

	if c.CommonName == "" {
		// Set common name to the hostname endpoint if not supplied by default
		if c.Endpoint != "" {
			if u, err := url.Parse(c.Endpoint); err == nil {
				c.CommonName = u.Hostname()
			}
		}

		// If no common name still exists (e.g. endpoint is missing or not parseable)
		// then return a missing field error
		if c.CommonName == "" {
			err = ValidationError(err, MissingField("common_name"))
		}
	}

	if c.Endpoint == "" {
		err = ValidationError(err, MissingField("endpoint"))
	}

	if c.Name == "" {
		err = ValidationError(err, MissingField("name"))
	}

	c.Country = strings.TrimSpace(strings.ToUpper(c.Country))
	if c.Country == "" {
		err = ValidationError(err, MissingField("country"))
	} else {
		if len(c.Country) != 2 {
			err = ValidationError(err, IncorrectField("country", "country must be the two character (alpha-2) country code"))
		}
	}

	return err
}

func (c *Counterparty) Model() (model *models.Counterparty, err error) {
	model = &models.Counterparty{
		Model: models.Model{
			ID:       c.ID,
			Created:  c.Created,
			Modified: c.Modified,
		},
		Source:              c.Source,
		DirectoryID:         sql.NullString{String: c.DirectoryID, Valid: c.DirectoryID != ""},
		RegisteredDirectory: sql.NullString{String: c.RegisteredDirectory, Valid: c.RegisteredDirectory != ""},
		Protocol:            c.Protocol,
		CommonName:          c.CommonName,
		Endpoint:            c.Endpoint,
		Name:                c.Name,
		Website:             sql.NullString{String: c.Website, Valid: c.Website != ""},
		Country:             c.Country,
		BusinessCategory:    sql.NullString{String: c.BusinessCategory, Valid: c.BusinessCategory != ""},
		VASPCategories:      models.VASPCategories(c.VASPCategories),
		VerifiedOn:          sql.NullTime{Time: c.VerifiedOn, Valid: !c.VerifiedOn.IsZero()},
		IVMSRecord:          nil,
	}

	if c.IVMSRecord != "" {
		model.IVMSRecord = &ivms101.LegalPerson{}
		if err = json.Unmarshal([]byte(c.IVMSRecord), model.IVMSRecord); err != nil {
			return nil, err
		}
	}

	return model, nil
}
