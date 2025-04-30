package sqlite_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	. "github.com/trisacrypto/envoy/pkg/store/sqlite"
	"go.rtnl.ai/ulid"
)

func TestIsURLCandidate(t *testing.T) {
	t.Run("True", func(t *testing.T) {
		tests := []string{
			"https://example.com",
			"http://example.com",
			"example.com",
			"www.example.com",
			"testnet.travel-rule.example.com",
			"example.com/path/to/resource",
			"envoy.local",
			"trisa.example.tr-envoy.com:443",
			"envoy.local:8100",
			"https://envoy.local:8100",
		}

		for _, test := range tests {
			require.True(t, IsURLCandidate(test), "expected %q to be a URL candidate", test)
		}
	})

	t.Run("False", func(t *testing.T) {
		tests := []string{
			"Rotational Labs",
			"rotational",
			"bit4x",
			"",
			"https://",
			"foo:80",
		}

		for _, test := range tests {
			require.False(t, IsURLCandidate(test), "expected %q to not be a URL candidate", test)
		}
	})
}

func TestNormURL(t *testing.T) {
	tests := []struct {
		in       string
		expected string
	}{
		{"", ""},
		{"https://example.com", "example.com"},
		{"http://example.com", "example.com"},
		{"example.com", "example.com"},
		{"www.example.com", "www.example.com"},
		{"testnet.travel-rule.example.com", "testnet.travel-rule.example.com"},
		{"example.com/path/to/resource", "example.com"},
		{"envoy.local", "envoy.local"},
		{"trisa.example.tr-envoy.com:443", "trisa.example.tr-envoy.com:443"},
		{"envoy.local:8100", "envoy.local:8100"},
		{"https://envoy.local:8100", "envoy.local:8100"},
		{"foo", ""},
		{"test.com://test.com", ""},
	}

	for _, test := range tests {
		require.Equal(t, test.expected, NormURL(test.in), "expected %q to normalize to %q", test.in, test.expected)
	}
}

func TestSetThresholdPanic(t *testing.T) {
	require.Panics(t, func() {
		SetThreshold(-0.0001)
	})

	require.Panics(t, func() {
		SetThreshold(3.14159)
	})
}

func TestSearchRank(t *testing.T) {
	// Make sure the similarity threshold is reset after the test.
	orig := GetThreshold()
	defer SetThreshold(orig)

	t.Run("Similarity", func(t *testing.T) {
		SetThreshold(float64(0.0))
		rank := NewSearchRank("example", 10)

		terms := []struct {
			id   string
			name string
		}{
			{"01JT1RQGSCWA43EZ1EP3H6WVWJ", "example.com"},
			{"01JT1RS3NJKXFP22A7XWD28A62", "EXAMPLE.NET"},
			{"01JT1RSKKBMQY89KT8Y854ZESP", "example"},
			{"01JT1RS3NJKXFP22A7XWD28A62", "example.net"},
			{"01JT1RY6CADNE7PC5Z34VKZQCB", "foo"},
			{"01JT1RYVDT99T23N2QF552E9WR", "ample"},
			{"01JT2345S2VAT1GKD0Z946FZN9", "éxamplé"},
		}

		for _, term := range terms {
			rank.Add(ulid.MustParse(term.id), term.name)
		}

		expected := RankList{
			{
				ID:         ulid.MustParse("01JT1RSKKBMQY89KT8Y854ZESP"),
				Name:       "example",
				Distance:   0,
				Similarity: 1.0,
			},
			{
				ID:         ulid.MustParse("01JT2345S2VAT1GKD0Z946FZN9"),
				Name:       "éxamplé",
				Distance:   0,
				Similarity: 1.0,
			},
			{
				ID:         ulid.MustParse("01JT1RYVDT99T23N2QF552E9WR"),
				Name:       "ample",
				Distance:   2,
				Similarity: 0.7142857142857143,
			},
			{
				ID:         ulid.MustParse("01JT1RQGSCWA43EZ1EP3H6WVWJ"),
				Name:       "example.com",
				Distance:   4,
				Similarity: 0.6363636363636364,
			},
			{
				ID:         ulid.MustParse("01JT1RS3NJKXFP22A7XWD28A62"),
				Name:       "EXAMPLE.NET",
				Distance:   4,
				Similarity: 0.6363636363636364,
			},
		}

		require.Equal(t, expected, rank.Results())
	})

	t.Run("Threshold", func(t *testing.T) {
		SetThreshold(float64(0.70))
		rank := NewSearchRank("example", 10)

		terms := []struct {
			id   string
			name string
		}{
			{"01JT1RQGSCWA43EZ1EP3H6WVWJ", "example.com"},
			{"01JT1RS3NJKXFP22A7XWD28A62", "EXAMPLE.NET"},
			{"01JT1RSKKBMQY89KT8Y854ZESP", "example"},
			{"01JT1RS3NJKXFP22A7XWD28A62", "example.net"},
			{"01JT1RY6CADNE7PC5Z34VKZQCB", "foo"},
			{"01JT1RYVDT99T23N2QF552E9WR", "ample"},
			{"01JT2345S2VAT1GKD0Z946FZN9", "éxamplé"},
		}

		for _, term := range terms {
			rank.Add(ulid.MustParse(term.id), term.name)
		}

		expected := RankList{
			{
				ID:         ulid.MustParse("01JT1RSKKBMQY89KT8Y854ZESP"),
				Name:       "example",
				Distance:   0,
				Similarity: 1.0,
			},
			{
				ID:         ulid.MustParse("01JT2345S2VAT1GKD0Z946FZN9"),
				Name:       "éxamplé",
				Distance:   0,
				Similarity: 1.0,
			},
			{
				ID:         ulid.MustParse("01JT1RYVDT99T23N2QF552E9WR"),
				Name:       "ample",
				Distance:   2,
				Similarity: 0.7142857142857143,
			},
		}

		require.Equal(t, expected, rank.Results())
	})

	t.Run("Limit", func(t *testing.T) {
		SetThreshold(float64(0.0))
		rank := NewSearchRank("example", 1)

		terms := []struct {
			id   string
			name string
		}{
			{"01JT1RQGSCWA43EZ1EP3H6WVWJ", "example.com"},
			{"01JT1RS3NJKXFP22A7XWD28A62", "EXAMPLE.NET"},
			{"01JT1RSKKBMQY89KT8Y854ZESP", "example"},
			{"01JT1RS3NJKXFP22A7XWD28A62", "example.net"},
			{"01JT1RY6CADNE7PC5Z34VKZQCB", "foo"},
			{"01JT1RYVDT99T23N2QF552E9WR", "ample"},
			{"01JT2345S2VAT1GKD0Z946FZN9", "éxamplé"},
		}

		for _, term := range terms {
			rank.Add(ulid.MustParse(term.id), term.name)
		}

		expected := RankList{
			{
				ID:         ulid.MustParse("01JT1RSKKBMQY89KT8Y854ZESP"),
				Name:       "example",
				Distance:   0,
				Similarity: 1.0,
			},
		}

		require.Equal(t, expected, rank.Results())
	})
}

func TestRank(t *testing.T) {
	tests := []struct {
		term  string
		query string
	}{
		{"example.com", "example.com"},
		{"example.com", "example"},
		{"AMPLE", "example"},
		{"path/to", "example.com/path/to/resource"},
		{"foo", "bar"},
		{"njonito argo juntian", "frim flam blam heck heck sooner"},
		{"cartwheel", "cartwhéél"},
	}

	for _, test := range tests {
		d1, s1 := Rank(test.term, test.query)
		d2, s2 := Rank(test.query, test.term)
		require.Equal(t, d1, d2, "expected distance to be equal")
		require.Equal(t, s1, s2, "expected similarity to be equal")
	}
}
