package sqlite

import (
	"context"
	"database/sql"
	"strings"

	"github.com/google/uuid"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

// #####################################################
// # ComplianceAuditLogStore implementation for SQLite #
// #####################################################

func (s *Store) ListComplianceAuditLogs(ctx context.Context, page *models.ComplianceAuditLogPageInfo) (out *models.ComplianceAuditLogPage, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if out, err = tx.ListComplianceAuditLogs(page); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return out, nil
}

func (s *Store) CreateComplianceAuditLog(ctx context.Context, log *models.ComplianceAuditLog) (err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return err
	}
	defer tx.Rollback()

	if err = tx.CreateComplianceAuditLog(log); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

// ###################################################
// # ComplianceAuditLogTxn implementation for SQLite #
// ###################################################

const listComplianceAuditLogsSQL = "SELECT id, timestamp, actor_id, actor_type, resource_id, resource_type, action, resource_action_meta, signature FROM compliance_audit_log ORDER BY timestamp DESC"

func (t *Tx) ListComplianceAuditLogs(page *models.ComplianceAuditLogPageInfo) (out *models.ComplianceAuditLogPage, err error) {

	// Setup out variable with page info
	out = &models.ComplianceAuditLogPage{
		Logs: make([]*models.ComplianceAuditLog, 0),
		// TODO: implement pagination
		Page: models.ComplianceAuditLogPageInfoFrom(page),
	}

	// Setup base query and lists for filters/params
	query := listComplianceAuditLogsSQL
	params := make([]any, 0, 4)
	filters := make([]string, 0, 4)

	// After timestamp (inclusive)
	if !out.Page.After.IsZero() {
		filters = append(filters, ":after <= timestamp")
		params = append(params, sql.Named("after", out.Page.After))
	}

	// Before timestamp (exclusive)
	if !out.Page.Before.IsZero() {
		filters = append(filters, "timestamp < :before")
		params = append(params, sql.Named("before", out.Page.Before))
	}

	// Resource filtering (if ResourceID is set, prefer it over of ResourceTypes)
	if out.Page.ResourceID != "" {
		filters = append(filters, "resource_id = :resourceId")
		params = append(params, sql.Named("resourceId", string(out.Page.ResourceID)))
	} else if 0 < len(out.Page.ResourceTypes) {
		inquery, inparams := listParametrize(out.Page.ResourceTypes, "r")
		filters = append(filters, "resource_type IN "+inquery)
		params = append(params, inparams...)
	}

	// Actor filtering (if ActorID is set, prefer it over of ActorTypes)
	if out.Page.ActorID != "" {
		filters = append(filters, "actor_id = :actorId")
		params = append(params, sql.Named("actorId", string(out.Page.ActorID)))
	} else if 0 < len(out.Page.ActorTypes) {
		inquery, inparams := listParametrize(out.Page.ActorTypes, "a")
		filters = append(filters, "actor_type IN "+inquery)
		params = append(params, inparams...)
	}

	// Concatenate filters with AND if there are any
	if len(filters) != 0 {
		query = "WITH logs AS (" + listComplianceAuditLogsSQL + ") SELECT * FROM logs WHERE "
		query += strings.Join(filters, " AND ")
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(query, params...); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		log := &models.ComplianceAuditLog{}
		if err = log.Scan(rows); err != nil {
			return nil, err
		}
		out.Logs = append(out.Logs, log)
	}

	return out, nil
}

const createComplianceAuditLogsSQL = "INSERT INTO compliance_audit_log (id, timestamp, actor_id, actor_type, resource_id, resource_type, action, resource_action_meta, signature) VALUES (:id, :timestamp, :actorId, :actorType, :resourceId, :resourceType, :action, :resourceActionMeta, :signature)"

func (t *Tx) CreateComplianceAuditLog(log *models.ComplianceAuditLog) (err error) {
	// Ensure the log has a timestamp
	// NOTE: this is different from most 'create' functions because we want the
	// modified time for the resource being modified to equal the timestamp, so
	// it should already be populated in the log because the log will be created
	// after the resource.
	if log.Timestamp.IsZero() {
		return dberr.ErrMissingTimestamp
	}

	// Complete the log with an ID, ensuring one wasn't set already
	if log.ID != uuid.Nil {
		return dberr.ErrNoIDOnCreate
	}
	log.ID = uuid.New()

	// Sign the log now that it is complete with an ID
	if err := log.Sign(); err != nil {
		return dberr.ErrInternal
	}

	if _, err = t.tx.Exec(createComplianceAuditLogsSQL, log.Params()...); err != nil {
		return dbe(err)
	}

	return nil
}
