package api

import (
	"net/mail"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"go.rtnl.ai/ulid"
	"google.golang.org/protobuf/types/known/anypb"
)

type Prepare struct {
	Routing     *Routing  `json:"routing"`
	Originator  *Person   `json:"originator"`
	Beneficiary *Person   `json:"beneficiary"`
	Transfer    *Transfer `json:"transfer"`
}

type Prepared struct {
	Routing     *Routing                 `json:"routing"`
	Identity    *ivms101.IdentityPayload `json:"identity"`
	Transaction *generic.Transaction     `json:"transaction"`
}

type Routing struct {
	Protocol       string    `json:"protocol"`
	TravelAddress  string    `json:"travel_address,omitempty"`
	CounterpartyID ulid.ULID `json:"counterparty_id,omitempty"`
	Counterparty   string    `json:"counterparty,omitempty"`
	EmailAddress   string    `json:"email,omitempty"`
}

type Person struct {
	CryptoAddress  string          `json:"crypto_address"`
	Forename       string          `json:"forename"`
	Surname        string          `json:"surname"`
	ResidesIn      string          `json:"country_of_residence"`
	CustomerID     string          `json:"customer_id"`
	Identification *Identification `json:"identification"`
	Addresses      []*Address      `json:"addresses"`
}

type Address struct {
	AddressType  string   `json:"address_type"`
	AddressLines []string `json:"address_lines"`
	Country      string   `json:"country"`
}

type Identification struct {
	TypeCode    string `json:"type_code"`
	Number      string `json:"number"`
	Country     string `json:"country"`
	DateOfBirth string `json:"dob"`
	BirthPlace  string `json:"birth_place"`
}

type Transfer struct {
	Amount    float64 `json:"amount"`
	Network   string  `json:"network"`
	AssetType string  `json:"asset_type"`
	TxID      string  `json:"transaction_id"`
	Tag       string  `json:"tag"`
}

//===========================================================================
// Validation
//===========================================================================

func (r *Routing) Validate() (err error) {
	// NOTE: this check must be first because of the use of the shadowed err variable.
	var protocol enum.Protocol
	if protocol, err = enum.ParseProtocol(r.Protocol); err != nil || protocol == enum.ProtocolUnknown {
		err = ValidationError(nil, IncorrectField("routing.protocol", "unknown protocol"))
	}

	// Trim whitespace in strings
	r.TravelAddress = strings.TrimSpace(r.TravelAddress)
	r.EmailAddress = strings.TrimSpace(r.EmailAddress)
	r.Counterparty = strings.TrimSpace(r.Counterparty)

	switch protocol {
	case enum.ProtocolTRISA:
		// For TRISA either the travel address or the counterparty ID must be set
		if r.TravelAddress == "" && r.CounterpartyID.IsZero() {
			err = ValidationError(err, OneOfMissing("routing.travel_address", "routing.counterparty_id"))
		}

		if r.TravelAddress != "" && !r.CounterpartyID.IsZero() {
			err = ValidationError(err, OneOfTooMany("routing.travel_address", "routing.counterparty_id"))
		}

		if r.Counterparty != "" {
			err = ValidationError(err, IncorrectField("routing.counterparty", "not used for trisa protocol"))
		}

		if r.EmailAddress != "" {
			err = ValidationError(err, IncorrectField("routing.email", "not used for trisa protocol"))
		}
	case enum.ProtocolTRP:
		// For TRP the travel address is required
		if r.TravelAddress == "" {
			err = ValidationError(err, MissingField("routing.travel_address"))
		}

		if !r.CounterpartyID.IsZero() {
			err = ValidationError(err, IncorrectField("routing.counterparty_id", "not used for trp protocol"))
		}

		if r.Counterparty != "" {
			err = ValidationError(err, IncorrectField("routing.counterparty", "not used for trp protocol"))
		}

		if r.EmailAddress != "" {
			err = ValidationError(err, IncorrectField("routing.email", "not used for trp protocol"))
		}
	case enum.ProtocolSunrise:
		// For Sunrise the email address or counterparty ID is required
		if r.EmailAddress == "" && r.CounterpartyID.IsZero() {
			err = ValidationError(err, OneOfMissing("routing.email", "routing.counterparty_id"))
		}

		if r.EmailAddress != "" && !r.CounterpartyID.IsZero() {
			err = ValidationError(err, OneOfTooMany("routing.email", "routing.counterparty_id"))
		}

		// Validate the email address can be parsed correctly
		if r.EmailAddress != "" {
			if _, perr := mail.ParseAddress(r.EmailAddress); perr != nil {
				err = ValidationError(err, IncorrectField("routing.email", perr.Error()))
			}
		}

		if r.TravelAddress != "" {
			err = ValidationError(err, IncorrectField("routing.travel_address", "not used for sunrise protocol"))
		}

	}

	return err
}

func (p *Prepare) Validate() (err error) {
	if p.Routing == nil {
		err = ValidationError(err, MissingField("routing"))
	} else {
		if verr := p.Routing.Validate(); verr != nil {
			err = ValidationError(err, verr.(ValidationErrors)...)
		}
	}

	if p.Originator == nil {
		err = ValidationError(err, MissingField("originator"))
	} else {
		if p.Originator.CryptoAddress == "" {
			err = ValidationError(err, MissingField("originator.crypto_address"))
		}

		if p.Originator.Identification == nil {
			p.Originator.Identification = &Identification{}
		}
	}

	if p.Beneficiary == nil {
		err = ValidationError(err, MissingField("beneficiary"))
	} else {
		if p.Beneficiary.CryptoAddress == "" {
			err = ValidationError(err, MissingField("beneficiary.crypto_address"))
		}

		if p.Beneficiary.Identification == nil {
			p.Beneficiary.Identification = &Identification{}
		}
	}

	if p.Transfer == nil {
		err = ValidationError(err, MissingField("transfer"))
	}

	return err
}

func (p *Prepared) Validate() (err error) {
	if p.Routing == nil {
		err = ValidationError(err, MissingField("routing"))
	} else {
		if err = p.Routing.Validate(); err != nil {
			err = ValidationError(err, err.(ValidationErrors)...)
		}
	}

	if p.Identity == nil {
		err = ValidationError(err, MissingField("identity"))
	}

	if p.Transaction == nil {
		err = ValidationError(err, MissingField("transaction"))
	}

	return err
}

//===========================================================================
// Data Handling
//===========================================================================

func (p *Prepare) Transaction() *generic.Transaction {
	return &generic.Transaction{
		Txid:        p.Transfer.TxID,
		Originator:  p.Originator.CryptoAddress,
		Beneficiary: p.Beneficiary.CryptoAddress,
		Amount:      p.Transfer.Amount,
		Network:     p.Transfer.Network,
		AssetType:   p.Transfer.AssetType,
		Tag:         p.Transfer.Tag,
		Timestamp:   "",
		ExtraJson:   "",
	}
}

func (p *Person) FullName() string {
	return strings.TrimSpace(p.Forename + " " + p.Surname)
}

func (p *Person) NaturalPerson() *ivms101.Person {
	// Clean up the address lines in the addresses
	addresses := []*ivms101.Address{}
	for _, addr := range p.Addresses {
		address := &ivms101.Address{
			AddressLine: make([]string, 0, len(addr.AddressLines)),
			Country:     addr.Country,
		}

		var err error
		if address.AddressType, err = ivms101.ParseAddressTypeCode(addr.AddressType); err != nil {
			address.AddressType = ivms101.AddressTypeMisc
		}

		for _, line := range addr.AddressLines {
			line = strings.TrimSpace(line)
			if line != "" && line != "," {
				address.AddressLine = append(address.AddressLine, line)
			}
		}

		addresses = append(addresses, address)
	}

	// NOTE: the country of the address is assigned country of residence
	return &ivms101.Person{
		Person: &ivms101.Person_NaturalPerson{
			NaturalPerson: &ivms101.NaturalPerson{
				Name: &ivms101.NaturalPersonName{
					NameIdentifiers: []*ivms101.NaturalPersonNameId{
						{
							PrimaryIdentifier:   p.Surname,
							SecondaryIdentifier: p.Forename,
							NameIdentifierType:  ivms101.NaturalPersonLegal,
						},
					},
					LocalNameIdentifiers:    nil,
					PhoneticNameIdentifiers: nil,
				},
				GeographicAddresses: addresses,
				NationalIdentification: &ivms101.NationalIdentification{
					NationalIdentifierType: p.Identification.NationalIdentifierTypeCode(),
					NationalIdentifier:     p.Identification.Number,
					CountryOfIssue:         p.Identification.Country,
				},
				CustomerIdentification: p.CustomerID,
				DateAndPlaceOfBirth: &ivms101.DateAndPlaceOfBirth{
					DateOfBirth:  p.Identification.DateOfBirth,
					PlaceOfBirth: p.Identification.BirthPlace,
				},
				CountryOfResidence: p.ResidesIn,
			},
		},
	}
}

func (p *Prepared) Payload() (payload *trisa.Payload, err error) {
	payload = &trisa.Payload{}

	if payload.Identity, err = anypb.New(p.Identity); err != nil {
		return nil, err
	}

	if payload.Transaction, err = anypb.New(p.Transaction); err != nil {
		return nil, err
	}

	payload.SentAt = time.Now().UTC().Format(time.RFC3339)
	return payload, nil
}

func (i *Identification) NationalIdentifierTypeCode() ivms101.NationalIdentifierTypeCode {
	i.TypeCode = strings.TrimSpace(i.TypeCode)
	if tc, err := ivms101.ParseNationalIdentifierTypeCode(i.TypeCode); err == nil {
		return tc
	}
	return ivms101.NationalIdentifierMISC
}
