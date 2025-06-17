package sqlite_test

import (
	"context"
	"fmt"
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
		ctx := context.Background()
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
		ctx := context.Background()
		require := s.Require()

		//test
		messages, err := s.store.ListSunrise(ctx, nil)
		require.NoError(err, "expected no error when listing sunrise messages")
		require.NotNil(messages, "expected a non-nil sunrise messages page")
		require.NotNil(messages.Messages, "expected a non-nil sunrise messages list")
		require.Len(messages.Messages, 1, fmt.Sprintf("expected 1 sunrise messages, got %d", len(messages.Messages)))
	})
}

func (s *storeTestSuite) TestCreateSunrise() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
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
		err = s.store.CreateSunrise(ctx, message)
		require.NoError(err, "expected no error when creating sunrise message")

		messages = nil
		messages, err = s.store.ListSunrise(ctx, &models.PageInfo{})
		require.NoError(err, "expected no error when listing sunrise messages")
		require.NotNil(messages, "expected a non-nil sunrise messages page")
		require.NotNil(messages.Messages, "expected a non-nil sunrise messages list")
		require.Len(messages.Messages, expectedLen, fmt.Sprintf("expected %d sunrise messages, got %d", expectedLen, len(messages.Messages)))
	})

	s.Run("FailureNonZeroID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		message := mock.GetSampleSunrise(true)
		message.EnvelopeID = uuid.MustParse("17c802fb-0c7d-4288-8a3a-bb49c95b85c7")
		message.Email = "compliance@daybreak.example.com"

		//test
		err := s.store.CreateSunrise(ctx, message)
		require.Error(err, "expected an error when creating sunrise message")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected ErrNoIDOnCreate")
	})

	s.Run("FailureNotFoundRandomEnvelopeID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		message := mock.GetSampleSunrise(true)
		message.ID = ulid.Zero
		message.EnvelopeID = uuid.New()
		message.Email = "compliance@daybreak.example.com"

		//test
		err := s.store.CreateSunrise(ctx, message)
		require.Error(err, "expected an error when creating sunrise message")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")
	})

	s.Run("FailureNotFoundRandomEmail", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		message := mock.GetSampleSunrise(true)
		message.ID = ulid.Zero
		message.EnvelopeID = uuid.MustParse("17c802fb-0c7d-4288-8a3a-bb49c95b85c7")
		message.Email = uuid.NewString() + "@example.com"

		//test
		err := s.store.CreateSunrise(ctx, message)
		require.Error(err, "expected an error when creating sunrise message")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")
	})

	s.Run("FailureUniquenessConstraint", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		message := mock.GetSampleSunrise(true)
		message.ID = ulid.Zero
		message.EnvelopeID = uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")
		message.Email = "compliance@daybreak.example.com"

		//test
		err := s.store.CreateSunrise(ctx, message)
		require.Error(err, "expected an error when creating sunrise message")
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")
	})
}

func (s *storeTestSuite) TestRetrieveSunrise() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
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
		ctx := context.Background()
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
		ctx := context.Background()
		require := s.Require()
		msgId := ulid.Zero

		//test
		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.Error(err, "expected an error when retrieving sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(msg, "expected a nil sunrise msg")
	})
}

func (s *storeTestSuite) TestUpdateSunrise() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
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
		err = s.store.UpdateSunrise(ctx, msg)
		require.NoError(err, "expected no error when updating sunrise msg")

		msg = nil
		msg, err = s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(newId, msg.EnvelopeID, "expected the new envelope ID")
		require.Equal(newEmail, msg.Email, "expected the new email")
		require.True(beforeUpdate.Before(msg.Modified), "expected a more recent modified timestamp")
	})

	s.Run("FailureZeroID", func() {
		//setup
		ctx := context.Background()
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
		err = s.store.UpdateSunrise(ctx, msg)
		require.Error(err, "expected an error when updating sunrise msg")
		require.Equal(errors.ErrMissingID, err, "expected ErrMissingID")

		msg = nil
		msg, err = s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(prevId, msg.EnvelopeID, "expected the same envelope ID")
		require.Equal(prevEmail, msg.Email, "expected the same email")
		require.True(prevMod.Equal(msg.Modified), "expected the same modified timestamp")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := context.Background()
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
		err = s.store.UpdateSunrise(ctx, msg)
		require.Error(err, "expected an error when updating sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		msg = nil
		msg, err = s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(prevId, msg.EnvelopeID, "expected the same envelope ID")
		require.Equal(prevEmail, msg.Email, "expected the same email")
		require.True(prevMod.Equal(msg.Modified), "expected the same modified timestamp")
	})
}

func (s *storeTestSuite) TestUpdateSunriseStatusSuccess() {
	//NOTE: separated because it modifies our sunrise fixture
	//setup
	ctx := context.Background()
	require := s.Require()
	msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
	envId := uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")
	beforeUpdate := time.Now()
	newStatus := enum.StatusRejected

	//test
	err := s.store.UpdateSunriseStatus(ctx, envId, newStatus)
	require.NoError(err, "expected no error when updating sunrise status")

	msg, err := s.store.RetrieveSunrise(ctx, msgId)
	require.NoError(err, "expected no error when retrieving sunrise msg")
	require.NotNil(msg, "expected a non-nil sunrise msg")
	require.Equal(newStatus, msg.Status, "expected the new status")
	require.True(beforeUpdate.Before(msg.Modified), "expected a more recent modified timestamp")
}

func (s *storeTestSuite) TestUpdateSunriseStatusFailures() {
	s.Run("FailureNotFoundNilID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		envId := uuid.Nil
		beforeUpdate := time.Now()
		newStatus := enum.StatusRejected

		//test
		err := s.store.UpdateSunriseStatus(ctx, envId, newStatus)
		require.Error(err, "expected an error when updating sunrise status")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(enum.StatusPending, msg.Status, "expected the same status")
		require.False(beforeUpdate.Before(msg.Modified), "expected the same modified timestamp")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")
		envId := uuid.New()
		beforeUpdate := time.Now()
		newStatus := enum.StatusRejected

		//test
		err := s.store.UpdateSunriseStatus(ctx, envId, newStatus)
		require.Error(err, "expected an error when updating sunrise status")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.NoError(err, "expected no error when retrieving sunrise msg")
		require.NotNil(msg, "expected a non-nil sunrise msg")
		require.Equal(enum.StatusPending, msg.Status, "expected the same status")
		require.False(beforeUpdate.Before(msg.Modified), "expected the same modified timestamp")
	})
}

func (s *storeTestSuite) TestDeleteSunrise() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		msgId := ulid.MustParse("01JXTGSFRC88HAY8V173976Z9D")

		//test
		err := s.store.DeleteSunrise(ctx, msgId)
		require.NoError(err, "expected no error when deleting sunrise msg")

		msg, err := s.store.RetrieveSunrise(ctx, msgId)
		require.Error(err, "expected no error when retrieving sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(msg, "expected a nil sunrise msg")
	})

	s.Run("FailureNotFoundZeroID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		msgId := ulid.Zero

		//test
		err := s.store.DeleteSunrise(ctx, msgId)
		require.Error(err, "expected an error when deleting sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		msgId := ulid.MakeSecure()

		//test
		err := s.store.DeleteSunrise(ctx, msgId)
		require.Error(err, "expected an error when deleting sunrise msg")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestGetOrCreateSunriseCounterparty() {
	//TODO
	s.T().SkipNow()
}
