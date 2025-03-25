package scene

import (
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
)

// Account is used instead of api.Account here to ensure we don't serialize and
// deserialize the IVMSRecord field in the API response. It does add a bit of duplication
// wrt the API code, but hopefully that will not be too onerous to maintain.
type Account struct {
	ID            string
	CustomerID    string
	FirstName     string
	LastName      string
	TravelAddress string
	IVMSRecord    Person
	HasIVMSRecord bool
	NumAddresses  int64
	Created       time.Time
	Modified      time.Time
}

func (s Scene) AccountDetail() *Account {
	if data, ok := s[APIData]; ok {
		if model, ok := data.(*models.Account); ok {
			account := &Account{
				ID:            model.ID.String(),
				CustomerID:    model.CustomerID.String,
				FirstName:     model.FirstName.String,
				LastName:      model.LastName.String,
				TravelAddress: model.TravelAddress.String,
				HasIVMSRecord: model.HasIVMSRecord(),
				NumAddresses:  model.NumAddresses(),
				Created:       model.Created,
				Modified:      model.Modified,
			}

			if model.IVMSRecord != nil {
				account.IVMSRecord = makePerson(model.IVMSRecord.GetNaturalPerson())
			} else {
				account.IVMSRecord = Person{
					Forename:       model.FirstName.String,
					Surname:        model.LastName.String,
					CustomerNumber: account.CustomerNumber(),
				}
			}

			return account
		}
	}
	return nil
}

func (a Account) CustomerNumber() string {
	if a.CustomerID != "" {
		return a.CustomerID
	}
	return a.ID
}
