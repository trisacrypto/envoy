package scene

import (
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

const defaultLegalNameType = "LEGL"

// Counterparty is a UI-friendly representation of counterparty records that flattens
// the IVMS101 legal person fields in a way that's easier for templates to consume.
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
	HasIVMSRecord       bool
	Created             time.Time
	Modified            time.Time
}

func (s Scene) CounterpartyDetail() *Counterparty {
	if data, ok := s[APIData]; ok {
		switch model := data.(type) {
		case *models.Counterparty:
			return makeCounterpartyFromModel(model)
		case *api.Counterparty:
			return makeCounterpartyFromAPI(model)
		}
	}
	return nil
}

// IsEditable returns true only if the counterparty is user-created and uses the Sunrise protocol,
// matching the UpdateCounterparty endpoint restriction.
func (c Counterparty) IsEditable() bool {
	return strings.EqualFold(c.Source, enum.SourceUserEntry.String()) &&
		strings.EqualFold(c.Protocol, enum.ProtocolSunrise.String())
}

func (c Counterparty) HasTravelAddress() bool {
	return strings.TrimSpace(c.TravelAddress) != ""
}

func makeCounterpartyFromModel(model *models.Counterparty) *Counterparty {
	if model == nil {
		return nil
	}

	out := &Counterparty{
		ID:                  model.ID.String(),
		Source:              model.Source.String(),
		DirectoryID:         model.DirectoryID.String,
		RegisteredDirectory: model.RegisteredDirectory.String,
		Protocol:            model.Protocol.String(),
		CommonName:          model.CommonName,
		Endpoint:            model.Endpoint,
		Name:                model.Name,
		Website:             model.Website.String,
		Country:             model.Country.String,
		BusinessCategory:    model.BusinessCategory.String,
		VASPCategories:      model.VASPCategories,
		HasIVMSRecord:       model.IVMSRecord != nil,
		Created:             model.Created,
		Modified:            model.Modified,
		IVMSRecord: Company{
			LegalName:             model.Name,
			LegalNameType:         defaultLegalNameType,
			CountryOfRegistration: model.Country.String,
		},
	}

	if model.VerifiedOn.Valid {
		verified := model.VerifiedOn.Time
		out.VerifiedOn = &verified
	}

	out.TravelAddress, _ = api.EndpointTravelAddress(model.Endpoint, model.Protocol)
	if model.IVMSRecord != nil {
		out.IVMSRecord = makeCompany(model.IVMSRecord)
	}

	return out
}

func makeCounterpartyFromAPI(model *api.Counterparty) *Counterparty {
	if model == nil {
		return nil
	}

	out := &Counterparty{
		ID:                  model.ID.String(),
		Source:              model.Source,
		DirectoryID:         model.DirectoryID,
		RegisteredDirectory: model.RegisteredDirectory,
		Protocol:            model.Protocol,
		CommonName:          model.CommonName,
		Endpoint:            model.Endpoint,
		TravelAddress:       model.TravelAddress,
		Name:                model.Name,
		Website:             model.Website,
		Country:             model.Country,
		BusinessCategory:    model.BusinessCategory,
		VASPCategories:      model.VASPCategories,
		VerifiedOn:          model.VerifiedOn,
		HasIVMSRecord:       model.IVMSRecord != "",
		Created:             model.Created,
		IVMSRecord: Company{
			LegalName:             model.Name,
			LegalNameType:         defaultLegalNameType,
			CountryOfRegistration: model.Country,
		},
	}

	if model.Modified != nil {
		out.Modified = *model.Modified
	}

	if out.HasIVMSRecord {
		if legalPerson, err := model.IVMS101(); err == nil {
			out.IVMSRecord = makeCompany(legalPerson)
		}
	}

	return out
}
