package models_test

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

//==========================================================================
// Tests
//==========================================================================

func TestCounterpartyParams(t *testing.T) {
	// setup a model
	theModel := getSampleCounterparty(true, false)

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exception 1) replace "ivms101" as "ivmsrecord"
	exceptions := map[string]string{ConvertNameForComparison("ivms101"): ConvertNameForComparison("IVMSRecord")}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestContactParams(t *testing.T) {
	// setup a model
	theModel := getSampleContact("")

	// create the model public field name comparison list
	fields := GetPublicFieldNames(*theModel)

	// create the `Params()` comparison list
	// Exceptions: None
	exceptions := map[string]string{}
	params := GetParamsNames(theModel, exceptions)

	// test
	require.ElementsMatch(t, fields, params, "the model's public fields and Params() lists should have the same names")
}

func TestCounterpartyContacts(t *testing.T) {
	// test 1: has contacts
	contacts, err := getSampleCounterparty(true, true).Contacts()
	require.NotNil(t, contacts, "contacts should not be nil")
	require.Nil(t, err, "error should be nil")

	//test 2: no contacts
	contacts, err = getSampleCounterparty(true, false).Contacts()
	require.Nil(t, contacts, "contacts should be nil")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")
}
func TestCounterpartyHasContact(t *testing.T) {
	// test 1: has contact
	exists, err := getSampleCounterparty(true, true).HasContact("email@example.com")
	require.True(t, exists, "contact should be present")
	require.Nil(t, err, "error should be nil")

	//test 2: no contact
	exists, err = getSampleCounterparty(true, false).HasContact("email@example.com")
	require.False(t, exists, "there should be no contact")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")

}

//==========================================================================
// Helpers
//==========================================================================

// Returns a sample Counterparty.
func getSampleCounterparty(includeNulls bool, includeContacts bool) (model *models.Counterparty) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.Counterparty{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		Source:              enum.SourceDirectorySync,
		DirectoryID:         sql.NullString{},
		RegisteredDirectory: sql.NullString{},
		Protocol:            enum.ProtocolTRISA,
		CommonName:          "CommonName",
		Endpoint:            "schema://endpoint",
		Name:                "Name",
		Website:             sql.NullString{},
		Country:             sql.NullString{},
		BusinessCategory:    sql.NullString{},
		VASPCategories:      models.VASPCategories{},
		VerifiedOn:          sql.NullTime{},
		IVMSRecord:          nil,
		LEI:                 sql.NullString{},
	}

	if includeNulls {
		model.DirectoryID = sql.NullString{String: "DirectoryID", Valid: true}
		model.RegisteredDirectory = sql.NullString{String: "RegisteredDirectory", Valid: true}
		model.Website = sql.NullString{String: "Website", Valid: true}
		model.Country = sql.NullString{String: "Country", Valid: true}
		model.BusinessCategory = sql.NullString{String: "BusinessCategory", Valid: true}
		model.VerifiedOn = sql.NullTime{Time: timeNow, Valid: true}
		model.LEI = sql.NullString{String: "LEI", Valid: true}
	}

	if includeContacts {
		model.SetContacts([]*models.Contact{getSampleContact("email@example.com"), getSampleContact("")})
	}

	return model
}

// Returns a sample Contact.
func getSampleContact(email string) (model *models.Contact) {
	id := ulid.MakeSecure()
	timeNow := time.Now()
	if email == "" {
		email = fmt.Sprintf("%s@example.com", id.String())
	}

	model = &models.Contact{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		Name:           "Name",
		Email:          email,
		Role:           "Role",
		CounterpartyID: ulid.MakeSecure(),
	}

	return model
}
