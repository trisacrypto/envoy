package sqlite

import (
	"context"
	"database/sql"
	"net/url"
	"regexp"
	"sort"
	"strings"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"go.rtnl.ai/ulid"

	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

const (
	counterpartySearchSQL       = "SELECT id, name FROM counterparties ORDER BY name ASC"
	counterpartyDomainSearchSQL = "SELECT id, website FROM counterparties WHERE website LIKE :domainParam UNION select id, endpoint FROM counterparties WHERE endpoint LIKE :domainParam"
	counterparytSearchExpandSQL = "SELECT id, source, protocol, endpoint, name, website, country, verified_on, created FROM counterparties WHERE id=:id"
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
	log.Debug().Str("query", query.Query).Int("limit", query.Limit).Msg("counterparty search")

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var rows *sql.Rows
	if rows, err = tx.Query(counterpartySearchSQL); err != nil {
		return nil, dbe(err)
	}
	defer rows.Close()

	// Perform a fuzzy search on all the countparty names in the database.
	stubs := make(stubs, 0, query.Limit)
	stubIDs := make(map[ulid.ULID]struct{}, query.Limit)

	for rows.Next() {
		stub := counterpartyStub{}
		if err = rows.Scan(&stub.id, &stub.name); err != nil {
			return nil, err
		}

		stub.distance = fuzzy.RankMatchNormalizedFold(query.Query, stub.name)
		if stub.distance >= 0 {
			stubs = append(stubs, stub)
			stubIDs[stub.id] = struct{}{}
		}
	}

	// If the query is a URL candidate, then check if it matches a website or endpoint
	// of any of the counterparties in the database and add them to the result stubs.
	// NOTE: deduplication is necessary in the case that the query matches both a
	// website and the name of the counterparty.
	if domainParam := NormURL(query.Query); domainParam != "" {
		log.Debug().Str("domain", domainParam).Msg("counterparty search website/endpoint match")
		var domainRows *sql.Rows
		if domainRows, err = tx.Query(counterpartyDomainSearchSQL, sql.Named("domainParam", "%"+domainParam+"%")); err != nil {
			return nil, dbe(err)
		}
		defer domainRows.Close()

	domainScan:
		for domainRows.Next() {
			stub := counterpartyStub{}
			if err = domainRows.Scan(&stub.id, &stub.name); err != nil {
				return nil, err
			}

			// Check if the stub is already in the list of stubs
			if _, ok := stubIDs[stub.id]; ok {
				continue domainScan // already in the list, skip it
			}

			// If not, add it to the list of stubs
			stub.distance = fuzzy.RankMatchNormalizedFold(domainParam, stub.name)
			if stub.distance >= 0 {
				stubs = append(stubs, stub)
				stubIDs[stub.id] = struct{}{}
			}
		}
	}

	// Sort by rank so only the lowest distance values are included
	sort.Sort(stubs)

	// Apply the limit
	if len(stubs) > query.Limit {
		stubs = stubs[:query.Limit]
	}

	// Now fetch the values for the entire search results list
	out = &models.CounterpartyPage{
		Page:           &models.CounterpartyPageInfo{PageInfo: models.PageInfo{PageSize: uint32(query.Limit)}},
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

var (
	urlCandidate = regexp.MustCompile(`^(https?://)?([a-z0-9][a-z0-9-]{0,61}[a-z0-9]\.)+[a-z]{2,}(:\d{1,5})?(/.*)?$`)
)

// Returns true if the query value looks like a domain or a URL to match against
// a counterparty website or endpoint. This is not a strict URL check and may miss
// things that are valid URLs and catch things that are not; however it handles the
// most cases where http://example.com or example.com are passed in.
func IsURLCandidate(query string) bool {
	return urlCandidate.MatchString(query)
}

// Normalize the URL to a domain name and port, stripping off the scheme and path if
// present. This is used to assist a like query in the database for URL lookup.
func NormURL(query string) string {
	if query == "" || !IsURLCandidate(query) {
		return ""
	}

	if !(strings.HasPrefix(query, "http://") || strings.HasPrefix(query, "https://")) {
		query = "http://" + query
	}

	if u, _ := url.Parse(query); u != nil {
		return u.Host
	}

	return ""
}
