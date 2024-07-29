package postman

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	api "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
)

func TransactionFromPayload(in *api.Payload) *models.Transaction {
	var (
		err                error
		originator         string
		originatorAddress  string
		beneficiary        string
		beneficiaryAddress string
		virtualAsset       string
		amount             float64
	)

	data := &generic.Transaction{}
	if err = in.Transaction.UnmarshalTo(data); err == nil {
		switch {
		case data.Network != "" && data.AssetType != "":
			virtualAsset = fmt.Sprintf("%s (%s)", data.Network, data.AssetType)
		case data.Network != "":
			virtualAsset = data.Network
		case data.AssetType != "":
			virtualAsset = data.AssetType
		}

		amount = data.Amount
		originatorAddress = data.Originator
		beneficiaryAddress = data.Beneficiary
	}

	identity := &ivms101.IdentityPayload{}
	if err = in.Identity.UnmarshalTo(identity); err == nil {
		if identity.Originator != nil {
			originator = FindName(identity.Originator.OriginatorPersons...)
		}

		if identity.Beneficiary != nil {
			beneficiary = FindName(identity.Beneficiary.BeneficiaryPersons...)
		}

		if originatorAddress == "" {
			originatorAddress = FindAccount(identity.Originator)
		}

		if beneficiaryAddress == "" {
			beneficiaryAddress = FindAccount(identity.Beneficiary)
		}
	}

	return &models.Transaction{
		Originator:         sql.NullString{Valid: originator != "", String: originator},
		OriginatorAddress:  sql.NullString{Valid: originatorAddress != "", String: originatorAddress},
		Beneficiary:        sql.NullString{Valid: beneficiary != "", String: beneficiary},
		BeneficiaryAddress: sql.NullString{Valid: beneficiaryAddress != "", String: beneficiaryAddress},
		VirtualAsset:       virtualAsset,
		Amount:             amount,
	}
}

func FindName(persons ...*ivms101.Person) (name string) {
	// Search all persons for the first legal name available. Use the last available
	// non-zero name for any other name identifier types.
	for _, person := range persons {
		switch t := person.Person.(type) {
		case *ivms101.Person_LegalPerson:
			if t.LegalPerson.Name != nil {
				for _, identifier := range t.LegalPerson.Name.NameIdentifiers {
					// Set the name found to the current legal person name
					if identifier.LegalPersonName != "" {
						name = identifier.LegalPersonName

						// If this is the legal name, short circuit and return it.
						if identifier.LegalPersonNameIdentifierType == ivms101.LegalPersonLegal {
							return name
						}
					}
				}
			}
		case *ivms101.Person_NaturalPerson:
			if t.NaturalPerson.Name != nil {
				for _, identifier := range t.NaturalPerson.Name.NameIdentifiers {
					// Set the name found to the current natural person name
					if identifier.PrimaryIdentifier != "" {
						name = strings.TrimSpace(fmt.Sprintf("%s %s", identifier.SecondaryIdentifier, identifier.PrimaryIdentifier))

						// If this is the legal name of the person, short circuit and return it.
						if identifier.NameIdentifierType == ivms101.NaturalPersonLegal {
							return name
						}
					}
				}
			}
		}

	}

	// Return whatever non-zero name we found, or empty string if we found nothing.
	return name
}

func FindAccount(person any) (account string) {
	if person == nil {
		return ""
	}

	switch t := person.(type) {
	case *ivms101.Originator:
		for _, account = range t.AccountNumbers {
			if account != "" {
				return account
			}
		}
	case *ivms101.Beneficiary:
		for _, account = range t.AccountNumbers {
			if account != "" {
				return account
			}
		}
	}

	// Return whatever non-zero account we found, or empty string if we found nothing.
	return account
}
