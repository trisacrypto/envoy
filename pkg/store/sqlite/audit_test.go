package sqlite_test

import (
	"context"
	"fmt"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestListComplianceAuditLogs() {
	s.Run("SuccessNilPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 6, fmt.Sprintf("there should be 6 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessZeroPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, &models.ComplianceAuditLogPageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 6, fmt.Sprintf("there should be 6 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByTime", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		afterTime, err := time.Parse(time.RFC3339, "2024-02-01T00:00:00Z")
		require.NoError(err, "could not parse time string")
		beforeTime, err := time.Parse(time.RFC3339, "2024-06-01T00:00:00Z")
		require.NoError(err, "could not parse time string")
		pageInfo := &models.ComplianceAuditLogPageInfo{
			After:  afterTime,
			Before: beforeTime,
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 4, fmt.Sprintf("there should be 4 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByResourceType", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			ResourceTypes: []string{"transaction", "user", "api_key"},
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 3, fmt.Sprintf("there should be 3 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByResourceIDOnly", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			ResourceID: "2c891c75-14fa-4c71-aa07-6405b98db7a3",
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 1, fmt.Sprintf("there should be 1 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByResourceIDWithOverriddenTypes", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			ResourceID:    "2c891c75-14fa-4c71-aa07-6405b98db7a3",
			ResourceTypes: []string{"sunrise", "user", "api_key"}, //should be overridden
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 1, fmt.Sprintf("there should be 1 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByActorIDWithOverriddenTypes", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			ActorID:    "01JXTGSFRC88HAY8V173976Z9D",
			ActorTypes: []string{"api_key", "user"}, //should be overridden
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 2, fmt.Sprintf("there should be 2 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByActorTypeAndResourceType", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			ResourceTypes: []string{"account", "counterparty"},
			ActorTypes:    []string{"user"},
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 2, fmt.Sprintf("there should be 2 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByActorIDAndResourceType", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			ResourceTypes: []string{"user", "sunrise"},
			ActorID:       "01HWQEJJDMS5EKNARHPJEDMHA4",
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 2, fmt.Sprintf("there should be 2 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByActorTypeAndResourceID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			ResourceID: "01HWQE29RW1S1D8ZN58M528A1M",
			ActorTypes: []string{"api_key", "sunrise"},
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 1, fmt.Sprintf("there should be 1 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByActorTypeAndResourceIDNoResults", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			ResourceID: "01HWQE29RW1S1D8ZN58M528A1M",
			ActorTypes: []string{"user", "sunrise"},
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 0, fmt.Sprintf("there should be 0 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByTimeAndActorIDAndResourceID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		afterTime, err := time.Parse(time.RFC3339, "2024-02-01T00:00:00Z")
		require.NoError(err, "could not parse time string")
		beforeTime, err := time.Parse(time.RFC3339, "2024-06-01T00:00:00Z")
		require.NoError(err, "could not parse time string")
		pageInfo := &models.ComplianceAuditLogPageInfo{
			After:      afterTime,
			Before:     beforeTime,
			ActorID:    "01JXTGSFRC88HAY8V173976Z9D",
			ResourceID: "01HWQEJJDMS5EKNARHPJEDMHA4",
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 1, fmt.Sprintf("there should be 1 logs, but there were %d", len(logs.Logs)))
	})

	s.Run("SuccessFilterByTimeAndActorIDAndResourceIDAndOverriddenTypes", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		afterTime, err := time.Parse(time.RFC3339, "2024-02-01T00:00:00Z")
		require.NoError(err, "could not parse time string")
		beforeTime, err := time.Parse(time.RFC3339, "2024-06-01T00:00:00Z")
		require.NoError(err, "could not parse time string")
		pageInfo := &models.ComplianceAuditLogPageInfo{
			After:         afterTime,
			Before:        beforeTime,
			ActorID:       "01JXTGSFRC88HAY8V173976Z9D",
			ActorTypes:    []string{"user"}, //should be ignored
			ResourceID:    "01HWQEJJDMS5EKNARHPJEDMHA4",
			ResourceTypes: []string{"user"}, //should be ignored
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 1, fmt.Sprintf("there should be 1 logs, but there were %d", len(logs.Logs)))
	})
}

func (s *storeTestSuite) TestCreateComplianceAuditLog() {
	s.Run("SuccessWithMeta", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		log := mock.GetComplianceAuditLog(true, true)
		log.ID = ulid.Zero

		//test
		err := s.store.CreateComplianceAuditLog(ctx, log)
		require.NoError(err, "no error was expected")

		logs, err := s.store.ListComplianceAuditLogs(ctx, &models.ComplianceAuditLogPageInfo{After: log.ResourceModified})
		require.NoError(err, "expected no error")
		require.NotNil(logs, "logs should not be nil")
		require.Len(logs.Logs, 1, fmt.Sprintf("expected 1 log, got %d", len(logs.Logs)))
		require.Equal(log.ID, logs.Logs[0].ID, fmt.Sprintf("log ID should be %s, found %s instead", log.ID, logs.Logs[0].ID))
		require.NotNil(log.Signature, "expected a non-nil log signature")
		// TODO (sc-32721): when signatures are implemented, uncomment below and remove the Error test
		// require.NoError(log.Verify(), "could not verify log signature")
		require.Error(log.Verify(), "log verification is not implemented yet")
	})

	s.Run("SuccessNoMeta", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		log := mock.GetComplianceAuditLog(false, true)
		log.ID = ulid.Zero

		//test
		err := s.store.CreateComplianceAuditLog(ctx, log)
		require.NoError(err, "no error was expected")

		logs, err := s.store.ListComplianceAuditLogs(ctx, &models.ComplianceAuditLogPageInfo{After: log.ResourceModified})
		require.NoError(err, "expected no error")
		require.NotNil(logs, "logs should not be nil")
		require.Len(logs.Logs, 1, fmt.Sprintf("expected 1 log, got %d", len(logs.Logs)))
		require.Equal(log.ID, logs.Logs[0].ID, fmt.Sprintf("log ID should be %s, found %s instead", log.ID, logs.Logs[0].ID))
		require.NotNil(log.Signature, "expected a non-nil log signature")
		// TODO (sc-32721): when signatures are implemented, uncomment below and remove the Error test
		// require.NoError(log.Verify(), "could not verify log signature")
		require.Error(log.Verify(), "log verification is not implemented yet")
	})

	s.Run("FailureNonZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		log := mock.GetComplianceAuditLog(false, true)

		//test
		err := s.store.CreateComplianceAuditLog(ctx, log)
		require.Error(err, "an error was expected")
		require.Equal(err, errors.ErrNoIDOnCreate, "expected ErrNoIDOnCreate")

		logs, err := s.store.ListComplianceAuditLogs(ctx, &models.ComplianceAuditLogPageInfo{After: log.ResourceModified})
		require.NoError(err, "expected no error")
		require.NotNil(logs, "logs should not be nil")
		require.Len(logs.Logs, 0, fmt.Sprintf("expected 0 logs, got %d", len(logs.Logs)))
	})

	s.Run("FailureNoTimestamp", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		log := mock.GetComplianceAuditLog(false, true)
		log.ID = ulid.Zero
		oldTimestamp := log.ResourceModified
		log.ResourceModified = time.Time{}

		//test
		err := s.store.CreateComplianceAuditLog(ctx, log)
		require.Error(err, "an error was expected")
		require.Equal(err, errors.ErrMissingTimestamp, "expected ErrMissingTimestamp")

		logs, err := s.store.ListComplianceAuditLogs(ctx, &models.ComplianceAuditLogPageInfo{After: oldTimestamp})
		require.NoError(err, "expected no error")
		require.NotNil(logs, "logs should not be nil")
		require.Len(logs.Logs, 0, fmt.Sprintf("expected 0 logs, got %d", len(logs.Logs)))
	})
}

func (s *storeTestSuite) TestRetrieveComplianceAuditLogs() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		id := ulid.MustParse("01JZ1HNFJ9KTA3Z6Q4RB3X9W2T")

		//test
		log, err := s.store.RetrieveComplianceAuditLog(ctx, id)
		require.NoError(err, "expected no errors")
		require.NotNil(log, "expected a non-nil log")
		require.True(log.ChangeNotes.Valid, "expected change notes")
		require.Equal("test_user_create_account", log.ChangeNotes.String, "expected a different value")
	})
}
