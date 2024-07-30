package sqlite

import (
	"context"
	"database/sql"
	"sort"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/oklog/ulid/v2"

	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

const (
	counterpartySearchSQL       = "SELECT id, name FROM counterparties ORDER BY name ASC"
	counterparytSearchExpandSQL = "SELECT id, source, protocol, endpoint, name, website, country, created FROM counterparties WHERE id=:id"
)

type counterpartyStub struct {
	id       ulid.ULID
	name     string
	distance int
}

type stubs []counterpartyStub

func (r stubs) Len() int {
	return len(r)
}

func (r stubs) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r stubs) Less(i, j int) bool {
	return r[i].distance < r[j].distance
}

func (s *Store) SearchCounterparties(ctx context.Context, query *models.SearchQuery) (out *models.CounterpartyPage, err error) {
	log := logger.Tracing(ctx)

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var rows *sql.Rows
	if rows, err = tx.Query(counterpartySearchSQL); err != nil {
		return nil, err
	}
	defer rows.Close()

	// Perform a fuzzy search on all the stubs
	stubs := make(stubs, 0, query.Limit)
	for rows.Next() {
		stub := counterpartyStub{}
		if err = rows.Scan(&stub.id, &stub.name); err != nil {
			return nil, err
		}

		stub.distance = fuzzy.RankMatchNormalizedFold(query.Query, stub.name)
		if stub.distance < 0 {
			continue
		}

		stubs = append(stubs, stub)
	}

	// Sort by rank so only the lowest distance values are included
	sort.Sort(stubs)

	// Apply the limit
	if len(stubs) > query.Limit {
		stubs = stubs[:query.Limit]
	}

	// Now fetch the values for the entire search results list
	out = &models.CounterpartyPage{
		Page:           &models.PageInfo{PageSize: uint32(query.Limit)},
		Counterparties: make([]*models.Counterparty, 0, len(stubs)),
	}

	// Expand the results to include the complete required payload.
	for _, stub := range stubs {
		log.Trace().Str("name", stub.name).Int("rank", stub.distance).Msg("counterparty search")

		cp := &models.Counterparty{}
		if err = cp.ScanSummary(tx.QueryRow(counterparytSearchExpandSQL, sql.Named("id", stub.id))); err != nil {
			return nil, err
		}
		out.Counterparties = append(out.Counterparties, cp)
	}

	return out, nil
}
