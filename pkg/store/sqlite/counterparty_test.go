package sqlite_test

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestListCounterparties() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		counterparties, err := s.store.ListCounterparties(ctx, &models.CounterpartyPageInfo{})
		require.NoError(err, "expected no error")
		require.NotNil(counterparties, "expected counterparties to be non-nil")
		require.Len(counterparties.Counterparties, 4, fmt.Sprintf("expected 4 counterparty, got %d", len(counterparties.Counterparties)))
	})

	s.Run("PanicsNilPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		require.Panics(func() { s.store.ListCounterparties(ctx, nil) })
	})
}

func (s *storeTestSuite) TestListCounterpartySourceInfo() {
	s.Run("SuccessUser", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		srcInfo, err := s.store.ListCounterpartySourceInfo(ctx, enum.SourceUserEntry)
		require.NoError(err, "expected no error")
		require.NotNil(srcInfo, "expected source info to be non-nil")
		require.Len(srcInfo, 1, fmt.Sprintf("expected 1 source info, got %d", len(srcInfo)))
	})

	s.Run("SuccessGDS", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		srcInfo, err := s.store.ListCounterpartySourceInfo(ctx, enum.SourceDirectorySync)
		require.NoError(err, "expected no error")
		require.NotNil(srcInfo, "expected source info to be non-nil")
		require.Len(srcInfo, 2, fmt.Sprintf("expected 2 source info, got %d", len(srcInfo)))
	})

	s.Run("SuccessDaybreak", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		srcInfo, err := s.store.ListCounterpartySourceInfo(ctx, enum.SourceDaybreak)
		require.NoError(err, "expected no error")
		require.NotNil(srcInfo, "expected source info to be non-nil")
		require.Len(srcInfo, 1, fmt.Sprintf("expected 1 source info, got %d", len(srcInfo)))
	})

	s.Run("SuccessUnknown", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		srcInfo, err := s.store.ListCounterpartySourceInfo(ctx, enum.SourceUnknown)
		require.NoError(err, "expected no error")
		require.NotNil(srcInfo, "expected source info to be non-nil")
		require.Len(srcInfo, 0, fmt.Sprintf("expected 0 source info, got %d", len(srcInfo)))
	})
}

func (s *storeTestSuite) TestCreateCounterparty() {
	s.Run("SuccessNoContacts", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterparty := mock.GetSampleCounterparty(true, false)
		counterparty.ID = ulid.Zero

		//test
		err := s.store.CreateCounterparty(ctx, counterparty)
		require.NoError(err, "expected no error")

		counterparty, err = s.store.RetrieveCounterparty(ctx, counterparty.ID)
		require.NoError(err, "expected no error")
		require.NotNil(counterparty, "expected counterparty to be non-nil")
	})

	s.Run("SuccessWithContacts", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterparty := mock.GetSampleCounterparty(true, true)
		counterparty.ID = ulid.Zero
		contacts, err := counterparty.Contacts()
		require.NoError(err, "expected no error")
		for _, contact := range contacts {
			contact.ID = ulid.Zero
			contact.CounterpartyID = ulid.Zero
		}

		//test
		err = s.store.CreateCounterparty(ctx, counterparty)
		require.NoError(err, "expected no error")

		counterparty, err = s.store.RetrieveCounterparty(ctx, counterparty.ID)
		require.NoError(err, "expected no error")
		require.NotNil(counterparty, "expected counterparty to be non-nil")
	})

	s.Run("FailureNonZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterparty := mock.GetSampleCounterparty(true, false)

		//test
		err := s.store.CreateCounterparty(ctx, counterparty)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected ErrNoIDOnCreate")
	})

	s.Run("FailureContactEmailDuplicate", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterparty := mock.GetSampleCounterparty(true, true)
		counterparty.ID = ulid.Zero
		contacts, err := counterparty.Contacts()
		require.NoError(err, "expected no error")
		for _, contact := range contacts {
			contact.ID = ulid.Zero
			contact.CounterpartyID = ulid.Zero
			contact.Email = "email@sample.example.com"
		}

		//test
		err = s.store.CreateCounterparty(ctx, counterparty)
		require.Error(err, "expected an error")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// which is correct but it is for the contact, so we should try and
		// still figure out a way to pass that info along somehow if possible
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")
	})
}

func (s *storeTestSuite) TestRetrieveCounterparty() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")

		//test
		counterparty, err := s.store.RetrieveCounterparty(ctx, counterpartyId)
		require.NoError(err, "expected no error")
		require.NotNil(counterparty, "expected counterparty to be non-nil")
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MakeSecure()

		//test
		counterparty, err := s.store.RetrieveCounterparty(ctx, counterpartyId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(counterparty, "expected counterparty to be nil")
	})
}

func (s *storeTestSuite) TestLookupCounterparty() {
	s.Run("SuccessCases", func() {
		type testCase struct {
			Field string
			Value string
		}
		testCases := []testCase{
			{
				Field: "name",
				Value: "Example Daybreak Counterparty",
			},
			{
				Field: "website",
				Value: "https://example.com",
			},
			{
				Field: "endpoint",
				Value: "email:compliance@example.com",
			},
			{
				Field: "directory_id",
				Value: "67e4a151-6607-505f-a6ac-55426aa8a677",
			},
			{
				Field: "lei",
				Value: "01234567889abcdef",
			},
		}

		for _, tc := range testCases {
			s.Run("SuccessByField_"+tc.Field, func() {
				//setup
				require := s.Require()
				ctx := context.Background()

				//test
				counterparty, err := s.store.LookupCounterparty(ctx, tc.Field, tc.Value)
				require.NoError(err, "expected no error")
				require.NotNil(counterparty, "expected counterparty to be non-nil")
			})
		}
	})

	s.Run("FailureCases", func() {
		type testCase struct {
			Field string
			Value string
			Error error
		}
		testCases := []testCase{
			{
				Field: "source",
				Value: "gds",
				Error: errors.ErrAmbiguous,
			},
			{
				Field: "protocol",
				Value: "trisa",
				Error: errors.ErrAmbiguous,
			},
			{
				Field: "business_category",
				Value: "PRIVATE_ORGANIZATION",
				Error: errors.ErrAmbiguous,
			},
			{
				Field: "directory_id",
				Value: uuid.NewString(),
				Error: errors.ErrNotFound,
			},
			{
				Field: "lei",
				Value: uuid.NewString(),
				Error: errors.ErrNotFound,
			},
		}

		for _, tc := range testCases {
			s.Run("FailureByField_"+tc.Field, func() {
				//setup
				require := s.Require()
				ctx := context.Background()

				//test
				counterparty, err := s.store.LookupCounterparty(ctx, tc.Field, tc.Value)
				require.Error(err, "expected an error")
				require.Equal(tc.Error, err, fmt.Sprintf("expected %s", tc.Error))
				require.Nil(counterparty, "expected counterparty to be nil")
			})
		}
	})
}

func (s *storeTestSuite) TestUpdateCounterparty() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")
		counterparty, err := s.store.RetrieveCounterparty(ctx, counterpartyId)
		require.NoError(err, "expected no error")
		require.NotNil(counterparty, "expected counterparty to be non-nil")

		newName := "New Counterparty Name"
		counterparty.Name = newName

		//test
		err = s.store.UpdateCounterparty(ctx, counterparty)
		require.NoError(err, "expected no error")

		counterparty, err = s.store.RetrieveCounterparty(ctx, counterpartyId)
		require.NoError(err, "expected no error")
		require.NotNil(counterparty, "expected counterparty to be non-nil")
		require.Equal(newName, counterparty.Name, "expected the new counterparty name")
	})

	s.Run("FailureZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")
		counterparty, err := s.store.RetrieveCounterparty(ctx, counterpartyId)
		require.NoError(err, "expected no error")
		require.NotNil(counterparty, "expected counterparty to be non-nil")

		counterparty.ID = ulid.Zero

		//test
		err = s.store.UpdateCounterparty(ctx, counterparty)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrMissingID, err, "expected ErrMissingID")
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")
		counterparty, err := s.store.RetrieveCounterparty(ctx, counterpartyId)
		require.NoError(err, "expected no error")
		require.NotNil(counterparty, "expected counterparty to be non-nil")

		counterparty.ID = ulid.MakeSecure()

		//test
		err = s.store.UpdateCounterparty(ctx, counterparty)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestDeleteCounterparty() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")

		//test
		err := s.store.DeleteCounterparty(ctx, counterpartyId)
		require.NoError(err, "expected no error")

		counterparty, err := s.store.RetrieveCounterparty(ctx, counterpartyId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(counterparty, "expected counterparty to be nil")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MakeSecure()

		//test
		err := s.store.DeleteCounterparty(ctx, counterpartyId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureNotFoundZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.Zero

		//test
		err := s.store.DeleteCounterparty(ctx, counterpartyId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestListContacts() {
	s.Run("SuccessByID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")

		//test
		contacts, err := s.store.ListContacts(ctx, counterpartyId, &models.PageInfo{})
		require.NoError(err, "expected no error")
		require.NotNil(contacts, "expected contacts to be non-nil")
		require.Len(contacts.Contacts, 1, fmt.Sprintf("expected 1 contact, got %d", len(contacts.Contacts)))

		require.Equal(counterpartyId, contacts.Contacts[0].CounterpartyID, "unexpected counterparty ID on contact")
		counterparty, err := contacts.Contacts[0].Counterparty()
		require.NoError(err, "expected no error")
		require.Equal(counterpartyId, counterparty.ID, "unexpected counterparty ID on counterparty")
	})

	s.Run("SuccessByCounterparty", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")
		counterparty, err := s.store.RetrieveCounterparty(ctx, counterpartyId)
		require.NoError(err, "expected no error")
		require.NotNil(counterparty, "expected a non-nil counterparty")

		//test
		contacts, err := s.store.ListContacts(ctx, counterparty, &models.PageInfo{})
		require.NoError(err, "expected no error")
		require.NotNil(contacts, "expected contacts to be non-nil")
		require.Len(contacts.Contacts, 1, fmt.Sprintf("expected 1 contact, got %d", len(contacts.Contacts)))

		require.Equal(counterpartyId, contacts.Contacts[0].CounterpartyID, "unexpected counterparty ID on contact")
		counterparty2, err := contacts.Contacts[0].Counterparty()
		require.NoError(err, "expected no error")
		require.Equal(counterpartyId, counterparty2.ID, "unexpected counterparty ID on counterparty")
	})

	s.Run("SuccessNoContacts", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01HWR68SNXH2PZCZX5Y9M5EMC3")

		//test
		contacts, err := s.store.ListContacts(ctx, counterpartyId, &models.PageInfo{})
		require.NoError(err, "expected an error")
		require.NotNil(contacts, "expected contacts to be non-nil")
		require.Len(contacts.Contacts, 0, fmt.Sprintf("expected 0 contact, got %d", len(contacts.Contacts)))
	})

	s.Run("SuccessNilPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01HWR68SNXH2PZCZX5Y9M5EMC3")

		//test
		contacts, err := s.store.ListContacts(ctx, counterpartyId, nil)
		require.NoError(err, "expected an error")
		require.NotNil(contacts, "expected contacts to be non-nil")
		require.Len(contacts.Contacts, 0, fmt.Sprintf("expected 0 contact, got %d", len(contacts.Contacts)))
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MakeSecure()

		//test
		contacts, err := s.store.ListContacts(ctx, counterpartyId, &models.PageInfo{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(contacts, "expected contacts to be nil")
	})

	s.Run("FailureNotFoundZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.Zero

		//test
		contacts, err := s.store.ListContacts(ctx, counterpartyId, &models.PageInfo{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(contacts, "expected contacts to be nil")
	})
}

func (s *storeTestSuite) TestCreateContact() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01HWR68SNXH2PZCZX5Y9M5EMC3")
		contact := mock.GetSampleContact("")
		contact.CounterpartyID = counterpartyId
		contact.ID = ulid.Zero

		//test
		contacts, err := s.store.ListContacts(ctx, counterpartyId, &models.PageInfo{})
		require.NoError(err, "expected no error")
		require.NotNil(contacts, "expected contacts to be non-nil")
		require.Len(contacts.Contacts, 0, fmt.Sprintf("expected 0 contact, got %d", len(contacts.Contacts)))

		err = s.store.CreateContact(ctx, contact)
		require.NoError(err, "expected no error")

		contacts, err = s.store.ListContacts(ctx, counterpartyId, &models.PageInfo{})
		require.NoError(err, "expected no error")
		require.NotNil(contacts, "expected contacts to be non-nil")
		require.Len(contacts.Contacts, 1, fmt.Sprintf("expected 1 contact, got %d", len(contacts.Contacts)))
	})

	s.Run("FailureNonZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		contact := mock.GetSampleContact("")

		//test
		err := s.store.CreateContact(ctx, contact)
		require.Error(err, "expected no error")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected ErrNoIDOnCreate")
	})

	s.Run("FailureCounterpartyNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		contact := mock.GetSampleContact("")
		contact.ID = ulid.Zero

		//test
		err := s.store.CreateContact(ctx, contact)
		require.Error(err, "expected no error")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")
	})

	s.Run("FailureUniqueEmail", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		counterpartyId := ulid.MustParse("01HWR68SNXH2PZCZX5Y9M5EMC3")

		contact := mock.GetSampleContact("")
		contact.ID = ulid.Zero
		contact.CounterpartyID = counterpartyId

		contact2 := mock.GetSampleContact(contact.Email) //same email
		contact2.ID = ulid.Zero
		contact2.CounterpartyID = counterpartyId

		//test
		err := s.store.CreateContact(ctx, contact)
		require.NoError(err, "expected no error")

		err = s.store.CreateContact(ctx, contact2)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")
	})
}

func (s *storeTestSuite) TestRetrieveContact() {
	//TODO
}

func (s *storeTestSuite) TestUpdateContact() {
	//TODO
}

func (s *storeTestSuite) TestDeleteContact() {
	//TODO
}
