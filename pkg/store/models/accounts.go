package models

import (
	"database/sql"

	"github.com/trisacrypto/envoy/pkg/store/errors"

	"github.com/trisacrypto/trisa/pkg/ivms101"
	"go.rtnl.ai/ulid"
)

// Account holds information about a user account for the local VASP including
// identifying information to link the account to the user record and IVMS101 data.
// Accounts automatically generate a TravelAddress for creating travel rule transactions
// with the specified VASPs. Accounts can also be associated with one or more wallet
// addresses for specific crypto currencies and networks.
type Account struct {
	Model
	CustomerID    sql.NullString   // Account ID of internal user record (optional)
	FirstName     sql.NullString   // First name (forename) of user
	LastName      sql.NullString   // Last name (surname) of user
	TravelAddress sql.NullString   // Generated TravelAddress for this user
	IVMSRecord    *ivms101.Person  // IVMS101 record for the account
	addresses     []*CryptoAddress // Associated crypto addresses
}

type CryptoAddress struct {
	Model
	AccountID     ulid.ULID      // Reference to account the crypto address belongs to
	CryptoAddress string         // The actual crypto address of the wallet
	Network       string         // The network associated with the crypto address in SIP0044 encoding
	AssetType     sql.NullString // The asset type with the crypto address (optional)
	Tag           sql.NullString // The memo or destination tag associated with the address (optional)
	TravelAddress sql.NullString // Generated TravelAddress for this wallet address
	account       *Account       // Associated account
}

// Scan a complete SELECT into the account model.
func (a *Account) Scan(scanner Scanner) error {
	return scanner.Scan(
		&a.ID,
		&a.CustomerID,
		&a.FirstName,
		&a.LastName,
		&a.TravelAddress,
		&a.IVMSRecord,
		&a.Created,
		&a.Modified,
	)
}

// ScanSummary scans only the summary information into the account model.
func (a *Account) ScanSummary(scanner Scanner) error {
	return scanner.Scan(
		&a.ID,
		&a.CustomerID,
		&a.FirstName,
		&a.LastName,
		&a.TravelAddress,
		&a.Created,
		&a.Modified,
	)
}

// Get the complete named params of the account from the model.
func (a *Account) Params() []any {
	return []any{
		sql.Named("id", a.ID),
		sql.Named("customerID", a.CustomerID),
		sql.Named("firstName", a.FirstName),
		sql.Named("lastName", a.LastName),
		sql.Named("travelAddress", a.TravelAddress),
		sql.Named("ivms101", a.IVMSRecord),
		sql.Named("created", a.Created),
		sql.Named("modified", a.Modified),
	}
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

// Scans a complete SELECT into the CryptoAddress model
func (a *CryptoAddress) Scan(scanner Scanner) error {
	return scanner.Scan(
		&a.ID,
		&a.AccountID,
		&a.CryptoAddress,
		&a.Network,
		&a.AssetType,
		&a.Tag,
		&a.TravelAddress,
		&a.Created,
		&a.Modified,
	)
}

// Get the complete named params of the crypto address from the model.
func (a *CryptoAddress) Params() []any {
	return []any{
		sql.Named("id", a.ID),
		sql.Named("accountID", a.AccountID),
		sql.Named("cryptoAddress", a.CryptoAddress),
		sql.Named("network", a.Network),
		sql.Named("assetType", a.AssetType),
		sql.Named("tag", a.Tag),
		sql.Named("travelAddress", a.TravelAddress),
		sql.Named("created", a.Created),
		sql.Named("modified", a.Modified),
	}
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
