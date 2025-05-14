package scene

import (
	"encoding/json"
	"strings"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

// IVMS101 is a struct that represents the complex IVMS101 data as a flattened struct
// with only the data that is required by our web application.
type IVMS101 struct {
	Originator      Entities
	OriginatorVASP  Company
	Beneficiary     Entities
	BeneficiaryVASP Company
}

type Entities []Entity

type Entity struct {
	Person  *Person
	Company *Company
}

type Person struct {
	AddressComponents
	Forename            string
	Surname             string
	PrimaryAddress      *ivms101.Address
	PrimaryAddressLines []string
	CustomerNumber      string
	NationalIdentifier  NationalIdentifier
	DateOfBirth         string
	PlaceOfBirth        string
	CountryOfResidence  string
	original            *ivms101.NaturalPerson
}

type Company struct {
	AddressComponents
	LegalName             string
	PrimaryAddress        *ivms101.Address
	PrimaryAddressLines   []string
	CustomerNumber        string
	CountryOfRegistration string
	NationalIdentifier    NationalIdentifier
	original              *ivms101.LegalPerson
}

type NationalIdentifier struct {
	Identifier            string
	TypeRepr              string
	TypeCode              string
	CountryOfIssue        string
	RegistrationAuthority string
}

type AddressComponents struct {
	AddressType    string
	AddressLine0   string
	AddressLine1   string
	AddressLine2   string
	AddressCountry string
}

// Return the simplified/flattened IVMS101 identity representation if an Envelope has
// been set as the APIData in the Scene.
func (s Scene) IVMS101() *IVMS101 {
	var envelope *Envelope
	if envelope = s.Envelope(); envelope == nil {
		return nil
	}
	return NewIVMS101(envelope)
}

func NewIVMS101(envelope *Envelope) *IVMS101 {
	// Create the IVMS101 struct from the envelope data
	ivms := &IVMS101{
		// Originator:      makePerson(envelope.FirstOriginator()),
		OriginatorVASP: makeCompany(envelope.OriginatorVASP()),
		// Beneficiary:     makePerson(envelope.FirstBeneficiary()),
		BeneficiaryVASP: makeCompany(envelope.BeneficiaryVASP()),
	}

	// Create the originator persons
	for _, person := range envelope.Originators() {
		var entity Entity
		switch {
		case person.GetNaturalPerson() != nil:
			person := makePerson(person.GetNaturalPerson())
			entity.Person = &person
		case person.GetLegalPerson() != nil:
			company := makeCompany(person.GetLegalPerson())
			entity.Company = &company
		}

		if entity.Person != nil || entity.Company != nil {
			ivms.Originator = append(ivms.Originator, entity)
		}
	}

	// Create the beneficiary persons
	for _, person := range envelope.Beneficiaries() {
		var entity Entity
		switch {
		case person.GetNaturalPerson() != nil:
			person := makePerson(person.GetNaturalPerson())
			entity.Person = &person
		case person.GetLegalPerson() != nil:
			company := makeCompany(person.GetLegalPerson())
			entity.Company = &company
		}

		if entity.Person != nil || entity.Company != nil {
			ivms.Beneficiary = append(ivms.Beneficiary, entity)
		}
	}

	return ivms
}

func (e Entities) Plural() bool {
	return len(e) > 1
}

func (e Entities) JSON() string {
	data, _ := json.Marshal(e)
	return string(data)
}

func (p Person) FullName() string {
	return strings.TrimSpace(p.Forename + " " + p.Surname)
}

func (p Person) JSON() string {
	if p.original != nil {
		data, _ := json.Marshal(p.original)
		return string(data)
	}
	return ""
}

func (p Person) MarshalJSON() ([]byte, error) {
	if p.original != nil {
		return json.Marshal(p.original)
	}
	return nil, nil
}

func (c Company) JSON() string {
	if c.original != nil {
		data, _ := json.Marshal(c.original)
		return string(data)
	}
	return ""
}

func (c Company) MarshalJSON() ([]byte, error) {
	if c.original != nil {
		return json.Marshal(c.original)
	}
	return nil, nil
}

func (a AddressComponents) AddressLabel() string {
	switch a.AddressType {
	case "HOME":
		return "Home Address"
	case "BIZZ":
		return "Business Address"
	case "GEOG":
		return "Geographic Address"
	case "MISC":
		return "Address"
	default:
		return "Address"
	}
}

func makePerson(person *ivms101.NaturalPerson) (p Person) {
	if person == nil {
		return p
	}

	p.original = person
	p.PrimaryAddress = api.FindPrimaryAddress(person)
	p.PrimaryAddressLines = api.MakeAddressLines(p.PrimaryAddress)
	p.AddressComponents = makeAddressComponents(p.PrimaryAddress)
	p.CustomerNumber = person.CustomerIdentification
	p.NationalIdentifier = makeNationalID(person.NationalIdentification)
	p.CountryOfResidence = person.CountryOfResidence

	// Handle the name of the person
	if nameIdx := api.FindLegalName(person); nameIdx >= 0 {
		name := person.Name.NameIdentifiers[nameIdx]
		p.Surname = name.PrimaryIdentifier
		p.Forename = name.SecondaryIdentifier
	}

	if person.DateAndPlaceOfBirth != nil {
		p.DateOfBirth = person.DateAndPlaceOfBirth.DateOfBirth
		p.PlaceOfBirth = person.DateAndPlaceOfBirth.PlaceOfBirth
	}

	return p
}

func makeCompany(vasp *ivms101.LegalPerson) (v Company) {
	if vasp == nil {
		return v
	}

	v.original = vasp
	v.PrimaryAddress = api.FindPrimaryAddress(vasp)
	v.PrimaryAddressLines = api.MakeAddressLines(v.PrimaryAddress)
	v.AddressComponents = makeAddressComponents(v.PrimaryAddress)
	v.CustomerNumber = vasp.CustomerNumber
	v.CountryOfRegistration = vasp.CountryOfRegistration
	v.NationalIdentifier = makeNationalID(vasp.NationalIdentification)

	if nameIdx := api.FindLegalName(vasp); nameIdx >= 0 {
		name := vasp.Name.NameIdentifiers[nameIdx]
		v.LegalName = name.LegalPersonName
	}

	return v
}

func makeNationalID(id *ivms101.NationalIdentification) (n NationalIdentifier) {
	if id == nil {
		return n
	}

	n.Identifier = id.NationalIdentifier
	n.TypeCode = strings.TrimPrefix(id.NationalIdentifierType.String(), "NATIONAL_IDENTIFIER_TYPE_CODE_")
	n.CountryOfIssue = id.CountryOfIssue
	n.RegistrationAuthority = id.RegistrationAuthority

	switch id.NationalIdentifierType {
	case ivms101.NationalIdentifierARNU:
		n.TypeRepr = "Alien Registration ID"
	case ivms101.NationalIdentifierCCPT:
		n.TypeRepr = "Passport Number"
	case ivms101.NationalIdentifierRAID:
		n.TypeRepr = "Registration Authority ID"
	case ivms101.NationalIdentifierDRLC:
		n.TypeRepr = "Driver's License Number"
	case ivms101.NationalIdentifierFIIN:
		n.TypeRepr = "Foreign Investor ID"
	case ivms101.NationalIdentifierTXID:
		n.TypeRepr = "Tax ID"
	case ivms101.NationalIdentifierSOCS:
		n.TypeRepr = "Social Security Number"
	case ivms101.NationalIdentifierIDCD:
		n.TypeRepr = "State Issued ID"
	case ivms101.NationalIdentifierLEIX:
		n.TypeRepr = "LEI"
	}

	return n
}

func makeAddressComponents(addr *ivms101.Address) (a AddressComponents) {
	if addr == nil {
		return a
	}

	a.AddressType = strings.TrimPrefix(addr.AddressType.String(), "ADDRESS_TYPE_CODE_")
	a.AddressCountry = addr.Country

	if len(addr.AddressLine) > 0 {
		a.AddressLine0 = addr.AddressLine[0]
	}

	if len(addr.AddressLine) > 1 {
		a.AddressLine1 = addr.AddressLine[1]
	}

	if len(addr.AddressLine) > 2 {
		a.AddressLine2 = addr.AddressLine[2]
	}

	return a
}
