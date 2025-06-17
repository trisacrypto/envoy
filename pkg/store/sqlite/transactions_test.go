package sqlite_test

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/mock"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"go.rtnl.ai/ulid"
)

func (s *storeTestSuite) TestListTransactions() {
	require := s.Require()
	ctx := context.Background()
	query := &models.TransactionPageInfo{}

	page, err := s.store.ListTransactions(ctx, query)
	require.NoError(err, "could not list transactions from database")
	require.NotNil(page, "a nil page was returned without transactions")
	require.Len(page.Transactions, 4, "expected transactions to be returned in list")
	require.Empty(page.Page.NextPageID, "expected next page ID to be empty")
	require.Empty(page.Page.PrevPageID, "expected prev page Id to be empty")
	require.Equal(uint32(50), page.Page.PageSize, "expected the default page size to be returned")

	// Ensure secure envelopes are counted correctly
	for i, tx := range page.Transactions {
		switch i {
		case 1, 3:
			require.Equal(int64(2), tx.NumEnvelopes())
		case 2:
			require.Equal(int64(4), tx.NumEnvelopes())
		case 0:
			require.Equal(int64(0), tx.NumEnvelopes())
		}
	}
}

func (s *storeTestSuite) TestCreateTransaction() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txn := mock.GetSampleTransaction(true, true, false)

		txn.ID = uuid.Nil
		txn.CounterpartyID = ulid.NullULID{ULID: ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ"), Valid: true}

		txns, err := s.store.ListTransactions(ctx, &models.TransactionPageInfo{})
		require.NoError(err, "expected no error when listing transactions")
		require.NotNil(txns, "expected a non-nil transactions page")
		expectedLen := len(txns.Transactions) + 1

		//test
		err = s.store.CreateTransaction(ctx, txn)
		require.NoError(err, "expected no error when creating transaction")

		txns, err = s.store.ListTransactions(ctx, &models.TransactionPageInfo{})
		require.NoError(err, "expected no error when listing transactions")
		require.NotNil(txns, "expected a non-nil transactions page")
		require.Len(txns.Transactions, expectedLen, fmt.Sprintf("expected %d transactions, got %d", expectedLen, len(txns.Transactions)))
	})

	s.Run("FailureNonZeroUUID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txn := mock.GetSampleTransaction(true, true, false)
		txn.CounterpartyID = ulid.NullULID{ULID: ulid.MustParse("01JXTQCDE6ZES5MPXNW7K19QVQ"), Valid: true}

		txns, err := s.store.ListTransactions(ctx, &models.TransactionPageInfo{})
		require.NoError(err, "expected no error when listing transactions")
		require.NotNil(txns, "expected a non-nil transactions page")
		expectedLen := len(txns.Transactions)

		//test
		err = s.store.CreateTransaction(ctx, txn)
		require.Error(err, "expected an error when creating transaction")
		require.Equal(errors.ErrNoIDOnCreate, err, "expected ErrNoIDOnCreate")

		txns, err = s.store.ListTransactions(ctx, &models.TransactionPageInfo{})
		require.NoError(err, "expected no error when listing transactions")
		require.NotNil(txns, "expected a non-nil transactions page")
		require.Len(txns.Transactions, expectedLen, fmt.Sprintf("expected %d transactions, got %d", expectedLen, len(txns.Transactions)))
	})

	s.Run("FailureUnknownCounterparty", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txn := mock.GetSampleTransaction(true, true, false)
		txn.ID = uuid.Nil

		txns, err := s.store.ListTransactions(ctx, &models.TransactionPageInfo{})
		require.NoError(err, "expected no error when listing transactions")
		require.NotNil(txns, "expected a non-nil transactions page")
		expectedLen := len(txns.Transactions)

		//test
		err = s.store.CreateTransaction(ctx, txn)
		require.Error(err, "expected an error when creating transaction")
		// TODO: (ticket sc-32339) this currently returns an ErrAlreadyExists
		// instead of an ErrNotFound as would be logical, because in the `dbe()`
		// function we return an ErrAlreadyExists for any SQLite constraint error
		require.Equal(errors.ErrAlreadyExists, err, "expected ErrAlreadyExists")

		txns, err = s.store.ListTransactions(ctx, &models.TransactionPageInfo{})
		require.NoError(err, "expected no error when listing transactions")
		require.NotNil(txns, "expected a non-nil transactions page")
		require.Len(txns.Transactions, expectedLen, fmt.Sprintf("expected %d transactions, got %d", expectedLen, len(txns.Transactions)))
	})
}

func (s *storeTestSuite) TestRetrieveTransaction() {
	s.Run("Success", func() {
		//FIXME: fails to load binary data from envelopes
		s.T().SkipNow()

		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.MustParse("2c891c75-14fa-4c71-aa07-6405b98db7a3")

		//test
		txn, err := s.store.RetrieveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when retrieving transaction")
		require.NotNil(txn, "expected a non-nil transaction")

		require.Equal("01HWR5VWW8V7ZFFVJVBEC7AV8A", txn.CounterpartyID.ULID.String(), "expected a different counterparty ID")
		require.Equal(4, txn.NumEnvelopes(), fmt.Sprintf("expected 4 envelopes, got %d", txn.NumEnvelopes()))
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.New()

		//test
		txn, err := s.store.RetrieveTransaction(ctx, txnId)
		require.Error(err, "expected an error when retrieving transaction")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(txn, "expected a nil transaction")
	})

	s.Run("FailureNotFoundNilID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.Nil

		//test
		txn, err := s.store.RetrieveTransaction(ctx, txnId)
		require.Error(err, "expected an error when retrieving transaction")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(txn, "expected a nil transaction")
	})
}

func (s *storeTestSuite) TestUpdateTransaction() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")
		txn, err := s.store.RetrieveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when retrieving transaction")
		require.NotNil(txn, "expected a non-nil transaction")

		prevUpd := txn.LastUpdate
		prevMod := txn.Modified
		newAmount := 808.808
		txn.Amount = newAmount

		//test
		err = s.store.UpdateTransaction(ctx, txn)
		require.NoError(err, "expected no error when updating transaction")

		txn = nil
		txn, err = s.store.RetrieveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when retrieving transaction")
		require.NotNil(txn, "expected a non-nil transaction")
		require.Equal(newAmount, txn.Amount, "transaction amount did not update")
		require.True(prevUpd.Time.Equal(txn.LastUpdate.Time), "expected the last update time to be the same")
		require.True(prevMod.Before(txn.Modified), "expected the modified time to be newer")
	})

	s.Run("FailureNotFound", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")
		transaction, err := s.store.RetrieveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when retrieving transaction")
		require.NotNil(transaction, "expected a non-nil transaction")

		transaction.ID = uuid.New()

		//test
		err = s.store.UpdateTransaction(ctx, transaction)
		require.Error(err, "expected an error when updating transaction")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureNilUUID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")
		transaction, err := s.store.RetrieveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when retrieving transaction")
		require.NotNil(transaction, "expected a non-nil transaction")

		transaction.ID = uuid.Nil

		//test
		err = s.store.UpdateTransaction(ctx, transaction)
		require.Error(err, "expected an error when updating transaction")
		require.Equal(errors.ErrMissingID, err, "expected ErrMissingID")
	})
}

func (s *storeTestSuite) TestDeleteTransaction() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")

		//test
		err := s.store.DeleteTransaction(ctx, txnId)
		require.NoError(err, "expected no error when deleting transaction")

		txn, err := s.store.RetrieveTransaction(ctx, txnId)
		require.Error(err, "expected an error when retrieving deleted txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.Nil(txn, "expected a nil txn")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.New()

		//test
		err := s.store.DeleteTransaction(ctx, txnId)
		require.Error(err, "expected an error when deleting txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureNotFoundNilID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.Nil

		//test
		err := s.store.DeleteTransaction(ctx, txnId)
		require.Error(err, "expected an error when deleting txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestArchiveUnarchiveTransaction() {
	s.Run("SuccessArchiveThenUnarchive", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")
		beforeTxn := time.Now()

		//test ArchiveTransaction
		err := s.store.ArchiveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when deleting transaction")

		txn, err := s.store.RetrieveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when retrieving txn")
		require.NotNil(txn, "expected a non-nil txn")
		require.True(txn.Archived, "expected transaction to be archived")
		require.True(txn.ArchivedOn.Valid, "expected a valid archived timestamp")
		require.True(beforeTxn.Before(txn.ArchivedOn.Time), "expected archived timestamp to be more recent")
		require.True(beforeTxn.Before(txn.Modified), "expected modified timestamp to be more recent")

		//test UnarchiveTransaction
		err = s.store.UnarchiveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when deleting transaction")

		txn = nil
		txn, err = s.store.RetrieveTransaction(ctx, txnId)
		require.NoError(err, "expected no error when retrieving txn")
		require.NotNil(txn, "expected a non-nil txn")
		require.False(txn.Archived, "expected transaction to be unarchived")
		require.False(txn.ArchivedOn.Valid, "expected an invalid archived timestamp")
		require.True(beforeTxn.Before(txn.Modified), "expected modified timestamp to be more recent")
	})

	s.Run("FailureArchiveNotFoundRandomID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.New()

		//test
		err := s.store.ArchiveTransaction(ctx, txnId)
		require.Error(err, "expected an error when archiving txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureArchiveNotFoundNilID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.Nil

		//test
		err := s.store.ArchiveTransaction(ctx, txnId)
		require.Error(err, "expected an error when archiving txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureUnarchiveNotFoundRandomID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.New()

		//test
		err := s.store.UnarchiveTransaction(ctx, txnId)
		require.Error(err, "expected an error when unarchiving txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})

	s.Run("FailureUnarchiveNotFoundNilID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.Nil

		//test
		err := s.store.UnarchiveTransaction(ctx, txnId)
		require.Error(err, "expected an error when unarchiving txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
	})
}

func (s *storeTestSuite) TestCountTransactions() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		expected := &models.TransactionCounts{
			Active: map[string]int{
				"completed": 1,
				"draft":     1,
				"pending":   1,
				"review":    1,
			},
			Archived: map[string]int{
				"draft": 1,
			},
		}

		//test
		counts, err := s.store.CountTransactions(ctx)
		require.NoError(err, "expected no error when counting transactions")
		require.NotNil(counts, "expected counts to be non-nil")
		require.Equal(expected, counts, "expected different counts")
	})
}

func (s *storeTestSuite) TestTransactionState() {
	s.Run("Success", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.MustParse("b04dc71c-7214-46a5-a514-381ef0bcc494")

		//test
		archived, status, err := s.store.TransactionState(ctx, txnId)
		require.NoError(err, "expected no error when checking txn state")
		require.False(archived, "expected an unarchived txn")
		require.Equal(enum.StatusDraft, status, "expected a 'draft' txn")
	})

	s.Run("FailureNotFoundRandomID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.New()

		//test
		archived, status, err := s.store.TransactionState(ctx, txnId)
		require.Error(err, "expected an error when deleting txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.False(archived, "expected an unarchived txn")
		require.Equal(enum.Status(0), status, "expected an zero txn status for an error")
	})

	s.Run("FailureNotFoundNilID", func() {
		//setup
		ctx := context.Background()
		require := s.Require()
		txnId := uuid.Nil

		//test
		archived, status, err := s.store.TransactionState(ctx, txnId)
		require.Error(err, "expected an error when deleting txn")
		require.Equal(errors.ErrNotFound, err, "expected ErrNotFound")
		require.False(archived, "expected an unarchived txn")
		require.Equal(enum.Status(0), status, "expected an zero txn status for an error")
	})
}

func (s *storeTestSuite) TestListSecureEnvelopes() {
	//FIXME: do this test once we can make secure envelope fixtures that are valid
	s.T().SkipNow()
}

func (s *storeTestSuite) TestCreateSecureEnvelope() {
	//FIXME: do this test once we can make secure envelope fixtures that are valid
	s.T().SkipNow()
}

func (s *storeTestSuite) TestRetrieveSecureEnvelope() {
	//FIXME: do this test once we can make secure envelope fixtures that are valid
	s.T().SkipNow()
}

func (s *storeTestSuite) TestUpdateSecureEnvelope() {
	//FIXME: do this test once we can make secure envelope fixtures that are valid
	s.T().SkipNow()
}

func (s *storeTestSuite) TestDeleteSecureEnvelope() {
	//FIXME: do this test once we can make secure envelope fixtures that are valid
	s.T().SkipNow()
}

func (s *storeTestSuite) TestLatestSecureEnvelope() {
	//FIXME: do this test once we can make secure envelope fixtures that are valid
	s.T().SkipNow()
}

func (s *storeTestSuite) TestLatestPayloadEnvelope() {
	//FIXME: do this test once we can make secure envelope fixtures that are valid
	s.T().SkipNow()
}

func (s *storeTestSuite) TestPreparedTransaction_Created() {
	defer s.ResetDB()
	require := s.Require()
	ctx := context.Background()

	envelopeID := uuid.New()
	db, err := s.store.PrepareTransaction(ctx, envelopeID)
	require.NoError(err, "could not start prepared transaction")
	defer db.Rollback()

	// Created should be true since the transaction has a new UUID
	require.True(db.Created(), "expected the transaction to be created in database")

	// Should be able to add a counterparty with TRISA information
	counterparty := &models.Counterparty{
		DirectoryID:         sql.NullString{Valid: true, String: "2666abb0-5e92-4d02-a9ba-5539323e9683"},
		RegisteredDirectory: sql.NullString{Valid: true, String: "trisatest.dev"},
	}

	err = db.AddCounterparty(counterparty)
	require.NoError(err, "could not add counterparty to database")

	// Should be able to update the transaction record
	record := &models.Transaction{
		Source:             enum.SourceLocal,
		Status:             enum.StatusPending,
		Originator:         sql.NullString{Valid: true, String: "Alessia Cremonesi"},
		OriginatorAddress:  sql.NullString{Valid: true, String: "mrfAEzGzK23kU23FxrToDRPmV1ReNfX43G"},
		Beneficiary:        sql.NullString{Valid: true, String: "Alesia Sosa Calvillo"},
		BeneficiaryAddress: sql.NullString{Valid: true, String: "n3Vgn8wF6ZkpKSe186NnytLPXdZ6j1JbHg"},
		VirtualAsset:       "LTC",
		Amount:             0.46602501,
		LastUpdate:         sql.NullTime{Valid: true, Time: time.Now()},
	}
	err = db.Update(record)
	require.NoError(err, "could not update record in database")

	// Add a secure envelope to the transaction
	payload, err := loadPayload("testdata/identity.pb.json", "testdata/transaction.pb.json")
	require.NoError(err, "could not create payload for secure envelope")

	env, err := envelope.New(payload, envelope.WithEnvelopeID(envelopeID.String()))
	require.NoError(err, "could not create envelope from payload")

	env, _, err = env.Encrypt()
	require.NoError(err, "cannot encrypt envelope")

	err = db.AddEnvelope(models.FromEnvelope(env))
	require.NoError(err, "could not add envelope to transactio")

	require.NoError(db.Commit(), "could not commit transaction to database")

	// Transaction and secure envelopes should be in database after commit
	_, err = s.store.RetrieveTransaction(ctx, envelopeID)
	require.NoError(err, "could not retrieve transaction from database")

	page, err := s.store.ListSecureEnvelopes(ctx, envelopeID, nil)
	require.NoError(err, "could not retrieve secure envelopes from database")
	require.NotNil(page)
	require.Len(page.Envelopes, 1, "expected one envelopes returned")
}

func (s *storeTestSuite) TestPreparedTransaction_Exists() {
	defer s.ResetDB()
	require := s.Require()
	ctx := context.Background()

	envelopeID := uuid.MustParse("c20a7cdf-5c23-4b44-b7cd-a29cd00761a3")
	db, err := s.store.PrepareTransaction(ctx, envelopeID)
	require.NoError(err, "could not start prepared transaction")
	defer db.Rollback()

	// Created should be false since the transaction is in the database
	require.False(db.Created(), "expected the transaction to be already existing in database")
}
