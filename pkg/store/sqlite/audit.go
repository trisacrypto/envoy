package sqlite

import (
	"context"
	"database/sql"
	"strings"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
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

	// Ensure the log has a ResourceModified timestamp (this should be)
	if log.ResourceModified.IsZero() {
		return dberr.ErrMissingTimestamp
	}

	// Complete the log with an ID, ensuring one wasn't set already
	if log.ID != ulid.Zero {
		return dberr.ErrNoIDOnCreate
	}
	log.ID = ulid.MakeSecure()

	// Sign the log now that it is complete with an ID
	var signature []byte
	if signature, err = s.Sign(log.Data()); err != nil {
		return dberr.ErrInternal
	}
	log.Signature = signature

	// Perform the transaction to insert the log
	if err = tx.CreateComplianceAuditLog(log); err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	return nil
}

func (s *Store) RetrieveComplianceAuditLog(ctx context.Context, id ulid.ULID) (log *models.ComplianceAuditLog, err error) {
	var tx *Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: false}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if log, err = tx.RetrieveComplianceAuditLog(id); err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, err
	}

	return log, err
}

// ###################################################
// # ComplianceAuditLogTxn implementation for SQLite #
// ###################################################

const listComplianceAuditLogsSummarySQL = "SELECT id, actor_id, actor_type, resource_id, resource_type, resource_modified, action FROM compliance_audit_log ORDER BY resource_modified DESC"
const listComplianceAuditLogsDetailedSQL = "SELECT id, actor_id, actor_type, resource_id, resource_type, resource_modified, action, change_notes, signature, key_id, algorithm FROM compliance_audit_log ORDER BY resource_modified DESC"

func (t *Tx) ListComplianceAuditLogs(page *models.ComplianceAuditLogPageInfo) (out *models.ComplianceAuditLogPage, err error) {
	// Setup out variable with page info
	out = &models.ComplianceAuditLogPage{
		Logs: make([]*models.ComplianceAuditLog, 0),
		// TODO: implement pagination
		Page: models.ComplianceAuditLogPageInfoFrom(page),
	}

	// Setup base query (by default return 'summary' information only)
	query := listComplianceAuditLogsSummarySQL
	if out.Page.DetailedLogs {
		query = listComplianceAuditLogsDetailedSQL
	}

	// Create lists for filters/params then process each filter option
	params := make([]any, 0, 4)
	filters := make([]string, 0, 4)

	// After resource_modified (inclusive)
	if !out.Page.After.IsZero() {
		filters = append(filters, ":after <= resource_modified")
		params = append(params, sql.Named("after", out.Page.After))
	}

	// Before resource_modified (exclusive)
	if !out.Page.Before.IsZero() {
		filters = append(filters, "resource_modified < :before")
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
		query = "WITH logs AS (" + query + ") SELECT * FROM logs WHERE "
		query += strings.Join(filters, " AND ")
	}

	var rows *sql.Rows
	if rows, err = t.tx.Query(query, params...); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		log := &models.ComplianceAuditLog{}

		// Scan the summary info only unless the user has requested detailed logs
		if !out.Page.DetailedLogs {
			if err = log.ScanSummary(rows); err != nil {
				return nil, err
			}
		} else {
			if err = log.Scan(rows); err != nil {
				return nil, err
			}
		}

		out.Logs = append(out.Logs, log)
	}

	return out, nil
}

const createComplianceAuditLogsSQL = "INSERT INTO compliance_audit_log (id, actor_id, actor_type, resource_id, resource_type, resource_modified, action, change_notes, signature, key_id, algorithm) VALUES (:id, :actorId, :actorType, :resourceId, :resourceType, :resourceModified, :action, :changeNotes, :signature, :keyId, :algorithm)"

func (t *Tx) CreateComplianceAuditLog(log *models.ComplianceAuditLog) (err error) {
	if _, err = t.tx.Exec(createComplianceAuditLogsSQL, log.Params()...); err != nil {
		return dbe(err)
	}

	return nil
}

const retrieveComplianceAuditLogSQL = "SELECT id, actor_id, actor_type, resource_id, resource_type, resource_modified, action, change_notes, signature, key_id, algorithm FROM compliance_audit_log WHERE id = :id"

func (t *Tx) RetrieveComplianceAuditLog(id ulid.ULID) (log *models.ComplianceAuditLog, err error) {
	log = &models.ComplianceAuditLog{}
	if err = log.Scan(t.tx.QueryRow(retrieveComplianceAuditLogSQL, sql.Named("id", id))); err != nil {
		return nil, dbe(err)
	}
	return log, nil
}
