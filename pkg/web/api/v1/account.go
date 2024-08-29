package api

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"
	"google.golang.org/protobuf/proto"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"github.com/trisacrypto/trisa/pkg/slip0044"
)

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
	CryptoAddresses []*CryptoAddress `json:"crypto_addresses,omitempty"`
	Created         time.Time        `json:"created,omitempty"`
	Modified        time.Time        `json:"modified,omitempty"`
}

type CryptoAddress struct {
	ID            ulid.ULID `json:"id,omitempty"`
	CryptoAddress string    `json:"crypto_address"`
	Network       string    `json:"network"`
	AssetType     string    `json:"asset_type,omitempty"`
	Tag           string    `json:"tag,omitempty"`
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

	// Render the IVMS101 data as as base64 encoded JSON string
	// TODO: select rendering using protocol buffers or JSON as a config option.
	if model.IVMSRecord != nil {
		if data, err := json.Marshal(model.IVMSRecord); err != nil {
			// Log the error but do not stop processing
			log.Error().Err(err).Str("account_id", model.ID.String()).Msg("could not marshal IVMS101 record to JSON")
		} else {
			out.IVMSRecord = base64.URLEncoding.EncodeToString(data)
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
	out = &AccountsList{
		Page:     &PageQuery{},
		Accounts: make([]*Account, 0, len(page.Accounts)),
	}

	for _, model := range page.Accounts {
		var account *Account
		if account, err = NewAccount(model); err != nil {
			return nil, err
		}

		// Remove list fields not needed for the summary info
		// TODO: instead of removing, do not select this data from the database.
		account.IVMSRecord = ""
		account.CryptoAddresses = nil

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
		if model.IVMSRecord, err = a.IVMS101(); err != nil {
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

func (a *Account) IVMS101() (p *ivms101.Person, err error) {
	// Don't handle empty strings.
	if a.IVMSRecord == "" {
		return nil, ErrParsingIVMS101Person
	}

	// Try decoding URL base64 first, then STD before resorting to a string
	var data []byte
	if data, err = base64.URLEncoding.DecodeString(a.IVMSRecord); err != nil {
		if data, err = base64.StdEncoding.DecodeString(a.IVMSRecord); err != nil {
			data = []byte(a.IVMSRecord)
		}
	}

	// Try unmarshaling JSON first, then protocol buffers
	p = &ivms101.Person{}
	if err = json.Unmarshal(data, p); err != nil {
		if err = proto.Unmarshal(data, p); err != nil {
			return nil, ErrParsingIVMS101Person
		}
	}

	return p, nil
}

func (a *Account) Validate(create bool) (err error) {
	if create {
		if !ulids.IsZero(a.ID) {
			err = ValidationError(err, ReadOnlyField("id"))
		}
	}

	if a.LastName == "" {
		err = ValidationError(err, MissingField("last_name"))
	}

	if a.TravelAddress != "" {
		err = ValidationError(err, ReadOnlyField("travel_address"))
	}

	if _, perr := a.IVMS101(); perr != nil {
		err = ValidationError(err, IncorrectField("ivms101", perr.Error()))
	}

	if len(a.CryptoAddresses) > 0 {
		err = ValidationError(err, ReadOnlyField("crypto_addresses"))
	}
	return err
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

func (c *CryptoAddress) Validate(create bool) (err error) {
	if create {
		if !ulids.IsZero(c.ID) {
			err = ValidationError(err, ReadOnlyField("id"))
		}
	}

	c.CryptoAddress = strings.TrimSpace(c.CryptoAddress)
	if c.CryptoAddress == "" {
		err = ValidationError(err, MissingField("crypto_address"))
	}

	c.Network = strings.TrimSpace(c.Network)
	if c.Network == "" {
		err = ValidationError(err, MissingField("network"))
	} else {
		// TODO: also try parsing DTI
		if _, perr := slip0044.ParseCoinType(c.Network); perr != nil {
			err = ValidationError(err, IncorrectField("network", perr.Error()))
		}
	}

	if c.TravelAddress != "" {
		err = ValidationError(err, ReadOnlyField("travel_address"))
	}

	return err
}
