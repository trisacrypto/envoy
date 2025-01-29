package scene

import (
	"strings"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

// IVMS101 is a struct that represents the complex IVMS101 data as a flattened struct
// with only the data that is required by our web application.
type IVMS101 struct {
	Originator      Person
	OriginatorVASP  VASP
	Beneficiary     Person
	BeneficiaryVASP VASP
}

type Person struct {
	AddressComponents
	FullLegalName       string
	PrimaryName         string
	SecondaryName       string
	PrimaryAddress      *ivms101.Address
	PrimaryAddressLines []string
	CustomerNumber      string
	NationalIdentifier  NationalIdentifier
	DateOfBirth         string
	PlaceOfBirth        string
	CountryOfResidence  string
}

type VASP struct {
	AddressComponents
	LegalName             string
	PrimaryAddress        *ivms101.Address
	PrimaryAddressLines   []string
	CustomerNumber        string
	CountryOfRegistration string
	NationalIdentifier    NationalIdentifier
}

type NationalIdentifier struct {
	Identifier            string
	TypeRepr              string
	TypeCode              string
	CountryOfIssue        string
	RegistrationAuthority string
}

type AddressComponents struct {
	AddressLine1 string
	AddressLine2 string
	City         string
	Region       string
	PostCode     string
}

// Return the simplified/flattened IVMS101 identity representation if an Envelope has
// been set as the APIData in the Scene.
func (s Scene) IVMS101() *IVMS101 {
	var envelope *api.Envelope
	if envelope = s.Envelope(); envelope == nil {
		return nil
	}

	// Create the IVMS101 struct from the envelope data
	ivms := &IVMS101{
		Originator:      makePerson(envelope.FirstOriginator()),
		OriginatorVASP:  makeVASP(envelope.OriginatorVASP()),
		Beneficiary:     makePerson(envelope.FirstBeneficiary()),
		BeneficiaryVASP: makeVASP(envelope.BeneficiaryVASP()),
	}

	return ivms
}

func makePerson(person *ivms101.NaturalPerson) (p Person) {
	if person == nil {
		return p
	}

	p.PrimaryAddress = api.FindPrimaryAddress(person)
	p.PrimaryAddressLines = api.MakeAddressLines(p.PrimaryAddress)
	p.AddressComponents = makeAddressComponents(p.PrimaryAddress)
	p.CustomerNumber = person.CustomerIdentification
	p.NationalIdentifier = makeNationalID(person.NationalIdentification)
	p.CountryOfResidence = person.CountryOfResidence

	// Handle the name of the person
	if nameIdx := api.FindLegalName(person); nameIdx >= 0 {
		name := person.Name.NameIdentifiers[nameIdx]
		p.PrimaryName = name.PrimaryIdentifier
		p.SecondaryName = name.SecondaryIdentifier
		p.FullLegalName = name.SecondaryIdentifier + " " + name.PrimaryIdentifier
	}

	if person.DateAndPlaceOfBirth != nil {
		p.DateOfBirth = person.DateAndPlaceOfBirth.DateOfBirth
		p.PlaceOfBirth = person.DateAndPlaceOfBirth.PlaceOfBirth
	}

	return p
}

func makeVASP(vasp *ivms101.LegalPerson) (v VASP) {
	if vasp == nil {
		return v
	}

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
	n.TypeCode = id.NationalIdentifierType.String()
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

	a.City = addr.TownName
	a.Region = addr.CountrySubDivision
	a.PostCode = addr.PostCode

	if len(addr.AddressLine) > 0 {
		a.AddressLine1 = addr.AddressLine[0]
	}

	if len(addr.AddressLine) > 1 {
		a.AddressLine2 = addr.AddressLine[1]
	}

	// Attempt to parse the city, state, postal code from line 3 if it exists
	if len(addr.AddressLine) > 2 {
		parts := strings.Split(addr.AddressLine[len(addr.AddressLine)-1], ",")
		if len(parts) > 0 && a.City == "" {
			a.City = strings.TrimSpace(parts[0])
		}

		if len(parts) > 1 && a.Region == "" {
			a.Region = strings.TrimSpace(parts[1])
		}

		if len(parts) > 2 && a.PostCode == "" {
			a.PostCode = strings.TrimSpace(parts[2])
		}
	}

	return a
}
