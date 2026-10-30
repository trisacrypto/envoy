package scene

import (
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
)

// Counterparty is used instead of api.Counterparty to ensure we don't serialize and
// deserialize the IVMSRecord field in the API response.
type Counterparty struct {
	ID                  string
	Source              string
	DirectoryID         string
	RegisteredDirectory string
	Protocol            string
	CommonName          string
	Endpoint            string
	TravelAddress       string
	Name                string
	Website             string
	Country             string
	BusinessCategory    string
	VASPCategories      []string
	VerifiedOn          *time.Time
	IVMSRecord          Company
	Created             time.Time
	Modified            time.Time
}

func (s Scene) CounterpartyDetail() *Counterparty {
	if data, ok := s[APIData]; ok {
		if model, ok := data.(*models.Counterparty); ok {
			counterparty := &Counterparty{
				ID:                  model.ID.String(),
				Source:              model.Source.String(),
				DirectoryID:         model.DirectoryID.String,
				RegisteredDirectory: model.RegisteredDirectory.String,
				Protocol:            model.Protocol.String(),
				CommonName:          model.CommonName,
				Endpoint:            model.Endpoint,
				TravelAddress:       model.TravelAddress.String,
				Name:                model.Name,
				Website:             model.Website.String,
				Country:             model.Country.String,
				BusinessCategory:    model.BusinessCategory.String,
				VASPCategories:      []string(model.VASPCategories),
				Created:             model.Created,
				Modified:            model.Modified,
			}

			if model.VerifiedOn.Valid {
				counterparty.VerifiedOn = &model.VerifiedOn.Time
			}

			if model.IVMSRecord != nil {
				counterparty.IVMSRecord = makeCompany(model.IVMSRecord)
				if counterparty.IVMSRecord.LegalNameType == "" {
					counterparty.IVMSRecord.LegalNameType = "LEGL"
				}
			} else {
				counterparty.IVMSRecord = Company{
					LegalName:             model.Name,
					LegalNameType:         "LEGL",
					CountryOfRegistration: model.Country.String,
					CustomerNumber:        model.ID.String(),
				}
			}

			return counterparty
		}
	}
	return nil
}
