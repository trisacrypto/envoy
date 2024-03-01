package models

import (
	"self-hosted-node/pkg/store/errors"

	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

// Account holds information about a user account for the local VASP including
// identifying information to link the account to the user record and IVMS101 data.
// Accounts automatically generate a TravelAddress for creating travel rule transactions
// with the specified VASPs. Accounts can also be associated with one or more wallet
// addresses for specific crypto currencies and networks.
type Account struct {
	Model
	CustomerID    string           `json:"customer_id"`    // Account ID of internal user record (optional)
	FirstName     string           `json:"first_name"`     // First name (forename) of user
	LastName      string           `json:"last_name"`      // Last name (surname) of user
	TravelAddress string           `json:"travel_address"` // Generated TravelAddress for this user
	IVMSRecord    *ivms101.Person  `json:"ivms101"`        // IVMS101 record for the account
	addresses     []*CryptoAddress `json:"-"`              // Associated crypto addresses
}

type CryptoAddress struct {
	Model
	AccountID     ulid.ULID `json:"account_id"`     // Reference to account the crypto address belongs to
	CryptoAddress string    `json:"crypto_address"` // The actual crypto address of the wallet
	Network       string    `json:"network"`        // The network associated with the crypto address in SIP0044 encoding
	AssetType     string    `json:"asset_type"`     // The asset type with the crypto address (optional)
	Tag           string    `json:"tag"`            // The memo or destination tag associated with the address (optional)
	account       *Account  `json:"-"`              // Associated account
}

// Returns associated crypto addresses if they are cached on the account model, returns
// an ErrMissingAssociation error if they are not cached.
func (a *Account) CryptoAddresses() ([]*CryptoAddress, error) {
	if a.addresses == nil {
		return nil, errors.ErrMissingAssociation
	}
	return a.addresses, nil
}

// Used by store implementations to cache associated crypto addresses on the account.
func (a *Account) SetCryptoAddresses(addresses []*CryptoAddress) {
	a.addresses = addresses
}

// Returns associated account if it is cached on the crypto address model, returns an
// ErrMissingAssociation error if they are not cached.
func (a *CryptoAddress) Account() (*Account, error) {
	if a.account == nil {
		return nil, errors.ErrMissingAssociation
	}
	return a.account, nil
}

// Used by store implementations to cache associated account on the crypto address.
func (a *CryptoAddress) SetAccount(account *Account) {
	a.account = account
}
