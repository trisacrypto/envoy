package sqlite_test

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
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
		Source:             "local",
		Status:             "pending",
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
