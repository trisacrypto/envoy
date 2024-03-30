package sqlite

import (
	"context"
	"self-hosted-node/pkg/store/models"

	"github.com/oklog/ulid/v2"
)

func (s *Store) ListCounterparties(ctx context.Context, page *models.PageInfo) (*models.CounterpartyPage, error) {
	return nil, nil
}

func (s *Store) CreateCounterparty(context.Context, *models.Counterparty) error {
	return nil
}

func (s *Store) RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error) {
	return nil, nil
}

func (s *Store) UpdateCounterparty(context.Context, *models.Counterparty) error {
	return nil
}

func (s *Store) DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error {
	return nil
}
