package models

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"

	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

const (
	SourceDirectorySync = "gds"
	SourceUserEntry     = "user"
	ProtocolTRISA       = "trisa"
	ProtocolTRP         = "trp"
)

// TODO: how to incorporate the TRIXO form into this model?
// TODO: make sure that we add contacts associated with this model.
type Counterparty struct {
	Model
	Source              string               // either directory or locally created
	DirectoryID         sql.NullString       // the directory ID associated with the counterparty (directory only)
	RegisteredDirectory sql.NullString       // the registered directory of the counterparty (directory only)
	Protocol            string               // either TRISA or TRP; the protocol to use to send travel rule information
	CommonName          string               // common name - a unique name to identify the endpoint
	Endpoint            string               // the full endpoint to connect to the counterparty on
	Name                string               // the counterparty's legal entity name
	Website             sql.NullString       // website with more information about the counterparty
	Country             string               // country the counterparty is associated with
	BusinessCategory    sql.NullString       // the business category of the counterparty
	VASPCategories      VASPCategories       // the categories of how the VASP handles crypto assets
	VerifiedOn          sql.NullTime         // the datetime the VASP was verified in the directory (directory only)
	IVMSRecord          *ivms101.LegalPerson // IVMS101 record for the counterparty
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
	}
}

type CounterpartySourceInfo struct {
	ID                  ulid.ULID
	Source              string         // either directory or locally created
	DirectoryID         sql.NullString // the directory ID associated with the counterparty (directory only)
	RegisteredDirectory sql.NullString // the registered directory of the counterparty (directory only)
	Protocol            string         // either TRISA or TRP; the protocol to use to send travel rule information
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

// VASPCategories allows the string list to be stored in the database as a JSON array.
type VASPCategories []string

func (c *VASPCategories) Scan(src interface{}) error {
	// Convert src into a byte array for unmarshaling
	var source []byte
	switch t := src.(type) {
	case []byte:
		source = t
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
