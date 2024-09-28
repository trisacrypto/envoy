package api

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"

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
	encoding        *EncodingQuery   `json:"-"`
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

func NewAccount(model *models.Account, encoding *EncodingQuery) (out *Account, err error) {
	if encoding == nil {
		encoding = &EncodingQuery{}
	}

	out = &Account{
		ID:            model.ID,
		CustomerID:    model.CustomerID.String,
		FirstName:     model.FirstName.String,
		LastName:      model.LastName.String,
		TravelAddress: model.TravelAddress.String,
		Created:       model.Created,
		Modified:      model.Modified,
		encoding:      encoding,
	}

	// Render the IVMS101 data as as base64 encoded JSON string
	if model.IVMSRecord != nil {
		if out.IVMSRecord, err = out.encoding.Marshal(model.IVMSRecord); err != nil {
			// Log the error but do not stop processing
			log.Error().Err(err).
				Str("account_id", model.ID.String()).
				Str("encoding", encoding.Encoding).
				Str("format", encoding.Format).
				Bool("is_base64_std", encoding.b64std).
				Msg("could not marshal IVMS101 record")
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
		if account, err = NewAccount(model, nil); err != nil {
			return nil, err
		}

		// Ensure the fields not returned in ScanSummary are set to zero-values so that
		// they are omitted from the JSON response (the database does not return these).
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

	if a.encoding == nil {
		a.encoding = &EncodingQuery{}
	}

	p = &ivms101.Person{}
	if err = a.encoding.Unmarshal(a.IVMSRecord, p); err != nil {
		return nil, err
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

	if a.IVMSRecord != "" {
		if record, perr := a.IVMS101(); perr != nil {
			err = ValidationError(err, IncorrectField("ivms101", perr.Error()))
		} else if verr := record.Validate(); verr != nil {
			switch e := verr.(type) {
			case ivms101.ValidationErrors:
				for _, ve := range e {
					err = ValidationError(err, InvalidIVMS101(ve))
				}
			case *ivms101.FieldError:
				err = ValidationError(err, InvalidIVMS101(e))
			default:
				err = ValidationError(err, IncorrectField("ivms101", verr.Error()))
			}
		}
	}

	if len(a.CryptoAddresses) > 0 {
		for i, address := range a.CryptoAddresses {
			if cerr := address.Validate(create); cerr != nil {
				switch e := cerr.(type) {
				case ValidationErrors:
					for _, fe := range e {
						err = ValidationError(err, fe.SubfieldArray("crypto_addresses", i))
					}
				case *FieldError:
					err = ValidationError(err, e.SubfieldArray("crypto_addresses", i))
				default:
					panic(fmt.Errorf("unhandled validation error type %T on crypto_addresses[%d]", e, i))
				}
			}
		}
	}
	return err
}

func (a *Account) SetEncoding(encoding *EncodingQuery) {
	if encoding == nil {
		encoding = &EncodingQuery{}
	}
	a.encoding = encoding
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
