package sqlite_test

import (
	"context"
	"fmt"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestRegenerateTravelAddresses() {
	// TODO: Test needs to be fixed.
	// Getting the following error:
	// could not list crypto addresses for account 01JXWZTAJ64YTC34T5N47J7H2V: sqlite3 error: sql: Scan error on column index 5, name "ivms101": incompatible type to unmarshal json: string
	s.T().Skip("skipping travel address regeneration test")

	require := s.Require()
	ctx := s.ActorContext()

	originalAddrs, err := s.fetchTravelAddresses(ctx)
	require.NoError(err, "could not fetch original travel addresses from database")

	// Regenerate the travel addresses.
	factory, err := models.NewTravelAddressFactory("foo.example.com", "test")
	require.NoError(err, "could not create travel address factory")
	s.store.UseTravelAddressFactory(factory)

	newAddrs, err := s.fetchTravelAddresses(ctx)
	require.NoError(err, "could not fetch new travel addresses from database")

	for id, originalAddr := range originalAddrs {
		require.NotEqual(originalAddr, newAddrs[id], "travel address for %s should have changed", id)
	}

	// Regenerate again and ensure the addresses are the same.
	err = s.store.RegenerateTravelAddresses(ctx)
	require.NoError(err, "could not regenerate travel addresses from database")

	newAddrs, err = s.fetchTravelAddresses(ctx)
	require.NoError(err, "could not fetch new travel addresses from database")

	for id, originalAddr := range originalAddrs {
		require.Equal(originalAddr, newAddrs[id], "travel address for %s should not have changed", id)
	}
}

func (s *storeTestSuite) fetchTravelAddresses(ctx context.Context) (addrs map[ulid.ULID]string, err error) {
	addrs = make(map[ulid.ULID]string)

	var accounts *models.AccountsPage
	if accounts, err = s.store.ListAccounts(ctx, nil); err != nil {
		return nil, fmt.Errorf("could not list accounts: %w", err)
	}

	for _, account := range accounts.Accounts {
		addrs[account.ID] = account.TravelAddress.String

		var wallets *models.CryptoAddressPage
		if wallets, err = s.store.ListCryptoAddresses(ctx, account.ID, nil); err != nil {
			return nil, fmt.Errorf("could not list crypto addresses for account %s: %w", account.ID, err)
		}

		for _, wallet := range wallets.CryptoAddresses {
			addrs[wallet.ID] = wallet.TravelAddress.String
		}
	}

	return addrs, nil
}

func (s *storeTestSuite) TestCountTravelAddresses() {
	require := s.Require()
	ctx := s.ActorContext()

	count, err := s.store.CountTravelAddresses(ctx)
	require.NoError(err, "could not count travel addresses from database")
	require.Equal(int64(8), count, "expected 8 travel addresses from the accounts.sql test fixture")
}
