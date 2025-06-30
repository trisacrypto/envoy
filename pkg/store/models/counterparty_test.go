package models_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

//==========================================================================
// Tests
//==========================================================================

func TestCounterpartyParams(t *testing.T) {
	// setup a model
	theModel := mock.GetSampleCounterparty(true, false)

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
	theModel := mock.GetSampleContact("")

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
	contacts, err := mock.GetSampleCounterparty(true, true).Contacts()
	require.NotNil(t, contacts, "contacts should not be nil")
	require.Nil(t, err, "error should be nil")

	//test 2: no contacts
	contacts, err = mock.GetSampleCounterparty(true, false).Contacts()
	require.Nil(t, contacts, "contacts should be nil")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")
}
func TestCounterpartyHasContact(t *testing.T) {
	// test 1: has contact
	exists, err := mock.GetSampleCounterparty(true, true).HasContact("CA6DF33A-26D9-4E0E-B5F7-66D1964BE696@example.com")
	require.True(t, exists, "contact should be present")
	require.Nil(t, err, "error should be nil")

	//test 2: no contact
	exists, err = mock.GetSampleCounterparty(true, false).HasContact("email@example.com")
	require.False(t, exists, "there should be no contact")
	require.Error(t, err, "error should not be nil")
	require.Equal(t, errors.ErrMissingAssociation, err, "error should be ErrMissingAssociation")

}

func TestCounterpartyScan(t *testing.T) {
	t.Run("SuccessFilled", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(),          // ID
			"gds",                               // Source
			"DirectoryID",                       // DirectoryID
			"RegisteredDirectory",               // RegisteredDirectory
			"trisa",                             // Protocol
			"CommonName",                        // CommonName
			"Endpoint",                          // Endpoint
			"Name",                              // Name
			"Website",                           // Website
			"Country",                           // Country
			"BusinessCategory",                  // BusinessCategory
			"[\"Category One\",\"Category 2\"]", // VASPCategories
			time.Now(),                          // VerifiedOn
			nil,                                 // IVMSRecord (ignored as null for now)
			time.Now(),                          // Created
			time.Now(),                          // Modified
			"LEI",                               // LEI
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Counterparty{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1) // IVMSRecord will not scan

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Source.String(), "expected field Source to match data[1]")
		require.Equal(t, data[2], model.DirectoryID.String, "expected field DirectoryID to match data[2]")
		require.Equal(t, data[3], model.RegisteredDirectory.String, "expected field RegisteredDirectory to match data[3]")
		require.Equal(t, data[4], model.Protocol.String(), "expected field Protocol to match data[4]")
		require.Equal(t, data[5], model.CommonName, "expected field CommonName to match data[5]")
		require.Equal(t, data[6], model.Endpoint, "expected field Endpoint to match data[6]")
		require.Equal(t, data[7], model.Name, "expected field Name to match data[7]")
		require.Equal(t, data[8], model.Website.String, "expected field Website to match data[8]")
		require.Equal(t, data[9], model.Country.String, "expected field Country to match data[9]")
		require.Equal(t, data[10], model.BusinessCategory.String, "expected field BusinessCategory to match data[10]")
		vaspCategoriesExp := models.VASPCategories(models.VASPCategories{"Category One", "Category 2"}) // special construction for expected value
		require.Equal(t, vaspCategoriesExp, model.VASPCategories, "expected field VASPCategories to match data[11]")
		require.Equal(t, data[12], model.VerifiedOn.Time, "expected field VerifiedOn to match data[12]")
		require.Equal(t, data[14], model.Created, "expected field Created to match data[14]")
		require.Equal(t, data[15], model.Modified, "expected field Modified to match data[15]")
		require.Equal(t, data[16], model.LEI.String, "expected field LEI to match data[16]")
	})

	t.Run("SuccessNulls", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			"gds",                      // Source
			nil,                        // DirectoryID (testing null)
			nil,                        // RegisteredDirectory (testing null)
			"trisa",                    // Protocol
			"CommonName",               // CommonName
			"Endpoint",                 // Endpoint
			"Name",                     // Name
			nil,                        // Website (testing null)
			nil,                        // Country (testing null)
			nil,                        // BusinessCategory (testing null)
			nil,                        // VASPCategories (testing null)
			nil,                        // VerifiedOn (testing null)
			nil,                        // IVMSRecord (ignored as null for now)
			time.Now(),                 // Created
			time.Time{},                // Modified (testing zero value)
			nil,                        // LEI (testing null)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Counterparty{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data)-1) // IVMSRecord will not scan
	})

	t.Run("FailureProtocol", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(),    // ID
			"gds",                         // Source
			nil,                           // DirectoryID (testing null)
			nil,                           // RegisteredDirectory (testing null)
			"not_a_protocol_name_8943879", // Protocol (will fail)
			"CommonName",                  // CommonName
			"Endpoint",                    // Endpoint
			"Name",                        // Name
			nil,                           // Website (testing null)
			nil,                           // Country (testing null)
			nil,                           // BusinessCategory (testing null)
			nil,                           // VASPCategories (testing null)
			nil,                           // VerifiedOn (testing null)
			nil,                           // IVMSRecord (ignored as null for now)
			time.Now(),                    // Created
			time.Time{},                   // Modified (testing zero value)
			nil,                           // LEI (testing null)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Counterparty{}
		err := model.Scan(mockScanner)
		require.Error(t, err, "expected an error from the scanner")
		require.ErrorContains(t, err, "invalid protocol", "expected an 'invalid protocol' error from the scanner")
	})

	t.Run("FailureSource", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			"not_a_source_9083452",     // Source (will fail)
			nil,                        // DirectoryID (testing null)
			nil,                        // RegisteredDirectory (testing null)
			"trisa",                    // Protocol
			"CommonName",               // CommonName
			"Endpoint",                 // Endpoint
			"Name",                     // Name
			nil,                        // Website (testing null)
			nil,                        // Country (testing null)
			nil,                        // BusinessCategory (testing null)
			nil,                        // VASPCategories (testing null)
			nil,                        // VerifiedOn (testing null)
			nil,                        // IVMSRecord (ignored as null for now)
			time.Now(),                 // Created
			time.Time{},                // Modified (testing zero value)
			nil,                        // LEI (testing null)
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Counterparty{}
		err := model.Scan(mockScanner)
		require.Error(t, err, "expected an error from the scanner")
		require.ErrorContains(t, err, "invalid source", "expected an 'invalid source' error from the scanner")
	})
}

func TestContactScan(t *testing.T) {
	t.Run("Success", func(t *testing.T) {
		// setup
		data := []any{
			ulid.MakeSecure().String(), // ID
			"Name",                     // Name
			"Email",                    // Email
			"Role",                     // Role
			ulid.MakeSecure().String(), // CounterpartyID
			time.Now(),                 // Created
			time.Now(),                 // Modified
		}
		mockScanner := &mock.MockScanner{}
		mockScanner.SetData(data)

		// test
		model := &models.Contact{}
		err := model.Scan(mockScanner)
		require.NoError(t, err, "expected no errors from the scanner")
		mockScanner.AssertScanned(t, len(data))

		// make sure scanned data matches the fields they were supposed to scan into
		require.Equal(t, data[0], model.ID.String(), "expected field ID to match data[0]")
		require.Equal(t, data[1], model.Name, "expected field Name to match data[1]")
		require.Equal(t, data[2], model.Email, "expected field Email to match data[2]")
		require.Equal(t, data[3], model.Role, "expected field Role to match data[3]")
		require.Equal(t, data[4], model.CounterpartyID.String(), "expected field CounterpartyID to match data[4]")
		require.Equal(t, data[5], model.Created, "expected field Created to match data[5]")
		require.Equal(t, data[6], model.Modified, "expected field Modified to match data[6]")
	})
}

func TestNormalizedWebsite(t *testing.T) {
	type SuccessCase struct {
		Input    string
		Expected string
	}
	testcases := []SuccessCase{
		// missing schemas
		{
			Input:    "example.com",
			Expected: "https://example.com",
		},
		{
			Input:    "www.example.com",
			Expected: "https://www.example.com",
		},
		{
			Input:    "localhost",
			Expected: "https://localhost",
		},
		{
			Input:    "127.0.0.1",
			Expected: "https://127.0.0.1",
		},
		{
			Input:    "username:password@the-weird-one.even_longer.example.website:12345/path/to/something/long?param1=123&secondparam=astringthistime",
			Expected: "https://username:password@the-weird-one.even_longer.example.website:12345/path/to/something/long?param1=123&secondparam=astringthistime",
		},
		// has schemas
		{
			Input:    "http://example.com",
			Expected: "http://example.com",
		},
		{
			Input:    "http://www.example.com",
			Expected: "http://www.example.com",
		},
		{
			Input:    "http://localhost",
			Expected: "http://localhost",
		},
		{
			Input:    "http://127.0.0.1",
			Expected: "http://127.0.0.1",
		},
		{
			Input:    "http://[fd18:74e8:d33f:ffff::]",
			Expected: "http://[fd18:74e8:d33f:ffff::]",
		},
		{
			Input:    "gopher://username:password@the-weird-one.even_longer.example.website:12345/path/to/something/long?param1=123&secondparam=astringthistime",
			Expected: "gopher://username:password@the-weird-one.even_longer.example.website:12345/path/to/something/long?param1=123&secondparam=astringthistime",
		},
	}

	for idx, tc := range testcases {
		t.Run(fmt.Sprintf("SuccessCase%d", idx), func(t *testing.T) {
			//setup
			counterparty := &models.Counterparty{}
			counterparty.Website.String = tc.Input
			if tc.Input != "" {
				counterparty.Website.Valid = true
			}

			//test
			website, err := counterparty.NormalizedWebsite()
			require.NoError(t, err, "expected no error")
			require.Equal(t, tc.Expected, website, "expected '%s', got '%s'")
		})
	}

	t.Run("FailureIPV6NoSchema", func(t *testing.T) {
		//setup
		counterparty := &models.Counterparty{}
		counterparty.Website.String = "[fd18:74e8:d33f:ffff::]"
		counterparty.Website.Valid = true

		//test
		website, err := counterparty.NormalizedWebsite()
		require.Error(t, err, "expected an error")
		require.ErrorContains(t, err, "first path segment in URL cannot contain colon", "expected error to contain 'first path segment in URL cannot contain colon'")
		require.Equal(t, website, "", "expected the empty string")
	})

	t.Run("ParsingFailureNull", func(t *testing.T) {
		//setup
		counterparty := &models.Counterparty{}
		counterparty.Website.String = "\x00"
		counterparty.Website.Valid = true

		//test
		website, err := counterparty.NormalizedWebsite()
		require.Error(t, err, "expected an error")
		require.ErrorContains(t, err, "invalid control character in URL", "expected error to contain 'invalid control character in URL'")
		require.Equal(t, website, "", "expected the empty string")
	})

	t.Run("ParsingFailureHTTPSNull", func(t *testing.T) {
		//setup
		counterparty := &models.Counterparty{}
		counterparty.Website.String = "https://\x00"
		counterparty.Website.Valid = true

		//test
		website, err := counterparty.NormalizedWebsite()
		require.Error(t, err, "expected an error")
		require.ErrorContains(t, err, "invalid control character in URL", "expected error to contain 'invalid control character in URL'")
		require.Equal(t, website, "", "expected the empty string")
	})

	t.Run("InvalidString", func(t *testing.T) {
		//setup
		counterparty := &models.Counterparty{}

		//test
		website, err := counterparty.NormalizedWebsite()
		require.Error(t, err, "expected an error")
		require.Equal(t, err, errors.ErrInvalidString, "expected ErrInvalidString")
		require.Equal(t, website, "", "expected the empty string")
	})
}
