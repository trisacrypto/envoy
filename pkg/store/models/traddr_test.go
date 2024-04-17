package models_test

import (
	"testing"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/require"
)

func TestTravelAddressFactory(t *testing.T) {
	t.Run("Empty", func(t *testing.T) {
		factory, err := models.NewTravelAddressFactory("", "")
		require.ErrorIs(t, err, dberr.ErrNoEndpoint)
		require.Nil(t, factory)
	})

	t.Run("Unparseable", func(t *testing.T) {
		factory, err := models.NewTravelAddressFactory("  ", "")
		require.Error(t, err)
		require.Nil(t, factory)
	})

	t.Run("Invalid", func(t *testing.T) {
		testCases := []struct {
			endpoint string
			protocol string
			input    any
			expected string
		}{
			{"example.com", "trisa", "foo", "cannot create travel address for unhandled type string"},
			{"example.com", "", "foo", "cannot create travel address for unhandled type string"},
		}

		for i, tc := range testCases {
			factory, err := models.NewTravelAddressFactory(tc.endpoint, tc.protocol)
			require.NoError(t, err, "could not create factory for test case %d", i)

			actual, err := factory(tc.input)
			require.EqualError(t, err, tc.expected, "expected error on input for test case %d", i)
			require.Zero(t, actual, "expected emptyh string for test case %d", i)
		}
	})

	t.Run("Valid", func(t *testing.T) {
		testCases := []struct {
			endpoint string
			protocol string
			input    any
			expected string
		}{
			{
				"trisa.example.com",
				"",
				&models.Account{Model: models.Model{ID: ulid.MustParse("01HV5F21BJ1APPMSMM84J919M4")}},
				"ta2a5jNFZT9286et5Y12gwsJMMxv57x3vPDuvG9yqwW4yaaH9f2Pw11erHvbJ1ce1aUB6i1vpA7UuBxkiWbE7R",
			},
			{
				"trisa.example.com",
				"trisa",
				&models.Account{Model: models.Model{ID: ulid.MustParse("01HV5F21BJ1APPMSMM84J919M4")}},
				"ta2ih14ZfMmba1ZK3succR82cUAXXj8hii4qWV5fUzVCgRxQPXWnWVHX9v9tdd5ZbiG7VA8o2R2ZjKykGbyzCsJD1jcr5X7KzFgsp",
			},
			{
				"trisa.example.com",
				"trp",
				&models.Account{Model: models.Model{ID: ulid.MustParse("01HV5F21BJ1APPMSMM84J919M4")}},
				"ta67o1xnBN1WpmRuBPrkG9XPyPxxFz7c1o61p8HUa2Mpz6Ch14nfqW9XfmRmyq4tzdhokxNepmVdZWvYzYmdBchoCsss5jr7At",
			},
			{
				"trisa.example.com",
				"trisa",
				&models.CryptoAddress{Model: models.Model{ID: ulid.MustParse("01HV5F21BJ1APPMSMM84J919M4")}},
				"taPaz7FCQjjGdq6uJstZwrdAPvNNCZMCHRsyXqiKuSdcYu18HvZNZbZV2aiieqzM4zo3zx1TAF8VuFbC5FasAQPiPxTa3GWsjT7",
			},
			{
				"trisa.example.com",
				"trisa",
				ulid.MustParse("01HV5F21BJ1APPMSMM84J919M4"),
				"taXbQt67CE1SKCedXpfRu9DBaW1iWnsEeXeXchMbMPghbjfyUR8QJYPcWEKrV9Yg151q8CDW4vFVLxPywYDcb7QF",
			},
			{
				"trisa.example.com",
				"trisa",
				uuid.MustParse("0e626296-536b-4f6c-9386-542d7e69cc9b"),
				"ta8b1ZGjAJfqahTcfhhEBKU27YtkKPx4WoXULwreqpxed2x8WRbakNywbenmmvgqzL5rmN8pyfLZsvwy4G3qmKCA2MqQ4wMMG9eBu2",
			},
		}

		for i, tc := range testCases {
			factory, err := models.NewTravelAddressFactory(tc.endpoint, tc.protocol)
			require.NoError(t, err, "could not create factory for test case %d", i)

			actual, err := factory(tc.input)
			require.NoError(t, err, "expected no error on input for test case %d", i)
			require.Equal(t, tc.expected, actual, "mismatch expectations for test case %d", i)
		}
	})
}
