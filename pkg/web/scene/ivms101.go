package scene

import (
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
	FullLegalName      string
	PrimaryAddress     []string
	CustomerNumber     string
	NationalIdentifier NationalIdentifier
	DateOfBirth        string
	PlaceOfBirth       string
	CountryOfResidence string
}

type VASP struct {
	LegalName             string
	PrimaryAddress        []string
	CustomerNumber        string
	CountryOfRegistration string
	NationalIdentifier    NationalIdentifier
}

type NationalIdentifier struct {
	Identifier            string
	TypeCode              string
	CountryOfIssue        string
	RegistrationAuthority string
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

	p.FullLegalName = api.FindLegalName(person)
	p.PrimaryAddress = api.FindPrimaryAddress(person)
	p.CustomerNumber = person.CustomerIdentification
	p.NationalIdentifier = makeNationalID(person.NationalIdentification)
	p.CountryOfResidence = person.CountryOfResidence

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

	v.LegalName = api.FindLegalName(vasp)
	v.PrimaryAddress = api.FindPrimaryAddress(vasp)
	v.CustomerNumber = vasp.CustomerNumber
	v.CountryOfRegistration = vasp.CountryOfRegistration
	v.NationalIdentifier = makeNationalID(vasp.NationalIdentification)

	return v
}

func makeNationalID(id *ivms101.NationalIdentification) (n NationalIdentifier) {
	if id == nil {
		return n
	}

	n.Identifier = id.NationalIdentifier
	n.CountryOfIssue = id.CountryOfIssue
	n.RegistrationAuthority = id.RegistrationAuthority

	switch id.NationalIdentifierType {
	case ivms101.NationalIdentifierARNU:
		n.TypeCode = "Alien Registration ID"
	case ivms101.NationalIdentifierCCPT:
		n.TypeCode = "Passport Number"
	case ivms101.NationalIdentifierRAID:
		n.TypeCode = "Registration Authority ID"
	case ivms101.NationalIdentifierDRLC:
		n.TypeCode = "Driver's License Number"
	case ivms101.NationalIdentifierFIIN:
		n.TypeCode = "Foreign Investor ID"
	case ivms101.NationalIdentifierTXID:
		n.TypeCode = "Tax ID"
	case ivms101.NationalIdentifierSOCS:
		n.TypeCode = "Social Security Number"
	case ivms101.NationalIdentifierIDCD:
		n.TypeCode = "State Issued ID"
	case ivms101.NationalIdentifierLEIX:
		n.TypeCode = "LEI"
	}

	return n
}
