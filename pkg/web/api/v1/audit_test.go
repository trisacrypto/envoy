package api_test

import (
	"errors"
	"reflect"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"go.rtnl.ai/ulid"
)

// ###########################################################################
// ComplianceAuditLog Tests
// ###########################################################################

func TestNewComplianceAuditLog(t *testing.T) {
	t.Run("SuccessSummaryInfo", func(t *testing.T) {
		// setup
		model := mock.GetComplianceAuditLog(false, false)

		// test
		log := api.NewComplianceAuditLog(model)
		require.NotNil(t, log, "expected a non-nil log")
		require.NoError(t, compareComplianceAuditLogs(log, model), "the model log did not match the api log")
	})

	t.Run("SuccessSignedNoChangeNotes", func(t *testing.T) {
		// setup
		model := mock.GetComplianceAuditLog(false, true)

		// test
		log := api.NewComplianceAuditLog(model)
		require.NotNil(t, log, "expected a non-nil log")
		require.NoError(t, compareComplianceAuditLogs(log, model), "the model log did not match the api log")
	})

	t.Run("SuccessAllInfo", func(t *testing.T) {
		// setup
		model := mock.GetComplianceAuditLog(true, true)

		// test
		log := api.NewComplianceAuditLog(model)
		require.NotNil(t, log, "expected a non-nil log")
		require.NoError(t, compareComplianceAuditLogs(log, model), "the model log did not match the api log")
	})
}

// ###########################################################################
// ComplianceAuditLogQuery Tests
// ###########################################################################

func TestComplianceAuditLogQueryQuery(t *testing.T) {
	t.Run("SuccessZero", func(t *testing.T) {
		//setup
		model := api.ComplianceAuditLogQuery{}

		//test
		query := model.Query()
		require.NotNil(t, query, "expected query to be non-nil")
		require.NoError(t, compareComplianceAuditLogQueries(&model, query))
	})

	t.Run("SuccessAllOptions", func(t *testing.T) {
		//setup
		after := time.Now().Add((-1 * time.Hour))
		before := time.Now().Add((1 * time.Hour))
		model := api.ComplianceAuditLogQuery{
			ResourceTypes: []string{"transaction", "user", "api_key", "counterparty", "account", "sunrise"},
			ResourceID:    ulid.MakeSecure().String(),
			ActorTypes:    []string{"user", "api_key", "sunrise"},
			ActorID:       uuid.NewString(),
			DetailedLogs:  true,
			After:         &after,
			Before:        &before,
		}

		//test
		query := model.Query()
		require.NotNil(t, query, "expected query to be non-nil")
		require.NoError(t, compareComplianceAuditLogQueries(&model, query))
	})

	t.Run("SuccessEmptyLists", func(t *testing.T) {
		//setup
		model := api.ComplianceAuditLogQuery{
			ResourceTypes: []string{},
			ActorTypes:    []string{},
		}

		//test
		query := model.Query()
		require.NotNil(t, query, "expected query to be non-nil")
		require.NoError(t, compareComplianceAuditLogQueries(&model, query))
	})
}

func TestComplianceAuditLogQueryValidate(t *testing.T) {
	t.Run("SuccessZero", func(t *testing.T) {
		//setup
		model := api.ComplianceAuditLogQuery{}

		//test
		err := model.Validate()
		require.NoError(t, err, "expected log to be valid")
	})

	t.Run("SuccessAllOptions", func(t *testing.T) {
		//setup
		after := time.Now().Add((-1 * time.Hour))
		before := time.Now().Add((1 * time.Hour))
		model := api.ComplianceAuditLogQuery{
			ResourceTypes: []string{"transaction", "user", "api_key", "counterparty", "account", "sunrise"},
			ResourceID:    ulid.MakeSecure().String(),
			ActorTypes:    []string{"user", "api_key", "sunrise"},
			ActorID:       uuid.NewString(),
			DetailedLogs:  true,
			After:         &after,
			Before:        &before,
		}

		//test
		err := model.Validate()
		require.NoError(t, err, "expected log to be valid")
	})

	t.Run("SuccessFutureAfter", func(t *testing.T) {
		//setup
		after := time.Now().Add((1 * time.Hour))
		model := api.ComplianceAuditLogQuery{
			After: &after,
		}

		//test
		err := model.Validate()
		require.NoError(t, err, "should be able to have an 'After' date in the future")
	})

	t.Run("FailureIncorrectResourceTypes", func(t *testing.T) {
		//setup
		model := api.ComplianceAuditLogQuery{
			ResourceTypes: []string{ulid.MakeSecure().String()},
		}

		//test
		err := model.Validate()
		require.ErrorContains(t, err, "invalid field resource_types: invalid resource_types value", "validation failed")
	})

	t.Run("FailureIncorrectActorTypes", func(t *testing.T) {
		//setup
		model := api.ComplianceAuditLogQuery{
			ActorTypes: []string{ulid.MakeSecure().String()},
		}

		//test
		err := model.Validate()
		require.ErrorContains(t, err, "invalid field actor_types: invalid actor_types value", "validation failed")
	})

	t.Run("FailureBeforeBeforeAfter", func(t *testing.T) {
		//setup
		after := time.Now().Add((-1 * time.Hour))
		before := time.Now().Add((-2 * time.Hour))
		model := api.ComplianceAuditLogQuery{
			After:  &after,
			Before: &before,
		}

		//test
		err := model.Validate()
		require.ErrorContains(t, err, "2 validation errors occurred", "wrong number of errors")
		require.ErrorContains(t, err, "invalid field before: before must come before after", "validation failed")
		require.ErrorContains(t, err, "invalid field after: after must come after before", "validation failed")
	})
}

// ###########################################################################
// ComplianceAuditLogList Tests
// ###########################################################################

func TestNewComplianceAuditLogList(t *testing.T) {
	//setup
	after := time.Now().Add((-1 * time.Hour))
	before := time.Now().Add((1 * time.Hour))
	page := &models.ComplianceAuditLogPage{
		Logs: []*models.ComplianceAuditLog{
			mock.GetComplianceAuditLog(true, true),
			mock.GetComplianceAuditLog(true, true),
			mock.GetComplianceAuditLog(true, true),
		},
		Page: &models.ComplianceAuditLogPageInfo{
			ResourceTypes: []string{"transaction", "user", "api_key", "counterparty", "account", "sunrise"},
			ResourceID:    ulid.MakeSecure().String(),
			ActorTypes:    []string{"user", "api_key", "sunrise"},
			ActorID:       uuid.NewString(),
			DetailedLogs:  true,
			After:         after,
			Before:        before,
		},
	}

	//test
	list, err := api.NewComplianceAuditLogList(page)
	require.NoError(t, err, "could not make a new list from page")
	require.NotNil(t, list, "expected a non-nil list")
	require.NoError(t, compareComplianceAuditLogLists(list, page), "list and page did not match")
}

// ###########################################################################
// Helpers
// ###########################################################################

// Compares an api.ComplianceAuditLog to a model.ComplianceAuditLog and returns
// an error if any field does not match (fails at the first unmatched field).
func compareComplianceAuditLogs(apiLog *api.ComplianceAuditLog, modelLog *models.ComplianceAuditLog) (err error) {
	if apiLog.ID != modelLog.ID {
		return errors.New("ID did not match")
	}

	if apiLog.ActorID != string(modelLog.ActorID) {
		return errors.New("ActorID did not match")
	}

	if apiLog.ActorType != modelLog.ActorType.String() {
		return errors.New("ActorType did not match")
	}

	if apiLog.ResourceID != string(modelLog.ResourceID) {
		return errors.New("ResourceID did not match")
	}

	if apiLog.ResourceType != modelLog.ResourceType.String() {
		return errors.New("ResourceType did not match")
	}

	if apiLog.ResourceModified != modelLog.ResourceModified {
		return errors.New("ResourceModified did not match")
	}

	if apiLog.Action != modelLog.Action.String() {
		return errors.New("Action did not match")
	}

	if apiLog.Signature != string(modelLog.Signature) {
		return errors.New("Signature did not match")
	}

	if apiLog.KeyID != modelLog.KeyID {
		return errors.New("KeyID did not match")
	}

	if apiLog.Algorithm != modelLog.Algorithm {
		return errors.New("Algorithm did not match")
	}

	if modelLog.ChangeNotes.Valid {
		if apiLog.ChangeNotes != modelLog.ChangeNotes.String {
			return errors.New("ChangeNotes did not match")
		}
	}

	return nil
}

// Compares an api.ComplianceAuditLogQuery to a models.ComplianceAuditLogPageInfo
// and returns an error if any field does not match (fails at the first unmatched
// field).
func compareComplianceAuditLogQueries(apiLogQuery *api.ComplianceAuditLogQuery, modelLogPageInfo *models.ComplianceAuditLogPageInfo) (err error) {
	if apiLogQuery.PageSize != int(modelLogPageInfo.PageSize) {
		return errors.New("PageSize did not match")
	}

	if apiLogQuery.ResourceTypes != nil && !reflect.DeepEqual(apiLogQuery.ResourceTypes, modelLogPageInfo.ResourceTypes) {
		return errors.New("ResourceTypes did not match")
	}

	if apiLogQuery.ResourceID != modelLogPageInfo.ResourceID {
		return errors.New("ResourceID did not match")
	}

	if apiLogQuery.ActorTypes != nil && !reflect.DeepEqual(apiLogQuery.ActorTypes, modelLogPageInfo.ActorTypes) {
		return errors.New("ActorTypes did not match")
	}

	if apiLogQuery.ActorID != modelLogPageInfo.ActorID {
		return errors.New("ActorID did not match")
	}

	if apiLogQuery.After != nil && !apiLogQuery.After.Equal(modelLogPageInfo.After) {
		return errors.New("After did not match")
	}

	if apiLogQuery.Before != nil && !apiLogQuery.Before.Equal(modelLogPageInfo.Before) {
		return errors.New("Before did not match")
	}

	if apiLogQuery.DetailedLogs != modelLogPageInfo.DetailedLogs {
		return errors.New("DetailedLogs did not match")
	}

	return nil
}

// Compares an api.ComplianceAuditLogList to a models.ComplianceAuditLogPage
// and returns an error if any field does not match (fails at the first unmatched
// field).
func compareComplianceAuditLogLists(apiLogList *api.ComplianceAuditLogList, modelLogPage *models.ComplianceAuditLogPage) (err error) {
	if err := compareComplianceAuditLogQueries(apiLogList.Page, modelLogPage.Page); err != nil {
		return err
	}

	if len(apiLogList.Logs) != len(modelLogPage.Logs) {
		return errors.New("number of logs are not equal")
	}

	for idx, apiLog := range apiLogList.Logs {
		if err := compareComplianceAuditLogs(apiLog, modelLogPage.Logs[idx]); err != nil {
			return err
		}
	}

	return nil
}
