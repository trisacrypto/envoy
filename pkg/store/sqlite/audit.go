package sqlite

import (
	"context"
	"database/sql"
	"fmt"

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

func (t *Tx) ListComplianceAuditLogs(page *models.ComplianceAuditLogPageInfo) (out *models.ComplianceAuditLogPage, err error) {
	//FIXME: implement it
	return nil, fmt.Errorf("not implemented")
}

func (t *Tx) CreateComplianceAuditLog(log *models.ComplianceAuditLog) (err error) {
	//FIXME: implement it
	return fmt.Errorf("not implemented")
}
