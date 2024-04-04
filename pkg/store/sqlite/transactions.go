package sqlite

import (
	"context"

	"self-hosted-node/pkg/store/models"

	"github.com/oklog/ulid/v2"
)

func (s *Store) ListTransactions(context.Context, *models.PageInfo) (*models.TransactionPage, error) {
	return nil, nil
}

func (s *Store) CreateTransaction(context.Context, *models.Transaction) error {
	return nil
}

func (s *Store) RetrieveTransaction(context.Context, ulid.ULID) (*models.Transaction, error) {
	return nil, nil
}

func (s *Store) UpdateTransaction(context.Context, *models.Transaction) error {
	return nil
}

func (s *Store) DeleteTransaction(context.Context, ulid.ULID) error {
	return nil
}

func (s *Store) ListSecureEnvelopes(ctx context.Context, txID ulid.ULID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	return nil, nil
}

func (s *Store) CreateSecureEnvelope(context.Context, *models.SecureEnvelope) error {
	return nil
}

func (s *Store) RetrieveSecureEnvelope(ctx context.Context, txID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	return nil, nil
}

func (s *Store) UpdateSecureEnvelope(context.Context, *models.SecureEnvelope) error {
	return nil
}

func (s *Store) DeleteSecureEnvelope(ctx context.Context, txID, envID ulid.ULID) error {
	return nil
}
