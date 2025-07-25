package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"net/mail"
	"net/url"

	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"go.rtnl.ai/ulid"
)

const (
	FieldCommonName = "common_name"
	FieldName       = "name"
	FieldLEI        = "lei"
)

// ###########################################################################
// Counterparty
// ###########################################################################

// TODO: how to incorporate the TRIXO form into this model?
type Counterparty struct {
	Model
	Source              enum.Source          // either directory or locally created
	DirectoryID         sql.NullString       // the directory ID associated with the counterparty (directory only)
	RegisteredDirectory sql.NullString       // the registered directory of the counterparty (directory only)
	Protocol            enum.Protocol        // the protocol to use to send travel rule information (TRISA, TRP, Sunrise, etc.)
	CommonName          string               // common name - a unique name to identify the endpoint
	Endpoint            string               // the full endpoint to connect to the counterparty on
	Name                string               // the counterparty's legal entity name
	Website             sql.NullString       // website with more information about the counterparty
	Country             sql.NullString       // country the counterparty is associated with
	BusinessCategory    sql.NullString       // the business category of the counterparty
	VASPCategories      VASPCategories       // the categories of how the VASP handles crypto assets
	VerifiedOn          sql.NullTime         // the datetime the VASP was verified in the directory (directory only)
	IVMSRecord          *ivms101.LegalPerson // IVMS101 record for the counterparty
	LEI                 sql.NullString       // Legal Entity Identifier for the counterparty (generally for TRP)
	contacts            []*Contact           // Associated contacts if any
}

type CounterpartyPageInfo struct {
	PageInfo
	Source string `json:"source,omitempty"`
}

// Scan a complete SELECT into the counterparty model
func (c *Counterparty) Scan(scanner Scanner) error {
	return scanner.Scan(
		&c.ID,
		&c.Source,
		&c.DirectoryID,
		&c.RegisteredDirectory,
		&c.Protocol,
		&c.CommonName,
		&c.Endpoint,
		&c.Name,
		&c.Website,
		&c.Country,
		&c.BusinessCategory,
		&c.VASPCategories,
		&c.VerifiedOn,
		&c.IVMSRecord,
		&c.Created,
		&c.Modified,
		&c.LEI,
	)
}

// Scan a partial SELECT into the counterparty model
func (c *Counterparty) ScanSummary(scanner Scanner) error {
	return scanner.Scan(
		&c.ID,
		&c.Source,
		&c.Protocol,
		&c.Endpoint,
		&c.Name,
		&c.Website,
		&c.Country,
		&c.VerifiedOn,
		&c.Created,
	)
}

// Get complete named params of the counterparty from the model.
func (c *Counterparty) Params() []any {
	return []any{
		sql.Named("id", c.ID),
		sql.Named("source", c.Source),
		sql.Named("directoryID", c.DirectoryID),
		sql.Named("registeredDirectory", c.RegisteredDirectory),
		sql.Named("protocol", c.Protocol),
		sql.Named("commonName", c.CommonName),
		sql.Named("endpoint", c.Endpoint),
		sql.Named("name", c.Name),
		sql.Named("website", c.Website),
		sql.Named("country", c.Country),
		sql.Named("businessCategory", c.BusinessCategory),
		sql.Named("vaspCategories", c.VASPCategories),
		sql.Named("verifiedOn", c.VerifiedOn),
		sql.Named("ivms101", c.IVMSRecord),
		sql.Named("created", c.Created),
		sql.Named("modified", c.Modified),
		sql.Named("lei", c.LEI),
	}
}

// Returns the associated contacts if they are cached on the counterparty, otherwise
// returns an ErrMissingAssociation error if not.
func (c *Counterparty) Contacts() ([]*Contact, error) {
	if c.contacts == nil {
		return nil, errors.ErrMissingAssociation
	}
	return c.contacts, nil
}

// Used by store implementation to cache associated contacts on the counterparty.
func (c *Counterparty) SetContacts(contacts []*Contact) {
	c.contacts = contacts
}

// Lookup an email address in the counterparty contacts to see if it exists.
func (c *Counterparty) HasContact(email string) (bool, error) {
	if c.contacts == nil {
		return false, errors.ErrMissingAssociation
	}

	for _, contact := range c.contacts {
		if contact.Email == email {
			return true, nil
		}
	}

	return false, nil
}

// Returns the Website as a string after it has been parsed with `url.Parse`,
// attempting to detect missing schemas and other parsing errors. This function
// will return an error if Website.String cannot be parsed by url.Parse or if
// Website.Valid is false.
func (c *Counterparty) NormalizedWebsite() (out string, err error) {
	if c.Website.Valid {
		var parsed *url.URL
		if parsed, err = url.Parse(c.Website.String); err != nil {
			return "", err
		}

		// This is a HACK but it's the best we can do right now. When there is no
		// schema, then usually the hostname is put into the schema and there is no
		// Host value, so we check for this and if so we will add the schema and
		// try again
		if parsed.Host == "" {
			if parsed, err = url.Parse("https://" + c.Website.String); err != nil {
				return "", err
			}
		}

		return parsed.String(), nil
	}

	return "", errors.ErrNullString

}

// ###########################################################################
// Contact
// ###########################################################################

type Contact struct {
	Model
	Name           string        // The full name of the contact
	Email          string        // A unique address for the contact (professional email) must be lowercase
	Role           string        // A description of what the contact does at the counterparty
	CounterpartyID ulid.ULID     // Reference to the counterparty the contact is associated with
	counterparty   *Counterparty // Associated counterparty if fetched from the database
}

// Scan a complete SELECT into the counterparty model
func (c *Contact) Scan(scanner Scanner) error {
	return scanner.Scan(
		&c.ID,
		&c.Name,
		&c.Email,
		&c.Role,
		&c.CounterpartyID,
		&c.Created,
		&c.Modified,
	)
}

// Get complete named params of the counterparty from the model.
func (c *Contact) Params() []any {
	return []any{
		sql.Named("id", c.ID),
		sql.Named("name", c.Name),
		sql.Named("email", c.Email),
		sql.Named("role", c.Role),
		sql.Named("counterpartyID", c.CounterpartyID),
		sql.Named("created", c.Created),
		sql.Named("modified", c.Modified),
	}
}

// Returns the associated counterparty if it is cached on the model, otherwise returns
// an ErrMissingAssociation error.
func (c *Contact) Counterparty() (*Counterparty, error) {
	if c.counterparty == nil {
		return nil, errors.ErrMissingAssociation
	}
	return c.counterparty, nil
}

func (c *Contact) SetCounterparty(counterparty *Counterparty) {
	c.counterparty = counterparty
}

// Return the RFC 5322 address as implemented by the net/mail package.
func (c *Contact) Address() *mail.Address {
	return &mail.Address{
		Name:    c.Name,
		Address: c.Email,
	}
}

// ###########################################################################
// CounterpartySourceInfo
// ###########################################################################

type CounterpartySourceInfo struct {
	ID                  ulid.ULID
	Source              string         // either directory or locally created
	DirectoryID         sql.NullString // the directory ID associated with the counterparty (directory only)
	RegisteredDirectory sql.NullString // the registered directory of the counterparty (directory only)
	Protocol            string         // the protocol to use to send travel rule information (TRISA, TRP, Sunrise, etc.)
}

func (c *CounterpartySourceInfo) Scan(scanner Scanner) error {
	return scanner.Scan(
		&c.ID,
		&c.Source,
		&c.DirectoryID,
		&c.RegisteredDirectory,
		&c.Protocol,
	)
}

// ###########################################################################
// VASPCategories
// ###########################################################################

// VASPCategories allows the string list to be stored in the database as a JSON array.
type VASPCategories []string

func (c *VASPCategories) Scan(src interface{}) error {
	// Convert src into a byte array for unmarshaling
	var source []byte
	switch t := src.(type) {
	case []byte:
		source = t
	case string:
		source = []byte(t)
	case nil:
		return nil
	default:
		return fmt.Errorf("incompatible type for vasp categories: %T", t)
	}

	// Unmarshal the JSON string array
	strs := make([]string, 0)
	if err := json.Unmarshal(source, &strs); err != nil {
		return err
	}

	// Convert into the VASP categories type
	*c = VASPCategories(strs)
	return nil
}

func (c VASPCategories) Value() (_ driver.Value, err error) {
	// Store NULL for empty lists
	if len(c) == 0 {
		return nil, nil
	}

	var data []byte
	if data, err = json.Marshal(c); err != nil {
		return nil, err
	}

	return driver.Value(data), nil
}
