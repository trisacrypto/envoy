package sqlite_test

import (
	"fmt"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestListSunrise() {
	s.Run("Success", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()

		//test
		messages, err := s.store.ListSunrise(ctx, &models.PageInfo{})
		require.NoError(err, "expected no error when listing sunrise messages")
		require.NotNil(messages, "expected a non-nil sunrise messages page")
		require.NotNil(messages.Messages, "expected a non-nil sunrise messages list")
		require.Len(messages.Messages, 1, fmt.Sprintf("expected 1 sunrise messages, got %d", len(messages.Messages)))
	})

	s.Run("SuccessNilPageInfo", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()

		//test
		messages, err := s.store.ListSunrise(ctx, nil)
		require.NoError(err, "expected no error when listing sunrise messages")
		require.NotNil(messages, "expected a non-nil sunrise messages page")
		require.NotNil(messages.Messages, "expected a non-nil sunrise messages list")
		require.Len(messages.Messages, 1, fmt.Sprintf("expected 1 sunrise messages, got %d", len(messages.Messages)))
	})
}

func (s *storeTestSuite) TestCreateSunrise_Success() {
	//setup
	ctx := s.ActorContext()
	require := s.Require()
	message := mock.GetSampleSunrise(true)
	message.ID = ulid.Zero
	message.EnvelopeID = uuid.MustParse("17c802fb-0c7d-4288-8a3a-bb49c95b85c7")
	message.Email = "compliance@daybreak.example.com"

	messages, err := s.store.ListSunrise(ctx, &models.PageInfo{})
	require.NoError(err, "expected no error when listing sunrise messages")
	require.NotNil(messages, "expected a non-nil sunrise messages page")
	require.NotNil(messages.Messages, "expected a non-nil sunrise messages list")
	expectedLen := len(messages.Messages) + 1

	//test
	err = s.store.CreateSunrise(ctx, message, &models.ComplianceAuditLog{})
	require.NoError(err, "expected no error when creating sunrise message")

	messages = nil
	messages, err = s.store.ListSunrise(ctx, &models.PageInfo{})
	require.NoError(err, "expected no error when listing sunrise messages")
	require.NotNil(messages, "expected a non-nil sunrise messages page")
	require.NotNil(messages.Messages, "expected a non-nil sunrise messages list")
	require.Len(messages.Messages, expectedLen, fmt.Sprintf("expected %d sunrise messages, got %d", expectedLen, len(messages.Messages)))

	//check for audit log creation
	ok := s.AssertAuditLogCount(map[string]int{
		ActionResourceKey(enum.ActionCreate, enum.ResourceSunrise): 1,
	})
	require.True(ok, "audit log count was off")
}

func (s *storeTestSuite) TestCreateSunriseFailures() {
	s.Run("FailureNonZeroID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		message := mock.GetSampleSunrise(true)
		message.EnvelopeID = uuid.MustParse("17c802fb-0c7d-4288-8a3a-bb49c95b85c7")
		message.Email = "compliance@daybreak.example.com"

		//test
		err := s.store.CreateSunrise(ctx, message, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when creating sunrise message")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected ErrNoIDOnCreate")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})

	s.Run("FailureNotFoundRandomEnvelopeID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		message := mock.GetSampleSunrise(true)
		message.ID = ulid.Zero
		message.EnvelopeID = uuid.New()
		message.Email = "compliance@daybreak.example.com"

		//test
		err := s.store.CreateSunrise(ctx, message, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when creating sunrise message")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})

	s.Run("FailureNotFoundRandomEmail", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		message := mock.GetSampleSunrise(true)
		message.ID = ulid.Zero
		message.EnvelopeID = uuid.MustParse("17c802fb-0c7d-4288-8a3a-bb49c95b85c7")
		message.Email = uuid.NewString() + "@example.com"

		//test
		err := s.store.CreateSunrise(ctx, message, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when creating sunrise message")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})

	s.Run("FailureUniquenessConstraint", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		message := mock.GetSampleSunrise(true)
		message.ID = ulid.Zero
		message.EnvelopeID = uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")
		message.Email = "compliance@daybreak.example.com"

		//test
		err := s.store.CreateSunrise(ctx, message, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when creating sunrise message")
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})
}

func (s *storeTestSuite) TestRetrieveSunrise() {
	s.Run("Success", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")

		//test
		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494"), msg.EnvelopeID, "expected a different envelope ID")
		require.Equal("compliance@daybreak.example.com", msg.Email, "expected email to be 'compliance@daybreak.example.com'")
	})

	s.Run("FailureRandomID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.MakeSecure()

		//test
		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.Error(err, "expected an error when retrieving sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(msg, "expected a nil sunrise msg")
	})

	s.Run("FailureZeroID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.Zero

		//test
		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.Error(err, "expected an error when retrieving sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(msg, "expected a nil sunrise msg")
	})
}

func (s *storeTestSuite) TestUpdateSunrise_Success() {
	//setup
	ctx := s.ActorContext()
	require := s.Require()
	msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
	msg, err := s.store.RetrieveSunrise(ctx, msgId)
	require.NoError(err, "expected no error when retrieving sunrise msg")
	require.NotNil(msg, "expected a non-nil sunrise msg")

	beforeUpdate := time.Now()
	newId := uuid.MustParse("17c802fb-0c7d-4288-8a3a-bb49c95b85c7")
	msg.EnvelopeID = newId
	newEmail := "technical@daybreak.example.com"
	msg.Email = newEmail

	//test
	err = s.store.UpdateSunrise(ctx, msg, &models.ComplianceAuditLog{})
	require.NoError(err, "expected no error when updating sunrise msg")

	msg = nil
	msg, err = s.store.RetrieveSunrise(ctx, msgId)
	require.NoError(err, "expected no error when retrieving sunrise msg")
	require.NotNil(msg, "expected a non-nil sunrise msg")
	require.Equal(newId, msg.EnvelopeID, "expected the new envelope ID")
	require.Equal(newEmail, msg.Email, "expected the new email")
	require.True(beforeUpdate.Before(msg.Modified), "expected a more recent modified timestamp")

	//check for audit log creation
	ok := s.AssertAuditLogCount(map[string]int{
		ActionResourceKey(enum.ActionUpdate, enum.ResourceSunrise): 1,
	})
	require.True(ok, "audit log count was off")
}

func (s *storeTestSuite) TestUpdateSunriseFailures() {
	s.Run("FailureZeroID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")

		prevMod := msg.Modified
		prevId := msg.EnvelopeID
		msg.EnvelopeID = uuid.MustParse("17c802fb-0c7d-4288-8a3a-bb49c95b85c7")
		prevEmail := msg.Email
		msg.Email = "technical@daybreak.example.com"
		msg.ID = ulid.Zero

		//test
		err = s.store.UpdateSunrise(ctx, msg, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when updating sunrise msg")
		require.Equal(errors.ErrMissingID, err, "expected ErrMissingID")

		msg = nil
		msg, err = s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(prevId, msg.EnvelopeID, "expected the same envelope ID")
		require.Equal(prevEmail, msg.Email, "expected the same email")
		require.True(prevMod.Equal(msg.Modified), "expected the same modified timestamp")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")

		prevMod := msg.Modified
		prevId := msg.EnvelopeID
		msg.EnvelopeID = uuid.MustParse("17c802fb-0c7d-4288-8a3a-bb49c95b85c7")
		prevEmail := msg.Email
		msg.Email = "technical@daybreak.example.com"
		msg.ID = ulid.MakeSecure()

		//test
		err = s.store.UpdateSunrise(ctx, msg, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when updating sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		msg = nil
		msg, err = s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(prevId, msg.EnvelopeID, "expected the same envelope ID")
		require.Equal(prevEmail, msg.Email, "expected the same email")
		require.True(prevMod.Equal(msg.Modified), "expected the same modified timestamp")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})
}

func (s *storeTestSuite) TestUpdateSunriseStatusSuccess() {
	//NOTE: separated because it modifies our sunrise fixture
	//setup
	ctx := s.ActorContext()
	require := s.Require()
	msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
	envId := uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")
	beforeUpdate := time.Now()
	newStatus := enum.StatusRejected

	//test
	err := s.store.UpdateSunriseStatus(ctx, envId, newStatus, &models.ComplianceAuditLog{})
	require.NoError(err, "expected no error when updating sunrise status")

	msg, err := s.store.RetrieveSunrise(ctx, msgId)
	require.NoError(err, "expected no error when retrieving sunrise msg")
	require.NotNil(msg, "expected a non-nil sunrise msg")
	require.Equal(newStatus, msg.Status, "expected the new status")
	require.True(beforeUpdate.Before(msg.Modified), "expected a more recent modified timestamp")

	//check for audit log creation
	ok := s.AssertAuditLogCount(map[string]int{
		ActionResourceKey(enum.ActionUpdate, enum.ResourceSunrise): 1,
	})
	require.True(ok, "audit log count was off")
}

func (s *storeTestSuite) TestUpdateSunriseStatusFailures() {
	s.Run("FailureNotFoundNilID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		envId := uuid.Nil
		beforeUpdate := time.Now()
		newStatus := enum.StatusRejected

		//test
		err := s.store.UpdateSunriseStatus(ctx, envId, newStatus, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when updating sunrise status")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(enum.StatusPending, msg.Status, "expected the same status")
		require.False(beforeUpdate.Before(msg.Modified), "expected the same modified timestamp")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		envId := uuid.New()
		beforeUpdate := time.Now()
		newStatus := enum.StatusRejected

		//test
		err := s.store.UpdateSunriseStatus(ctx, envId, newStatus, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when updating sunrise status")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(enum.StatusPending, msg.Status, "expected the same status")
		require.False(beforeUpdate.Before(msg.Modified), "expected the same modified timestamp")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})
}

func (s *storeTestSuite) TestDeleteSunrise_Success() {
	//setup
	ctx := s.ActorContext()
	require := s.Require()
	msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")

	//test
	err := s.store.DeleteSunrise(ctx, msgId, &models.ComplianceAuditLog{})
	require.NoError(err, "expected no error when deleting sunrise msg")

	msg, err := s.store.RetrieveSunrise(ctx, msgId)
	require.Error(err, "expected no error when retrieving sunrise msg")
	require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	require.Nil(msg, "expected a nil sunrise msg")

	//check for audit log creation
	ok := s.AssertAuditLogCount(map[string]int{
		ActionResourceKey(enum.ActionDelete, enum.ResourceSunrise): 1,
	})
	require.True(ok, "audit log count was off")
}

func (s *storeTestSuite) TestDeleteSunrise() {
	s.Run("FailureNotFoundZeroID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.Zero

		//test
		err := s.store.DeleteSunrise(ctx, msgId, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when deleting sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		msgId := ulid.MakeSecure()

		//test
		err := s.store.DeleteSunrise(ctx, msgId, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error when deleting sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		//check for audit log creation
		ok := s.AssertAuditLogCount(map[string]int{})
		require.True(ok, "audit log count was off")
	})
}

func (s *storeTestSuite) TestGetOrCreateSunriseCounterparty_ByContact() {
	testcases := []string{
		"compliance@daybreak.example.com",                          //regular
		"Compliance@Daybreak.Example.Com",                          //caps
		"Compliance Officer <compliance@daybreak.example.com>",     //regular
		"Compliance Officer <Compliance@Daybreak.Example.Com>",     //caps
		"\"Compliance Officer\" <Compliance@Daybreak.Example.Com>", //quoted
	}
	for i, contactEmail := range testcases {
		s.Run(fmt.Sprintf("Email_%d", i), func() {
			//setup
			ctx := s.ActorContext()
			require := s.Require()
			counterpartyName := "Example Daybreak Counterparty"
			counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")

			//test
			counterparty, err := s.store.GetOrCreateSunriseCounterparty(ctx, contactEmail, "bananna", &models.ComplianceAuditLog{})
			require.NoError(err, "expected no errors when getting sunrise counterparty by email")
			require.NotNil(counterparty, "expected a non-nil counterparty")
			require.Equal(counterpartyId, counterparty.ID, "expected a different counterparty ID")
			require.Equal(counterpartyName, counterparty.Name, "expected a different counterparty name")

			addr, err := mail.ParseAddress(contactEmail)
			require.NoError(err, "expected no errors when parsing email")
			ok, err := counterparty.HasContact(strings.ToLower(addr.Address))
			require.NoError(err, "expected no error when finding the contact by email on the counterparty")
			require.True(ok, "expected the counterparty to have a contact with the same email address")

			//check for audit log creation
			ok = s.AssertAuditLogCount(map[string]int{})
			require.True(ok, "audit log count was off")
		})
	}
}

func (s *storeTestSuite) TestGetOrCreateSunriseCounterparty_SuccessFoundCounterpartyName() {
	//setup
	ctx := s.ActorContext()
	require := s.Require()
	contactEmail := "bananna@bananna.example.com"
	counterpartyName := "Example Daybreak Counterparty"
	counterpartyId := ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ")

	//test
	counterparty, err := s.store.GetOrCreateSunriseCounterparty(ctx, contactEmail, counterpartyName, &models.ComplianceAuditLog{})
	require.NoError(err, "expected no errors when getting sunrise counterparty by email")
	require.NotNil(counterparty, "expected a non-nil counterparty")
	require.Equal(counterpartyId, counterparty.ID, "expected a different counterparty")
	require.Equal(counterpartyName, counterparty.Name, "expected a different counterparty name")

	ok, err := counterparty.HasContact(contactEmail)
	require.NoError(err, "expected no error when finding the contact by email on the counterparty")
	require.True(ok, "expected the counterparty to have a contact with the same email address")

	//check for audit log creation
	ok = s.AssertAuditLogCount(map[string]int{
		ActionResourceKey(enum.ActionCreate, enum.ResourceContact): 1,
	})
	require.True(ok, "audit log count was off")
}

func (s *storeTestSuite) TestGetOrCreateSunriseCounterparty_SuccessCreatedNewCounterparty() {
	//setup
	ctx := s.ActorContext()
	require := s.Require()
	contactEmail := "mango@mango.example.com"
	counterpartyName := "Mango Counterparty"

	counterparties, err := s.store.ListCounterparties(ctx, &models.CounterpartyPageInfo{})
	require.NoError(err, "expected no errors listing counterparties")
	require.NotNil(counterparties, "expceted counterparties to be non-nil")
	require.NotNil(counterparties.Counterparties, "expceted counterparties.Counterparties to be non-nil")
	expLen := len(counterparties.Counterparties) + 1

	//test
	counterparty, err := s.store.GetOrCreateSunriseCounterparty(ctx, contactEmail, counterpartyName, &models.ComplianceAuditLog{})
	require.NoError(err, "expected no errors when getting sunrise counterparty by email")
	require.NotNil(counterparty, "expected a non-nil counterparty")
	require.Equal(counterpartyName, counterparty.Name, "expected a different counterparty name")

	ok, err := counterparty.HasContact(contactEmail)
	require.NoError(err, "expected no error when finding the contact by email on the counterparty")
	require.True(ok, "expected the counterparty to have a contact with the same email address")

	counterparties, err = s.store.ListCounterparties(ctx, &models.CounterpartyPageInfo{})
	require.NoError(err, "expected no errors listing counterparties")
	require.NotNil(counterparties, "expceted counterparties to be non-nil")
	require.NotNil(counterparties.Counterparties, "expceted counterparties.Counterparties to be non-nil")
	require.Len(counterparties.Counterparties, expLen, fmt.Sprintf("expected %d counterparties, got %d", expLen, len(counterparties.Counterparties)))

	//check for audit log creation
	ok = s.AssertAuditLogCount(map[string]int{
		ActionResourceKey(enum.ActionCreate, enum.ResourceCounterparty): 1,
		ActionResourceKey(enum.ActionCreate, enum.ResourceContact):      1,
	})
	require.True(ok, "audit log count was off")
}

func (s *storeTestSuite) TestGetOrCreateSunriseCounterpartyFailures() {
	s.Run("FailureUnparseableEmails", func() {
		//setup
		ctx := s.ActorContext()
		require := s.Require()
		testcases := []string{
			// general stuff
			"",                                     //blank
			"user#example.com",                     // missing @ / invalid symbol
			"user[at]example[dot]com",              // weird format
			"user@email@example.com",               // extra @
			"user@example.co.uk \"",                // trailing quote
			"\"First Last\" <username@example.com", // RFC format missing closing bracket
			"\"First Last <username@example.com>",  // RFC format hanging quote
			// invalid chars in RFC name
			"error:achieved <username@example.com>", // unqoted colon
			"2<3 <username@example.com>",            // unqoted less than
			"4>3 <username@example.com>",            // unqoted greater than
			// invalid chars in username
			"user name@example.com",         // space
			"\"user@example.co.uk",          // hanging quote
			"user+pass:invalid@example.com", // colon
			"2<3@example.com",               // less than
			"4>3@example.com",               // greater than
			// invalid chars in domain
			"username@dmain example.com",        // space
			"username@\"domain\".co.uk",         // quote
			"username@pass:invalid.example.com", // colon
			"username@2<3.com",                  // less than
			"username@4>3.com",                  // greater than
		}

		//test
		for _, email := range testcases {
			counterparty, err := s.store.GetOrCreateSunriseCounterparty(ctx, email, "name", &models.ComplianceAuditLog{})
			require.Error(err, fmt.Sprintf("case %s: expected an error for invalid email", email))
			require.ErrorContains(err, "could not parse the provided email address", fmt.Sprintf("case %s: expected an email parsing error", email))
			require.Nil(counterparty, fmt.Sprintf("case %s: expected a nil counterparty", email))

			//check for audit log creation
			ok := s.AssertAuditLogCount(map[string]int{})
			require.True(ok, "audit log count was off")
		}
	})
}
