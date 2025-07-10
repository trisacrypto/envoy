package web_test

import (
	"context"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"go.rtnl.ai/ulid"
)

func (w *webTestSuite) TestServerListComplianceAuditLogs() {
	w.Run("QueryValidation", func() {
		w.Run("SuccessAllFields", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			after := time.Now().Add(-1 * time.Hour)
			before := after.Add(2 * time.Hour)
			query := &api.ComplianceAuditLogQuery{
				PageQuery: api.PageQuery{
					PageSize:      999,
					NextPageToken: "this can be anything",
					PrevPageToken: "this can be anything",
				},
				ResourceTypes: []string{"transaction", "user", "api_key", "counterparty", "account", "sunrise"},
				ResourceID:    "this can be anything",
				ActorTypes:    []string{"user", "api_key", "sunrise"},
				ActorID:       "this can be anything",
				After:         &after,
				Before:        &before,
				DetailedLogs:  true,
			}
			w.store.OnListComplianceAuditLogs = func(ctx context.Context, page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error) {
				return &models.ComplianceAuditLogPage{
					Page: page,
					Logs: []*models.ComplianceAuditLog{},
				}, nil
			}

			//test
			logs, err := w.ClientWithPermissions(AllPermissions).ListComplianceAuditLogs(ctx, query)
			require.NoError(err, "unexpected client request error")
			require.NotNil(logs, "response object was unexpectedly nil")
			require.Len(logs.Logs, 0, "expected no logs")
		})

		w.Run("FailureAll", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			after := time.Now().Add(1 * time.Hour)
			before := after.Add(-1 * time.Hour)
			query := &api.ComplianceAuditLogQuery{
				PageQuery: api.PageQuery{
					PageSize:      999,
					NextPageToken: "this can be anything",
					PrevPageToken: "this can be anything",
				},
				ResourceTypes: []string{ulid.MakeSecure().String()},
				ResourceID:    "this can be anything",
				ActorTypes:    []string{ulid.MakeSecure().String()},
				ActorID:       "this can be anything",
				After:         &after,
				Before:        &before,
			}

			//test
			logs, err := w.ClientWithPermissions(AllPermissions).ListComplianceAuditLogs(ctx, query)
			require.ErrorContains(err, "5 validation errors occurred", "should have found 5 validation errors")
			require.Nil(logs, "response object should be nil")
		})
	})

	w.Run("Auth", func() {
		w.Run("SuccessTailoredPermissions", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			query := &api.ComplianceAuditLogQuery{}
			w.store.OnListComplianceAuditLogs = func(ctx context.Context, page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error) {
				return &models.ComplianceAuditLogPage{
					Page: &models.ComplianceAuditLogPageInfo{},
					Logs: []*models.ComplianceAuditLog{},
				}, nil
			}
			permissions := []string{
				"users:view",
				"apikeys:view",
				"counterparties:view",
				"accounts:view",
				"travelrule:view",
			}

			//test
			logs, err := w.ClientWithPermissions(permissions).ListComplianceAuditLogs(ctx, query)
			require.NoError(err, "unexpected client request error")
			require.NotNil(logs, "response object was unexpectedly nil")
			require.Len(logs.Logs, 0, "expected no logs")
		})

		w.Run("SuccessAllPermissions", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			query := &api.ComplianceAuditLogQuery{}
			w.store.OnListComplianceAuditLogs = func(ctx context.Context, page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error) {
				return &models.ComplianceAuditLogPage{
					Page: &models.ComplianceAuditLogPageInfo{},
					Logs: []*models.ComplianceAuditLog{},
				}, nil
			}
			permissions := AllPermissions

			//test
			logs, err := w.ClientWithPermissions(permissions).ListComplianceAuditLogs(ctx, query)
			require.NoError(err, "unexpected client request error")
			require.NotNil(logs, "response object was unexpectedly nil")
			require.Len(logs.Logs, 0, "expected no logs")
		})

		w.Run("FailureSomePermissions", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			query := &api.ComplianceAuditLogQuery{}
			w.store.OnListComplianceAuditLogs = func(ctx context.Context, page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error) {
				return &models.ComplianceAuditLogPage{
					Page: &models.ComplianceAuditLogPageInfo{},
					Logs: []*models.ComplianceAuditLog{},
				}, nil
			}
			permissions := []string{
				"users:view",
				"apikeys:view",
				"counterparties:view",
				"accounts:view",
				"travelrule:view",
			}

			for idx := range permissions {
				// remove a permission
				if idx < len(permissions) {
					permissions = append(permissions[:idx], permissions[idx+1:]...)
				} else {
					permissions = permissions[:idx]
				}

				//test
				logs, err := w.ClientWithPermissions(permissions).ListComplianceAuditLogs(ctx, query)
				require.Error(err, "expected a client request error")
				require.ErrorContains(err, "user does not have permission to perform this operation", "the user should not be authorized")
				require.Nil(logs, "expected a nil response object")
			}

		})

		w.Run("FailureNoPermissions", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			query := &api.ComplianceAuditLogQuery{}
			permissions := []string{}

			//test
			logs, err := w.ClientWithPermissions(permissions).ListComplianceAuditLogs(ctx, query)
			require.Error(err, "expected a client request error")
			require.ErrorContains(err, "user does not have permission to perform this operation", "the user should not be authorized")
			require.Nil(logs, "expected a nil response object")
		})

		w.Run("FailureNoAuth", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			query := &api.ComplianceAuditLogQuery{}

			//test
			logs, err := w.ClientNoAuth().ListComplianceAuditLogs(ctx, query)
			require.Error(err, "expected a client request error")
			require.ErrorContains(err, "this endpoint requires authentication", "the user should not be authenticated")
			require.Nil(logs, "expected a nil response object")
		})
	})
}

func (w *webTestSuite) TestServerComplianceAuditLogDetail() {
	w.Run("Auth", func() {
		w.Run("SuccessTailoredPermissions", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			logID := ulid.MakeSecure()
			w.store.OnRetrieveComplianceAuditLog = func(ctx context.Context, id ulid.ULID) (*models.ComplianceAuditLog, error) {
				return mock.GetComplianceAuditLog(true, true), nil
			}
			permissions := []string{
				"users:view",
				"apikeys:view",
				"counterparties:view",
				"accounts:view",
				"travelrule:view",
			}

			//test
			log, err := w.ClientWithPermissions(permissions).ComplianceAuditLogDetail(ctx, logID)
			require.NoError(err, "unexpected client request error")
			require.NotNil(log, "response object was unexpectedly nil")
		})

		w.Run("SuccessAllPermissions", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			logID := ulid.MakeSecure()
			w.store.OnRetrieveComplianceAuditLog = func(ctx context.Context, id ulid.ULID) (*models.ComplianceAuditLog, error) {
				return mock.GetComplianceAuditLog(true, true), nil
			}
			permissions := AllPermissions

			//test
			log, err := w.ClientWithPermissions(permissions).ComplianceAuditLogDetail(ctx, logID)
			require.NoError(err, "unexpected client request error")
			require.NotNil(log, "response object was unexpectedly nil")
		})

		w.Run("FailureSomePermissions", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			logID := ulid.MakeSecure()
			permissions := []string{
				"users:view",
				"apikeys:view",
				"counterparties:view",
				"accounts:view",
				"travelrule:view",
			}

			for idx := range permissions {
				// remove a permission
				if idx < len(permissions) {
					permissions = append(permissions[:idx], permissions[idx+1:]...)
				} else {
					permissions = permissions[:idx]
				}

				//test
				log, err := w.ClientWithPermissions(permissions).ComplianceAuditLogDetail(ctx, logID)
				require.Error(err, "expected a client request error")
				require.ErrorContains(err, "user does not have permission to perform this operation", "the user should not be authorized")
				require.Nil(log, "expected a nil response object")
			}

		})

		w.Run("FailureNoPermissions", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			logID := ulid.MakeSecure()
			permissions := []string{}

			//test
			log, err := w.ClientWithPermissions(permissions).ComplianceAuditLogDetail(ctx, logID)
			require.Error(err, "expected a client request error")
			require.ErrorContains(err, "user does not have permission to perform this operation", "the user should not be authorized")
			require.Nil(log, "expected a nil response object")
		})

		w.Run("FailureNoAuth", func() {
			//setup
			require := w.Require()
			ctx := context.Background()
			logID := ulid.MakeSecure()

			//test
			log, err := w.ClientNoAuth().ComplianceAuditLogDetail(ctx, logID)
			require.Error(err, "expected a client request error")
			require.ErrorContains(err, "this endpoint requires authentication", "the user should not be authenticated")
			require.Nil(log, "expected a nil response object")
		})
	})
}
