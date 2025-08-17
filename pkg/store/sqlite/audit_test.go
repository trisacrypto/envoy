package sqlite_test

import (
	"context"
	"fmt"
	"time"

	"github.com/trisacrypto/envoy/pkg/audit"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestListComplianceAuditLogsViews() {
	s.Run("SuccessSummaryNilPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 6, fmt.Sprintf("there should be 6 logs, but there were %d", len(logs.Logs)))
		for idx, log := range logs.Logs {
			require.Falsef(log.ChangeNotes.Valid, "%d: expected an invalid change notes string", idx)
			require.Nilf(log.Signature, "%d: expected a nil signature", idx)
			require.Equalf("", log.KeyID, "%d: expected a blank key id", idx)
			require.Equalf("", log.Algorithm, "%d: expected a blank algorithm", idx)
		}
	})

	s.Run("SuccessSummaryZeroPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, &models.ComplianceAuditLogPageInfo{})
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 6, fmt.Sprintf("there should be 6 logs, but there were %d", len(logs.Logs)))
		for idx, log := range logs.Logs {
			require.Falsef(log.ChangeNotes.Valid, "%d: expected an invalid change notes string", idx)
			require.Nilf(log.Signature, "%d: expected a nil signature", idx)
			require.Equalf("", log.KeyID, "%d: expected a blank key id", idx)
			require.Equalf("", log.Algorithm, "%d: expected a blank algorithm", idx)
		}
	})

	s.Run("SuccessDetailedLogs", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		pageInfo := &models.ComplianceAuditLogPageInfo{
			DetailedLogs: true,
		}

		//test
		logs, err := s.store.ListComplianceAuditLogs(ctx, pageInfo)
		require.NoError(err, "expected no errors")
		require.NotNil(logs.Logs, "there were no logs")
		require.Len(logs.Logs, 6, fmt.Sprintf("there should be 6 logs, but there were %d", len(logs.Logs)))
		for idx, log := range logs.Logs {
			require.Truef(log.ChangeNotes.Valid, "%d: expected a valid change notes NullString", idx)
			require.NotEqualf("", log.ChangeNotes.String, "%d: expected change notes", idx)
			require.NotNilf(log.Signature, "%d: expected a non-nil signature", idx)
			require.NotEqualf("", log.KeyID, "%d: expected a key id", idx)
			require.NotEqualf("", log.Algorithm, "%d: expected an algorithm", idx)
		}
	})
}

func (s *storeTestSuite) TestListComplianceAuditLogsFilters() {
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
	s.Run("SuccessWithChangeNotes", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		log := mock.GetComplianceAuditLog(true, true)
		log.ID = ulid.Zero

		//test
		err := s.store.CreateComplianceAuditLog(ctx, log)
		require.NoError(err, "no error was expected")

		log2, err := s.store.RetrieveComplianceAuditLog(ctx, log.ID)
		require.NoError(err, "expected no error")
		require.NotNil(log2, "log2 should not be nil")
		require.Equal(log.Data(), log2.Data(), "log data doesn't match")
		require.NotNil(log2.Signature, "expected a non-nil log signature")
		require.NoError(audit.Verify(log2), "could not verify log signature")
	})

	s.Run("SuccessNoChangeNotes", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		log := mock.GetComplianceAuditLog(false, true)
		log.ID = ulid.Zero

		//test
		err := s.store.CreateComplianceAuditLog(ctx, log)
		require.NoError(err, "no error was expected")

		log2, err := s.store.RetrieveComplianceAuditLog(ctx, log.ID)
		require.NoError(err, "expected no error")
		require.NotNil(log2, "log2 should not be nil")
		require.Equal(log.Data(), log2.Data(), "log data doesn't match")
		require.NotNil(log2.Signature, "expected a non-nil log signature")
		require.NoError(audit.Verify(log2), "could not verify log signature")
	})

	// ActorID and ActorType are defaulted to "unknown" if they are not set
	// before a transaction begins that requires and audit log, so they must be
	// successful.
	s.Run("SuccessUnknownActor", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		log := mock.GetComplianceAuditLog(false, true)
		log.ID = ulid.Zero

		log.ActorID = []byte("unknown")
		log.ActorType = enum.ActorUnknown

		//test
		err := s.store.CreateComplianceAuditLog(ctx, log)
		require.NoError(err, "no error was expected")

		log2, err := s.store.RetrieveComplianceAuditLog(ctx, log.ID)
		require.NoError(err, "expected no error")
		require.NotNil(log2, "log2 should not be nil")
		require.Equal(log.Data(), log2.Data(), "log data doesn't match")
		require.NotNil(log2.Signature, "expected a non-nil log signature")
		require.NoError(audit.Verify(log2), "could not verify log signature")
	})

	s.Run("FailureNonZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		log := mock.GetComplianceAuditLog(false, true)

		//test
		err := s.store.CreateComplianceAuditLog(ctx, log)
		require.ErrorIs(err, errors.ErrNoIDOnCreate, "expected ErrNoIDOnCreate")

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
		require.ErrorIs(err, errors.ErrMissingTimestamp, "expected ErrMissingTimestamp")

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

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		id := ulid.MustParse("01JZ1HNFJ9KTA3Z6Q4R0ABCXYZ")

		//test
		log, err := s.store.RetrieveComplianceAuditLog(ctx, id)
		require.ErrorIsf(err, errors.ErrNotFound, "expected ErrNotFound, got %s", err)
		require.Nil(log, "expected a nil log")
	})
}
