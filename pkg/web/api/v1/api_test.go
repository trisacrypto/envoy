package api_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

func TestSearchQuery(t *testing.T) {
	t.Run("Validate", func(t *testing.T) {
		q := &api.SearchQuery{Query: "coinbase", Limit: 10}
		require.NoError(t, q.Validate())

		q = &api.SearchQuery{Query: "coinbase", Limit: 0}
		require.NoError(t, q.Validate())
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			q   *api.SearchQuery
			err error
		}{
			{
				&api.SearchQuery{Limit: 12},
				api.MissingField("query"),
			},
			{
				&api.SearchQuery{Query: "coinbase", Limit: -14},
				api.IncorrectField("limit", "limit cannot be less than zero"),
			},
			{
				&api.SearchQuery{Query: "coinbase", Limit: 100},
				api.IncorrectField("limit", "maximum number of search results that can be returned is 50"),
			},
		}

		for i, tc := range tests {
			require.EqualError(t, tc.q.Validate(), tc.err.Error(), "test case %d failed", i)
		}
	})

	t.Run("Model", func(t *testing.T) {
		q := &api.SearchQuery{Query: "coinbase", Limit: 10}
		model := q.Model()
		require.Equal(t, q.Query, model.Query)
		require.Equal(t, q.Limit, model.Limit)
	})
}
