package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestListAccounts() {
	//setup
	require := s.Require()
	ctx := context.Background()

	//test
	accounts, err := s.store.ListAccounts(ctx, nil)
	require.NoError(err, "expected no errors")
	require.NotNil(accounts.Accounts, "there were no accounts")
	require.Len(accounts.Accounts, 2, fmt.Sprintf("there should be 2 accounts, but there were %d", len(accounts.Accounts)))
}

func (s *storeTestSuite) TestCreateAccount() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		account := mock.GetSampleAccount(true, true, true)
		account.ID = ulid.Zero

		//test
		err := s.store.CreateAccount(ctx, account)
		require.NoError(err, "no error was expected")

		account2, err := s.store.RetrieveAccount(ctx, account.ID)
		require.NoError(err, "expected no error")
		require.NotNil(account2, "account should not be nil")
		require.Equal(account.ID, account2.ID, fmt.Sprintf("account ID should be %s, found %s instead", account.ID, account2.ID))
	})

	s.Run("FailureNonZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		account := mock.GetSampleAccount(true, true, true)

		//test
		err := s.store.CreateAccount(ctx, account)
		require.Error(err, "an error was expected")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected an ErrNoIDOnCreate error")
	})
}

func (s *storeTestSuite) TestLookupAccount() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		cryptoAddress := "n2irvV1QpYfV2XysspZ9hdiQyHHHh8xtX3"
		accountId := "01HV6QS6AK4KNS46Q9HEB7DTPR"

		//test
		account, err := s.store.LookupAccount(ctx, cryptoAddress)
		require.NoError(err, "expected no error")
		require.NotNil(account, "account should not be nil")
		require.Equal(accountId, account.ID.String(), fmt.Sprintf("account ID should be %s, found %s instead", accountId, account.ID))

		cryptoAddresses, err := account.CryptoAddresses()
		require.NoError(err, "expected no error")
		require.NotNil(account, "cryptoAddresses should not be nil")
		require.Len(cryptoAddresses, 1, fmt.Sprintf("there should be 1 cryptoAddresses but there are %d", len(cryptoAddresses)))
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		cryptoAddress := "mzWQkWXcT8idugd2v3MUGucBaUCSJp948B" // fake generated address

		//test
		account, err := s.store.LookupAccount(ctx, cryptoAddress)
		require.Nil(account, "account should be nil")
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")
	})
}

func (s *storeTestSuite) TestRetrieveAccount() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")

		//test
		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")
		require.NotNil(account, "account should not be nil")
		require.Equal(accountId, account.ID, fmt.Sprintf("account ID should be %s, found %s instead", accountId, account.ID))

		cryptoAddresses, err := account.CryptoAddresses()
		require.NoError(err, "expected no error")
		require.NotNil(account, "cryptoAddresses should not be nil")
		require.Len(cryptoAddresses, 1, fmt.Sprintf("there should be %d cryptoAddresses but there are %d", 1, len(cryptoAddresses)))
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MakeSecure()

		//test
		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.Nil(account, "account should be nil")
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")
	})
}

func (s *storeTestSuite) TestUpdateAccount() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")

		newFirstName := sql.NullString{String: account.FirstName.String + "extrastuff", Valid: true}
		account.FirstName = newFirstName
		newLastName := sql.NullString{String: account.LastName.String + "extrastuff", Valid: true}
		account.LastName = newLastName

		//test
		err = s.store.UpdateAccount(ctx, account)
		require.NoError(err, "expected no error")

		account, err = s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")
		require.Equal(newFirstName, account.FirstName)
		require.Equal(newLastName, account.LastName)
	})

	s.Run("FailureZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")

		account.ID = ulid.Zero

		//test
		err = s.store.UpdateAccount(ctx, account)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrMissingID, err, "expected an ErrMissingID error")
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")

		account.ID = ulid.MakeSecure()

		//test
		err = s.store.UpdateAccount(ctx, account)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")
	})
}

func (s *storeTestSuite) TestDeleteAccount() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")

		//test
		err := s.store.DeleteAccount(ctx, accountId)
		require.NoError(err, "expected no error")

		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.Nil(account, "account should be nil")
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MakeSecure()

		//test
		err := s.store.DeleteAccount(ctx, accountId)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")
	})
}

func (s *storeTestSuite) TestListAccountTransactions() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")

		//test
		transactions, err := s.store.ListAccountTransactions(ctx, accountId, &models.TransactionPageInfo{})
		require.NoError(err, "expected no error")
		require.NotNil(transactions, "transactions should not be nil")
		require.NotNil(transactions.Transactions, "transactions.Transactions should not be nil")
		require.Len(transactions.Transactions, 2, fmt.Sprintf("should be 2 transactions, found %d instead", len(transactions.Transactions)))
	})

	s.Run("SuccessStatusFilter", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")

		//test
		transactions, err := s.store.ListAccountTransactions(ctx, accountId, &models.TransactionPageInfo{Status: []string{"pending"}})
		require.NoError(err, "expected no error")
		require.NotNil(transactions, "transactions should not be nil")
		require.NotNil(transactions.Transactions, "transactions.Transactions should not be nil")
		require.Len(transactions.Transactions, 1, fmt.Sprintf("should be 1 transactions, found %d instead", len(transactions.Transactions)))
	})

	s.Run("SuccessAssetFilter", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")

		//test
		transactions, err := s.store.ListAccountTransactions(ctx, accountId, &models.TransactionPageInfo{VirtualAsset: []string{"LTC"}})
		require.NoError(err, "expected no error")
		require.NotNil(transactions, "transactions should not be nil")
		require.NotNil(transactions.Transactions, "transactions.Transactions should not be nil")
		require.Len(transactions.Transactions, 1, fmt.Sprintf("should be 1 transactions, found %d instead", len(transactions.Transactions)))
	})

	s.Run("SuccessArchiveFilter", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")

		//test
		transactions, err := s.store.ListAccountTransactions(ctx, accountId, &models.TransactionPageInfo{Archives: true})
		require.NoError(err, "expected no error")
		require.NotNil(transactions, "transactions should not be nil")
		require.NotNil(transactions.Transactions, "transactions.Transactions should not be nil")
		require.Len(transactions.Transactions, 1, fmt.Sprintf("should be 1 transactions, found %d instead", len(transactions.Transactions)))
	})

	s.Run("NoTransactions", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MakeSecure()

		//test
		transactions, err := s.store.ListAccountTransactions(ctx, accountId, &models.TransactionPageInfo{})
		require.NoError(err, "expected no error")
		require.NotNil(transactions, "transactions should not be nil")
		require.NotNil(transactions.Transactions, "transactions.Transactions should not be nil")
		require.Len(transactions.Transactions, 0, fmt.Sprintf("should be 0 transactions, found %d instead", len(transactions.Transactions)))
	})

	s.Run("PanicsNilPageInfo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")

		//test
		require.Panics(func() { s.store.ListAccountTransactions(ctx, accountId, nil) }, "should panic with nil page info")
	})
}

func (s *storeTestSuite) TestListCryptoAddresses() {
	s.Run("SuccessOne", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")

		//test
		cryptoAddresses, err := s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(cryptoAddresses.CryptoAddresses, "there were no crypto addresses")
		require.Len(cryptoAddresses.CryptoAddresses, 1, fmt.Sprintf("there should be 1 crypto address, but there were %d", len(cryptoAddresses.CryptoAddresses)))
	})

	s.Run("SuccessTwo", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")

		//test
		cryptoAddresses, err := s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(cryptoAddresses.CryptoAddresses, "there were no crypto addresses")
		require.Len(cryptoAddresses.CryptoAddresses, 2, fmt.Sprintf("there should be 2 crypto address, but there were %d", len(cryptoAddresses.CryptoAddresses)))
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MakeSecure()

		//test
		cryptoAddresses, err := s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound error")
		require.Nil(cryptoAddresses, "cryptoAddresses should be nil")
	})

	s.Run("FailureNotFoundZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.Zero

		//test
		cryptoAddresses, err := s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound error")
		require.Nil(cryptoAddresses, "cryptoAddresses should be nil")
	})
}

func (s *storeTestSuite) TestCreateCryptoAddress() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")
		cryptoAddress := mock.GetSampleCryptoAddress(accountId)

		//test
		addresses, err := s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no error")
		require.NotNil(addresses, "addresses should not be nil")
		require.Len(addresses.CryptoAddresses, 2, fmt.Sprintf("expected 2 crypto addresses, got %d", len(addresses.CryptoAddresses)))

		err = s.store.CreateCryptoAddress(ctx, cryptoAddress)
		require.NoError(err, "no error was expected")

		addresses, err = s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no error")
		require.NotNil(addresses, "addresses should not be nil")
		require.Len(addresses.CryptoAddresses, 3, fmt.Sprintf("expected 3 crypto addresses, got %d", len(addresses.CryptoAddresses)))
	})

	s.Run("FailureNotFoundAccountID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MakeSecure()
		cryptoAddress := mock.GetSampleCryptoAddress(accountId)

		//test
		err := s.store.CreateCryptoAddress(ctx, cryptoAddress)
		require.Error(err, "an error was expected")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected error ErrAlreadyExists")
	})

	s.Run("FailureZeroAccountID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.Zero
		cryptoAddress := mock.GetSampleCryptoAddress(accountId)

		//test
		err := s.store.CreateCryptoAddress(ctx, cryptoAddress)
		require.Error(err, "an error was expected")
		require.Equal(errors.ErrMissingReference, err, "expected error ErrMissingReference")
	})

	s.Run("FailureAddressNotZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MakeSecure()
		cryptoAddress := mock.GetSampleCryptoAddress(accountId)
		cryptoAddress.ID = ulid.MakeSecure()

		//test
		err := s.store.CreateCryptoAddress(ctx, cryptoAddress)
		require.Error(err, "an error was expected")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected error ErrNoIDOnCreate")
	})
}

func (s *storeTestSuite) TestRetrieveCryptoAddress() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")

		//test
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.Nil(err, "expected no error")
		require.NotNil(address, "crypto address should not be nil")
		require.Equal(address.AccountID, accountId, fmt.Sprintf("expected address ID %s, got %s", address.AccountID.String(), accountId.String()))
		require.Equal(address.ID, addressId, fmt.Sprintf("expected address ID %s, got %s", address.ID.String(), addressId.String()))

		// NOTE: currently the account is not retrieved and associated with the crypto address, but this may change
		account, err := address.Account()
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrMissingAssociation, err, "expected the ErrMissingAssociation error")
		require.Nil(account, "account should be nil")
	})

	s.Run("FailureNotFoundRandomAccountID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MakeSecure()
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")

		//test
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(address, "crypto address should be nil")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MakeSecure()

		//test
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(address, "crypto address should be nil")
	})

	s.Run("FailureNotFoundZeroAccountID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.Zero
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")

		//test
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(address, "crypto address should be nil")
	})

	s.Run("FailureNotFoundZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.Zero

		//test
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(address, "crypto address should be nil")
	})
}

func (s *storeTestSuite) TestUpdateCryptoAddress() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		newNetwork := "LTC"
		if address.Network == "LTC" {
			newNetwork = "BTC"
		}
		address.Network = newNetwork

		//test
		err = s.store.UpdateCryptoAddress(ctx, address)
		require.NoError(err, "expected no error")

		address, err = s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")
		require.Equal(newNetwork, address.Network)
	})

	s.Run("FailureZeroID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		address.ID = ulid.Zero

		//test
		err = s.store.UpdateCryptoAddress(ctx, address)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrMissingID, err, "expected an ErrMissingID error")
	})

	s.Run("FailureZeroAccountID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		address.AccountID = ulid.Zero

		//test
		err = s.store.UpdateCryptoAddress(ctx, address)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrMissingReference, err, "expected an ErrMissingReference error")
	})

	s.Run("FailureNotFoundAddress", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		address.ID = ulid.MakeSecure()

		//test
		err = s.store.UpdateCryptoAddress(ctx, address)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")
	})

	s.Run("FailureNotFoundAccount", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		address.AccountID = ulid.MakeSecure()

		//test
		err = s.store.UpdateCryptoAddress(ctx, address)
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")
	})
}

func (s *storeTestSuite) TestDeleteCryptoAddress() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")
		addressId := ulid.MustParse("01HV6RV08YNR2GH8MEEKB7DH2W")

		//test
		cryptoAddresses, err := s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(cryptoAddresses.CryptoAddresses, "there were no crypto addresses")
		require.Len(cryptoAddresses.CryptoAddresses, 2, fmt.Sprintf("there should be 2 crypto addresses, but there were %d", len(cryptoAddresses.CryptoAddresses)))

		err = s.store.DeleteCryptoAddress(ctx, accountId, addressId)
		require.Nil(err, "expected no error")

		cryptoAddresses, err = s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(cryptoAddresses.CryptoAddresses, "there were no crypto addresses")
		require.Len(cryptoAddresses.CryptoAddresses, 1, fmt.Sprintf("there should be 1 crypto address, but there were %d", len(cryptoAddresses.CryptoAddresses)))
	})

	s.Run("FailureNotFoundAccountID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MakeSecure()
		addressId := ulid.MustParse("01HV6RV08YNR2GH8MEEKB7DH2W")

		//test
		err := s.store.DeleteCryptoAddress(ctx, accountId, addressId)
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureNotFoundAddressID", func() {
		//setup
		require := s.Require()
		ctx := context.Background()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")
		addressId := ulid.MakeSecure()

		//test
		err := s.store.DeleteCryptoAddress(ctx, accountId, addressId)
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}
