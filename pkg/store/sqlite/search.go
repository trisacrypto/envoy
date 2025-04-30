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

//===========================================================================
// Similarity Threshold
//===========================================================================

var threshold float64 = 0.0

// SetThreshold sets the threshold for the fuzzy search. The threshold must be a number
// between 0.0 and 1.0 where a higher threshold requires stricter matching. Setting a
// threshold of 0.0 will allow any rank fold matches.
func SetThreshold(t float64) {
	if t < 0.0 || t > 1.0 {
		panic("threshold must be between 0.0 and 1.0")
	}
	threshold = t
}

// GetThreshold returns the current threshold for the fuzzy search.
func GetThreshold() float64 {
	return threshold
}

//===========================================================================
// Counterparty Fuzzy Search
//===========================================================================

const (
	counterpartySearchSQL       = "SELECT id, name FROM counterparties ORDER BY name ASC"
	counterpartyDomainSearchSQL = "SELECT id, website FROM counterparties WHERE website LIKE :domainParam UNION select id, endpoint FROM counterparties WHERE endpoint LIKE :domainParam"
	counterpartySearchExpandSQL = "SELECT id, source, protocol, endpoint, name, website, country, verified_on, created FROM counterparties WHERE id=:id"
)

// SearchCounterparties uses an in-memory fuzzy search to find counterparties whose name
// matches the query using a unicode normalization, case-insensitive method. Additionally,
// if the query looks like a URL, it will also search for counterparties whose website or
// endpoint matches the query. The results are sorted by the distance of the match, with
// the closest matches first. The results are limited to the number of results specified
// in the query. Note that this in-memory fuzzy search is required for SQLite since there
// is no native search in SQLite without an extension.
func (s *Store) SearchCounterparties(ctx context.Context, query *models.SearchQuery) (out *models.CounterpartyPage, err error) {
	log := logger.Tracing(ctx)
	log.Debug().Str("query", query.Query).Int("limit", query.Limit).Msg("counterparty search")

	var tx *sql.Tx
	if tx, err = s.BeginTx(ctx, &sql.TxOptions{ReadOnly: true}); err != nil {
		return nil, err
	}
	defer tx.Rollback()

	// Perform a fuzzy search on all the countparty names in the database.
	rank := NewSearchRank(query.Query, query.Limit)
	if err = s.fuzzySearchCounterparties(tx, rank); err != nil {
		return nil, err
	}

	// If the query is a URL candidate, then check if it matches a website or endpoint
	// of any of the counterparties in the database and add them to the result stubs.
	// NOTE: deduplication is necessary in the case that the query matches both a
	// website and the name of the counterparty.
	if domainParam := NormURL(query.Query); domainParam != "" {
		log.Debug().Str("domain", domainParam).Msg("counterparty search website/endpoint match")
		if err = s.domainSearchCounterparties(tx, domainParam, rank); err != nil {
			return nil, err
		}
	}

	// Now fetch the values for the entire search results list
	results := rank.Results()
	out = &models.CounterpartyPage{
		Page:           &models.CounterpartyPageInfo{PageInfo: models.PageInfo{PageSize: uint32(query.Limit)}},
		Counterparties: make([]*models.Counterparty, 0, len(results)),
	}

	// Expand the results to include the complete required payload.
	for _, item := range results {
		log.Trace().Str("name", item.Name).Int("rank", item.Distance).Float64("similarity", item.Similarity).Msg("counterparty search")

		cp := &models.Counterparty{}
		if err = cp.ScanSummary(tx.QueryRow(counterpartySearchExpandSQL, sql.Named("id", item.ID))); err != nil {
			return nil, err
		}
		out.Counterparties = append(out.Counterparties, cp)
	}

	return out, nil
}

func (s *Store) fuzzySearchCounterparties(tx *sql.Tx, rank *SearchRank) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(counterpartySearchSQL); err != nil {
		return dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		item := &RankItem{}
		if err = rows.Scan(&item.ID, &item.Name); err != nil {
			return err
		}

		rank.Append(item)
	}

	return rows.Err()
}

func (s *Store) domainSearchCounterparties(tx *sql.Tx, domain string, rank *SearchRank) (err error) {
	var rows *sql.Rows
	if rows, err = tx.Query(counterpartyDomainSearchSQL, sql.Named("domainParam", "%"+domain+"%")); err != nil {
		return dbe(err)
	}
	defer rows.Close()

	for rows.Next() {
		item := &RankItem{}
		if err = rows.Scan(&item.ID, &item.Name); err != nil {
			return err
		}

		rank.Append(item)
	}

	return rows.Err()
}

//===========================================================================
// URL/Domain Search Helpers
//===========================================================================

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

//===========================================================================
// Ranking Search Results
//===========================================================================

type SearchRank struct {
	query string
	limit int
	items RankList
	idset map[ulid.ULID]struct{}
}

func NewSearchRank(query string, limit int) *SearchRank {
	return &SearchRank{
		query: query,
		limit: limit,
		items: make(RankList, 0, limit+1),
		idset: make(map[ulid.ULID]struct{}, limit+1),
	}
}

func (s *SearchRank) Add(id ulid.ULID, name string) bool {
	return s.Append(&RankItem{ID: id, Name: name})
}

func (s *SearchRank) Append(item *RankItem) bool {
	if _, ok := s.idset[item.ID]; ok {
		return false // already in the list
	}

	item.Distance, item.Similarity = Rank(item.Name, s.query)
	if item.Distance < 0 {
		return false // not a match
	}

	if item.Similarity < threshold {
		return false // not within the similarity threshold
	}

	s.idset[item.ID] = struct{}{}
	s.items = append(s.items, *item)

	if len(s.items) > s.limit {
		// Sort the items by distance to get the closest matches first
		sort.Sort(s.items)

		// Remove any IDs from any items that are not in the limit
		for _, item := range s.items[s.limit:] {
			delete(s.idset, item.ID)
		}

		// Trim the list to the limit
		s.items = s.items[:s.limit]
	}

	return true
}

func (s *SearchRank) Results() RankList {
	sort.Sort(s.items)
	return s.items
}

// Rank attempts to perform a substring match of the term on the query using unicode
// normalized case-insensitive fuzzy search. If the term is longer than the query then
// the query is matched to the term to find similarity regardless of substring
// containment. E.g. a query for example.com should match the example and a query for
// ample should match the same term. The distance and the similarity is returned.
func Rank(term, query string) (int, float64) {
	if len(term) > len(query) {
		return Rank(query, term)
	}

	// Attempt to match term to query then query to term.
	distance := fuzzy.RankMatchNormalizedFold(term, query)
	if distance < 0 {
		return -1, 0.0
	}

	similarity := 1.0 - float64(distance)/float64(len(query))
	return distance, similarity
}

type RankItem struct {
	ID         ulid.ULID
	Name       string
	Distance   int
	Similarity float64
}

type RankList []RankItem

func (r RankList) Len() int {
	return len(r)
}

func (r RankList) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r RankList) Less(i, j int) bool {
	return r[i].Distance < r[j].Distance
}
