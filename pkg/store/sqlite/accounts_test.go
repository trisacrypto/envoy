package sqlite_test

import (
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
	ctx := s.ActorContext()

	//test
	accounts, err := s.store.ListAccounts(ctx, nil)
	require.NoError(err, "expected no errors")
	require.NotNil(accounts.Accounts, "there were no accounts")
	require.Len(accounts.Accounts, 5, fmt.Sprintf("there should be 5 accounts, but there were %d", len(accounts.Accounts)))
}

func (s *storeTestSuite) TestAccountIVMSRecords() {
	//setup
	require := s.Require()
	ctx := s.ActorContext()

	//test
	accounts, err := s.store.ListAccounts(ctx, nil)
	require.NoError(err, "expected no errors")
	require.NotNil(accounts, "there is no accounts page")
	require.NotNil(accounts.Accounts, "there is no accounts list")

	for _, account := range accounts.Accounts {
		// ensure accounts from the test data SQL have correct IMVS records
		switch account.FirstName.String {
		case "Frank":
			require.False(account.HasIVMSRecord(), "expected a nil IVMS record for Frank")
		case "Mary":
			require.False(account.HasIVMSRecord(), "expected a nil IVMS record for Mary")
		case "Julius":
			require.True(account.HasIVMSRecord(), "expected an IVMS record for Julius")
		case "Yami":
			require.True(account.HasIVMSRecord(), "expected an IVMS record for Yami")
		case "Rakuro":
			require.True(account.HasIVMSRecord(), "expected an IVMS record for Rakuro")
		default:
			// not added to test yet: fail it
			require.True(false, fmt.Sprintf("%s %s has not been added to the IVMS test", account.FirstName.String, account.LastName.String))
		}
	}
}

func (s *storeTestSuite) TestCreateAccount() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		account := mock.GetSampleAccount(true, true, true)
		account.ID = ulid.Zero

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) + 3 // 1 account, 2 crypto addresses

		//test
		err = s.store.CreateAccount(ctx, account, &models.ComplianceAuditLog{})
		require.NoError(err, "no error was expected")

		account2, err := s.store.RetrieveAccount(ctx, account.ID)
		require.NoError(err, "expected no error")
		require.NotNil(account2, "account should not be nil")
		require.Equal(account.ID, account2.ID, fmt.Sprintf("account ID should be %s, found %s instead", account.ID, account2.ID))

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureNonZeroID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		account := mock.GetSampleAccount(true, true, true)

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.CreateAccount(ctx, account, &models.ComplianceAuditLog{})
		require.Error(err, "an error was expected")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected an ErrNoIDOnCreate error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})
}

func (s *storeTestSuite) TestLookupAccount() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")

		prevMod := account.Modified
		newFirstName := sql.NullString{String: account.FirstName.String + "extrastuff", Valid: true}
		account.FirstName = newFirstName
		newLastName := sql.NullString{String: account.LastName.String + "extrastuff", Valid: true}
		account.LastName = newLastName

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) + 1

		//test
		err = s.store.UpdateAccount(ctx, account, &models.ComplianceAuditLog{})
		require.NoError(err, "expected no error")

		account = nil
		account, err = s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")
		require.Equal(newFirstName, account.FirstName)
		require.Equal(newLastName, account.LastName)
		require.True(prevMod.Before(account.Modified), "expected the modified time to be newer")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureZeroID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		account.ID = ulid.Zero

		//test
		err = s.store.UpdateAccount(ctx, account, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrMissingID, err, "expected an ErrMissingID error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.NoError(err, "expected no error")

		account.ID = ulid.MakeSecure()

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.UpdateAccount(ctx, account, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})
}

func (s *storeTestSuite) TestDeleteAccount() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) + 1

		//test
		err = s.store.DeleteAccount(ctx, accountId, &models.ComplianceAuditLog{})
		require.NoError(err, "expected no error")

		account, err := s.store.RetrieveAccount(ctx, accountId)
		require.Nil(account, "account should be nil")
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureNotFound", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MakeSecure()

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // no extras expected

		//test
		err = s.store.DeleteAccount(ctx, accountId, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})
}

func (s *storeTestSuite) TestListAccountTransactions() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")

		//test
		require.Panics(func() { s.store.ListAccountTransactions(ctx, accountId, nil) }, "should panic with nil page info")
	})
}

func (s *storeTestSuite) TestListCryptoAddresses() {
	s.Run("SuccessOne", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")
		cryptoAddress := mock.GetSampleCryptoAddress(accountId)

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) + 1

		//test
		addresses, err := s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no error")
		require.NotNil(addresses, "addresses should not be nil")
		require.Len(addresses.CryptoAddresses, 2, fmt.Sprintf("expected 2 crypto addresses, got %d", len(addresses.CryptoAddresses)))

		err = s.store.CreateCryptoAddress(ctx, cryptoAddress, &models.ComplianceAuditLog{})
		require.NoError(err, "no error was expected")

		addresses, err = s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no error")
		require.NotNil(addresses, "addresses should not be nil")
		require.Len(addresses.CryptoAddresses, 3, fmt.Sprintf("expected 3 crypto addresses, got %d", len(addresses.CryptoAddresses)))

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureNotFoundAccountID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MakeSecure()
		cryptoAddress := mock.GetSampleCryptoAddress(accountId)

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.CreateCryptoAddress(ctx, cryptoAddress, &models.ComplianceAuditLog{})
		require.Error(err, "an error was expected")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected error ErrAlreadyExists")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureZeroAccountID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.Zero
		cryptoAddress := mock.GetSampleCryptoAddress(accountId)

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.CreateCryptoAddress(ctx, cryptoAddress, &models.ComplianceAuditLog{})
		require.Error(err, "an error was expected")
		require.Equal(errors.ErrMissingReference, err, "expected error ErrMissingReference")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureAddressNotZeroID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MakeSecure()
		cryptoAddress := mock.GetSampleCryptoAddress(accountId)
		cryptoAddress.ID = ulid.MakeSecure()

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.CreateCryptoAddress(ctx, cryptoAddress, &models.ComplianceAuditLog{})
		require.Error(err, "an error was expected")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected error ErrNoIDOnCreate")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})
}

func (s *storeTestSuite) TestRetrieveCryptoAddress() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
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
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		prevMod := address.Modified
		newNetwork := "LTC"
		if address.Network == "LTC" {
			newNetwork = "BTC"
		}
		address.Network = newNetwork

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) + 1

		//test
		err = s.store.UpdateCryptoAddress(ctx, address, &models.ComplianceAuditLog{})
		require.NoError(err, "expected no error")

		address = nil
		address, err = s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")
		require.Equal(newNetwork, address.Network)
		require.True(prevMod.Before(address.Modified), "expected the modified time to be newer")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureZeroID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		address.ID = ulid.Zero

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.UpdateCryptoAddress(ctx, address, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrMissingID, err, "expected an ErrMissingID error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureZeroAccountID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		address.AccountID = ulid.Zero

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.UpdateCryptoAddress(ctx, address, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrMissingReference, err, "expected an ErrMissingReference error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureNotFoundAddress", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		address.ID = ulid.MakeSecure()

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.UpdateCryptoAddress(ctx, address, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureNotFoundAccount", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6QS6AK4KNS46Q9HEB7DTPR")
		addressId := ulid.MustParse("01HV6QS6AK4KNS46Q9HFHBEQAP")
		address, err := s.store.RetrieveCryptoAddress(ctx, accountId, addressId)
		require.NoError(err, "expected no error")

		address.AccountID = ulid.MakeSecure()

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.UpdateCryptoAddress(ctx, address, &models.ComplianceAuditLog{})
		require.Error(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected an ErrNotFound error")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})
}

func (s *storeTestSuite) TestDeleteCryptoAddress() {
	s.Run("Success", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")
		addressId := ulid.MustParse("01HV6RV08YNR2GH8MEEKB7DH2W")

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) + 1

		//test
		cryptoAddresses, err := s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(cryptoAddresses.CryptoAddresses, "there were no crypto addresses")
		require.Len(cryptoAddresses.CryptoAddresses, 2, fmt.Sprintf("there should be 2 crypto addresses, but there were %d", len(cryptoAddresses.CryptoAddresses)))

		err = s.store.DeleteCryptoAddress(ctx, accountId, addressId, &models.ComplianceAuditLog{})
		require.Nil(err, "expected no error")

		cryptoAddresses, err = s.store.ListCryptoAddresses(ctx, accountId, nil)
		require.NoError(err, "expected no errors")
		require.NotNil(cryptoAddresses.CryptoAddresses, "there were no crypto addresses")
		require.Len(cryptoAddresses.CryptoAddresses, 1, fmt.Sprintf("there should be 1 crypto address, but there were %d", len(cryptoAddresses.CryptoAddresses)))

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureNotFoundAccountID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MakeSecure()
		addressId := ulid.MustParse("01HV6RV08YNR2GH8MEEKB7DH2W")

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.DeleteCryptoAddress(ctx, accountId, addressId, &models.ComplianceAuditLog{})
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})

	s.Run("FailureNotFoundAddressID", func() {
		//setup
		require := s.Require()
		ctx := s.ActorContext()
		accountId := ulid.MustParse("01HV6RV08YNR2GH8MEEFCV4NKN")
		addressId := ulid.MakeSecure()

		//count audit logs before
		logs, err := s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		logsExp := len(logs.Logs) // expect no extras

		//test
		err = s.store.DeleteCryptoAddress(ctx, accountId, addressId, &models.ComplianceAuditLog{})
		require.NotNil(err, "expected an error")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")

		//check for audit log creation
		logs, err = s.store.ListComplianceAuditLogs(ctx, nil)
		require.NoError(err, "error getting logs")
		require.NotNil(logs, "logs was nil")
		require.Lenf(logs.Logs, logsExp, "expected %d logs, got %d", logsExp, len(logs.Logs))
	})
}
