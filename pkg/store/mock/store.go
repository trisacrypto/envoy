package mock

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/dsn"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/store/txn"

	"github.com/google/uuid"
	"go.rtnl.ai/ulid"
)

// Store implements the store.Store interface with callback functions that the tester
// can specify to simulate a specific behavior. The Store is not thread-safe and one
// mock store should be used per test.
type Store struct {
	callbacks map[string]any
	calls     map[string]int
	readonly  bool
}

// Open a new mock store. Generally, the nil uri can be used to create the mock;
// otherwise a DSN with the mock scheme can be used.
func Open(uri *dsn.DSN) (*Store, error) {
	if uri != nil && uri.Scheme != dsn.Mock {
		return nil, errors.ErrUnknownScheme
	}

	if uri == nil {
		uri = &dsn.DSN{ReadOnly: false, Scheme: dsn.Mock}
	}

	return &Store{
		callbacks: make(map[string]any),
		calls:     make(map[string]int),
		readonly:  uri.ReadOnly,
	}, nil
}

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Reset all the calls and callbacks in the store.
func (s *Store) Reset() {
	// Set maps to nil to free up memory
	s.calls = nil
	s.callbacks = nil

	// Create new calls and callbacks maps
	s.calls = make(map[string]int)
	s.callbacks = make(map[string]any)
}

// Assert that the expected number of calls were made to the given method.
func (s *Store) AssertCalls(t testing.TB, method string, expected int) {
	require.Equal(t, expected, s.calls[method], "expected %d calls to %s, got %d", expected, method, s.calls[method])
}

//===========================================================================
// Store Interface Methods
//===========================================================================

// Set a callback for when "Close()" is called on the mock store.
func (s *Store) OnClose(fn func() error) {
	s.callbacks["Close"] = fn
}

// If present, calls the callback previously set for "Close()" (set the callback
// with "OnClose()"), otherwise returns `nil` indicating success.
func (s *Store) Close() error {
	s.calls["Close"]++
	// perform callback if there is one
	if fn, ok := s.callbacks["Close"]; ok {
		return fn.(func() error)()
	}
	// return no error
	return nil
}

// Set a callback for when "Begin()" is called on the mock store.
func (s *Store) OnBegin(fn func(context.Context, *sql.TxOptions) (txn.Txn, error)) {
	s.callbacks["Begin"] = fn
}

// If present, calls the callback previously set for "Begin()" (set the callback
// with "OnBegin()"), otherwise returns a `Txn` mock.
func (s *Store) Begin(ctx context.Context, opts *sql.TxOptions) (txn.Txn, error) {
	s.calls["Begin"]++

	// perform callback if there is one
	if fn, ok := s.callbacks["Close"]; ok {
		return fn.(func(context.Context, *sql.TxOptions) (txn.Txn, error))(ctx, opts)
	}

	// make sure readonly option matches the store setting
	if opts == nil {
		opts = &sql.TxOptions{ReadOnly: s.readonly}
	} else if s.readonly && !opts.ReadOnly {
		return nil, errors.ErrReadOnly
	}

	// return a mock for the Txn interface
	return &Tx{
		opts:      opts,
		callbacks: make(map[string]any),
		calls:     make(map[string]int),
	}, nil
}

//===========================================================================
// Transaction Store Methods
//===========================================================================

// Set a callback for when "ListTransactions()" is called on the mock store.
func (s *Store) OnListTransactions(fn func(ctx context.Context, in *models.TransactionPageInfo) (*models.TransactionPage, error)) {
	s.callbacks["ListTransactions"] = fn
}

// Calls the callback previously set for "ListTransactions()" (set the callback with "OnListTransactions()")
func (s *Store) ListTransactions(ctx context.Context, in *models.TransactionPageInfo) (*models.TransactionPage, error) {
	s.calls["ListTransactions"]++
	if fn, ok := s.callbacks["ListTransactions"]; ok {
		return fn.(func(ctx context.Context, in *models.TransactionPageInfo) (*models.TransactionPage, error))(ctx, in)
	}
	panic("ListTransactions callback not set")
}

// Set a callback for when "CreateTransaction()" is called on the mock store.
func (s *Store) OnCreateTransaction(fn func(ctx context.Context, in *models.Transaction) error) {
	s.callbacks["CreateTransaction"] = fn
}

// Calls the callback previously set for "CreateTransaction()" (set the callback with "OnCreateTransaction()")
func (s *Store) CreateTransaction(ctx context.Context, in *models.Transaction) error {
	s.calls["CreateTransaction"]++
	if fn, ok := s.callbacks["CreateTransaction"]; ok {
		return fn.(func(ctx context.Context, in *models.Transaction) error)(ctx, in)
	}
	panic("CreateTransaction callback not set")
}

// Set a callback for when "RetrieveTransaction()" is called on the mock store.
func (s *Store) OnRetrieveTransaction(fn func(ctx context.Context, id uuid.UUID) (*models.Transaction, error)) {
	s.callbacks["RetrieveTransaction"] = fn
}

// Calls the callback previously set for "RetrieveTransaction()" (set the callback with "OnRetrieveTransaction()")
func (s *Store) RetrieveTransaction(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	s.calls["RetrieveTransaction"]++
	if fn, ok := s.callbacks["RetrieveTransaction"]; ok {
		return fn.(func(ctx context.Context, id uuid.UUID) (*models.Transaction, error))(ctx, id)
	}
	panic("RetrieveTransaction callback not set")
}

// Set a callback for when "UpdateTransaction()" is called on the mock store.
func (s *Store) OnUpdateTransaction(fn func(ctx context.Context, in *models.Transaction) error) {
	s.callbacks["UpdateTransaction"] = fn
}

// Calls the callback previously set for "UpdateTransaction()" (set the callback with "OnUpdateTransaction()")
func (s *Store) UpdateTransaction(ctx context.Context, in *models.Transaction) error {
	s.calls["UpdateTransaction"]++
	if fn, ok := s.callbacks["UpdateTransaction"]; ok {
		return fn.(func(ctx context.Context, in *models.Transaction) error)(ctx, in)
	}
	panic("UpdateTransaction callback not set")
}

// Set a callback for when "DeleteTransaction()" is called on the mock store.
func (s *Store) OnDeleteTransaction(fn func(ctx context.Context, id uuid.UUID) error) {
	s.callbacks["DeleteTransaction"] = fn
}

// Calls the callback previously set for "DeleteTransaction()" (set the callback with "OnDeleteTransaction()")
func (s *Store) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["DeleteTransaction"]++
	if fn, ok := s.callbacks["DeleteTransaction"]; ok {
		return fn.(func(ctx context.Context, id uuid.UUID) error)(ctx, id)
	}
	panic("DeleteTransaction callback not set")
}

// Set a callback for when "ArchiveTransaction()" is called on the mock store.
func (s *Store) OnArchiveTransaction(fn func(ctx context.Context, id uuid.UUID) error) {
	s.callbacks["ArchiveTransaction"] = fn
}

// Calls the callback previously set for "ArchiveTransaction()" (set the callback with "OnArchiveTransaction()")
func (s *Store) ArchiveTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["ArchiveTransaction"]++
	if fn, ok := s.callbacks["ArchiveTransaction"]; ok {
		return fn.(func(ctx context.Context, id uuid.UUID) error)(ctx, id)
	}
	panic("ArchiveTransaction callback not set")
}

// Set a callback for when "UnarchiveTransaction()" is called on the mock store.
func (s *Store) OnUnarchiveTransaction(fn func(ctx context.Context, id uuid.UUID) error) {
	s.callbacks["UnarchiveTransaction"] = fn
}

// Calls the callback previously set for "UnarchiveTransaction()" (set the callback with "OnUnarchiveTransaction()")
func (s *Store) UnarchiveTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["UnarchiveTransaction"]++
	if fn, ok := s.callbacks["UnarchiveTransaction"]; ok {
		return fn.(func(ctx context.Context, id uuid.UUID) error)(ctx, id)
	}
	panic("UnarchiveTransaction callback not set")
}

// Set a callback for when "CountTransactions()" is called on the mock store.
func (s *Store) OnCountTransactions(fn func(ctx context.Context) (*models.TransactionCounts, error)) {
	s.callbacks["CountTransactions"] = fn
}

// Calls the callback previously set for "CountTransactions()" (set the callback with "OnCountTransactions()")
func (s *Store) CountTransactions(ctx context.Context) (*models.TransactionCounts, error) {
	s.calls["CountTransactions"]++
	if fn, ok := s.callbacks["CountTransactions"]; ok {
		return fn.(func(ctx context.Context) (*models.TransactionCounts, error))(ctx)
	}
	panic("CountTransactions callback not set")
}

// Set a callback for when "PrepareTransaction()" is called on the mock store.
func (s *Store) OnPrepareTransaction(fn func(ctx context.Context, id uuid.UUID) (models.PreparedTransaction, error)) {
	s.callbacks["PrepareTransaction"] = fn
}

// Calls the callback previously set for "PrepareTransaction()" (set the callback with "OnPrepareTransaction()")
func (s *Store) PrepareTransaction(ctx context.Context, id uuid.UUID) (models.PreparedTransaction, error) {
	s.calls["PrepareTransaction"]++
	if fn, ok := s.callbacks["PrepareTransaction"]; ok {
		return fn.(func(ctx context.Context, id uuid.UUID) (models.PreparedTransaction, error))(ctx, id)
	}
	// if no callback is set, return a mock PreparedTransaction
	return &PreparedTransaction{
		callbacks: make(map[string]any),
		calls:     make(map[string]int),
	}, nil
}

// Set a callback for when "TransactionState()" is called on the mock store.
func (s *Store) OnTransactionState(fn func(ctx context.Context, id uuid.UUID) (bool, enum.Status, error)) {
	s.callbacks["TransactionState"] = fn
}

// Calls the callback previously set for "TransactionState()" (set the callback with "OnTransactionState()")
func (s *Store) TransactionState(ctx context.Context, id uuid.UUID) (bool, enum.Status, error) {
	s.calls["TransactionState"]++
	if fn, ok := s.callbacks["TransactionState"]; ok {
		return fn.(func(ctx context.Context, id uuid.UUID) (bool, enum.Status, error))(ctx, id)
	}
	panic("TransactionState callback not set")
}

//===========================================================================
// SecureEnvelope Store Methods
//===========================================================================

// Set a callback for when "ListSecureEnvelopes()" is called on the mock store.
func (s *Store) OnListSecureEnvelopes(fn func(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error)) {
	s.callbacks["ListSecureEnvelopes"] = fn
}

// Calls the callback previously set for "ListSecureEnvelopes()" (set the callback with "OnListSecureEnvelopes()")
func (s *Store) ListSecureEnvelopes(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	s.calls["ListSecureEnvelopes"]++
	if fn, ok := s.callbacks["ListSecureEnvelopes"]; ok {
		return fn.(func(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error))(ctx, txID, page)
	}
	panic("ListSecureEnvelopes callback not set")
}

// Set a callback for when "CreateSecureEnvelope()" is called on the mock store.
func (s *Store) OnCreateSecureEnvelope(fn func(ctx context.Context, in *models.SecureEnvelope) error) {
	s.callbacks["CreateSecureEnvelope"] = fn
}

// Calls the callback previously set for "CreateSecureEnvelope()" (set the callback with "OnCreateSecureEnvelope()")
func (s *Store) CreateSecureEnvelope(ctx context.Context, in *models.SecureEnvelope) error {
	s.calls["CreateSecureEnvelope"]++
	if fn, ok := s.callbacks["CreateSecureEnvelope"]; ok {
		return fn.(func(ctx context.Context, in *models.SecureEnvelope) error)(ctx, in)
	}
	panic("CreateSecureEnvelope callback not set")
}

// Set a callback for when "RetrieveSecureEnvelope()" is called on the mock store.
func (s *Store) OnRetrieveSecureEnvelope(fn func(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error)) {
	s.callbacks["RetrieveSecureEnvelope"] = fn
}

// Calls the callback previously set for "RetrieveSecureEnvelope()" (set the callback with "OnRetrieveSecureEnvelope()")
func (s *Store) RetrieveSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	s.calls["RetrieveSecureEnvelope"]++
	if fn, ok := s.callbacks["RetrieveSecureEnvelope"]; ok {
		return fn.(func(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error))(ctx, txID, envID)
	}
	panic("RetrieveSecureEnvelope callback not set")
}

// Set a callback for when "UpdateSecureEnvelope()" is called on the mock store.
func (s *Store) OnUpdateSecureEnvelope(fn func(ctx context.Context, in *models.SecureEnvelope) error) {
	s.callbacks["UpdateSecureEnvelope"] = fn
}

// Calls the callback previously set for "UpdateSecureEnvelope()" (set the callback with "OnUpdateSecureEnvelope()")
func (s *Store) UpdateSecureEnvelope(ctx context.Context, in *models.SecureEnvelope) error {
	s.calls["UpdateSecureEnvelope"]++
	if fn, ok := s.callbacks["UpdateSecureEnvelope"]; ok {
		return fn.(func(ctx context.Context, in *models.SecureEnvelope) error)(ctx, in)
	}
	panic("UpdateSecureEnvelope callback not set")
}

// Set a callback for when "DeleteSecureEnvelope()" is called on the mock store.
func (s *Store) OnDeleteSecureEnvelope(fn func(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error) {
	s.callbacks["DeleteSecureEnvelope"] = fn
}

// Calls the callback previously set for "DeleteSecureEnvelope()" (set the callback with "OnDeleteSecureEnvelope()")
func (s *Store) DeleteSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error {
	s.calls["DeleteSecureEnvelope"]++
	if fn, ok := s.callbacks["DeleteSecureEnvelope"]; ok {
		return fn.(func(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error)(ctx, txID, envID)
	}
	panic("DeleteSecureEnvelope callback not set")
}

// Set a callback for when "LatestSecureEnvelope()" is called on the mock store.
func (s *Store) OnLatestSecureEnvelope(fn func(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)) {
	s.callbacks["LatestSecureEnvelope"] = fn
}

// Calls the callback previously set for "LatestSecureEnvelope()" (set the callback with "OnLatestSecureEnvelope()")
func (s *Store) LatestSecureEnvelope(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	s.calls["LatestSecureEnvelope"]++
	if fn, ok := s.callbacks["LatestSecureEnvelope"]; ok {
		return fn.(func(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error))(ctx, txID, direction)
	}
	panic("LatestSecureEnvelope callback not set")
}

// Set a callback for when "LatestPayloadEnvelope()" is called on the mock store.
func (s *Store) OnLatestPayloadEnvelope(fn func(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)) {
	s.callbacks["LatestPayloadEnvelope"] = fn
}

// Calls the callback previously set for "LatestPayloadEnvelope()" (set the callback with "OnLatestPayloadEnvelope()")
func (s *Store) LatestPayloadEnvelope(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	s.calls["LatestPayloadEnvelope"]++
	if fn, ok := s.callbacks["LatestPayloadEnvelope"]; ok {
		return fn.(func(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error))(ctx, txID, direction)
	}
	panic("LatestPayloadEnvelope callback not set")
}

//===========================================================================
// Account Store Methods
//===========================================================================

// Set a callback for when "ListAccounts()" is called on the mock store.
func (s *Store) OnListAccounts(fn func(ctx context.Context, in *models.PageInfo) (*models.AccountsPage, error)) {
	s.callbacks["ListAccounts"] = fn
}

// Calls the callback previously set for "ListAccounts()" (set the callback with "OnListAccounts()")
func (s *Store) ListAccounts(ctx context.Context, in *models.PageInfo) (*models.AccountsPage, error) {
	s.calls["ListAccounts"]++
	if fn, ok := s.callbacks["ListAccounts"]; ok {
		return fn.(func(ctx context.Context, in *models.PageInfo) (*models.AccountsPage, error))(ctx, in)
	}
	panic("ListAccounts callback not set")
}

// Set a callback for when "CreateAccount()" is called on the mock store.
func (s *Store) OnCreateAccount(fn func(ctx context.Context, in *models.Account) error) {
	s.callbacks["CreateAccount"] = fn
}

// Calls the callback previously set for "CreateAccount()" (set the callback with "OnCreateAccount()")
func (s *Store) CreateAccount(ctx context.Context, in *models.Account) error {
	s.calls["CreateAccount"]++
	if fn, ok := s.callbacks["CreateAccount"]; ok {
		return fn.(func(ctx context.Context, in *models.Account) error)(ctx, in)
	}
	panic("CreateAccount callback not set")
}

// Set a callback for when "LookupAccount()" is called on the mock store.
func (s *Store) OnLookupAccount(fn func(ctx context.Context, cryptoAddress string) (*models.Account, error)) {
	s.callbacks["LookupAccount"] = fn
}

// Calls the callback previously set for "LookupAccount()" (set the callback with "OnLookupAccount()")
func (s *Store) LookupAccount(ctx context.Context, cryptoAddress string) (*models.Account, error) {
	s.calls["LookupAccount"]++
	if fn, ok := s.callbacks["LookupAccount"]; ok {
		return fn.(func(ctx context.Context, cryptoAddress string) (*models.Account, error))(ctx, cryptoAddress)
	}
	panic("LookupAccount callback not set")
}

// Set a callback for when "RetrieveAccount()" is called on the mock store.
func (s *Store) OnRetrieveAccount(fn func(ctx context.Context, id ulid.ULID) (*models.Account, error)) {
	s.callbacks["RetrieveAccount"] = fn
}

// Calls the callback previously set for "RetrieveAccount()" (set the callback with "OnRetrieveAccount()")
func (s *Store) RetrieveAccount(ctx context.Context, id ulid.ULID) (*models.Account, error) {
	s.calls["RetrieveAccount"]++
	if fn, ok := s.callbacks["RetrieveAccount"]; ok {
		return fn.(func(ctx context.Context, id ulid.ULID) (*models.Account, error))(ctx, id)
	}
	panic("RetrieveAccount callback not set")
}

// Set a callback for when "UpdateAccount()" is called on the mock store.
func (s *Store) OnUpdateAccount(fn func(ctx context.Context, in *models.Account) error) {
	s.callbacks["UpdateAccount"] = fn
}

// Calls the callback previously set for "UpdateAccount()" (set the callback with "OnUpdateAccount()")
func (s *Store) UpdateAccount(ctx context.Context, in *models.Account) error {
	s.calls["UpdateAccount"]++
	if fn, ok := s.callbacks["UpdateAccount"]; ok {
		return fn.(func(ctx context.Context, in *models.Account) error)(ctx, in)
	}
	panic("UpdateAccount callback not set")
}

// Set a callback for when "DeleteAccount()" is called on the mock store.
func (s *Store) OnDeleteAccount(fn func(ctx context.Context, id ulid.ULID) error) {
	s.callbacks["DeleteAccount"] = fn
}

// Calls the callback previously set for "DeleteAccount()" (set the callback with "OnDeleteAccount()")
func (s *Store) DeleteAccount(ctx context.Context, id ulid.ULID) error {
	s.calls["DeleteAccount"]++
	if fn, ok := s.callbacks["DeleteAccount"]; ok {
		return fn.(func(ctx context.Context, id ulid.ULID) error)(ctx, id)
	}
	panic("DeleteAccount callback not set")
}

// Set a callback for when "ListAccountTransactions()" is called on the mock store.
func (s *Store) OnListAccountTransactions(fn func(ctx context.Context, accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error)) {
	s.callbacks["ListAccountTransactions"] = fn
}

// Calls the callback previously set for "ListAccountTransactions()" (set the callback with "OnListAccountTransactions()")
func (s *Store) ListAccountTransactions(ctx context.Context, accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error) {
	s.calls["ListAccountTransactions"]++
	if fn, ok := s.callbacks["ListAccountTransactions"]; ok {
		return fn.(func(ctx context.Context, accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error))(ctx, accountID, page)
	}
	panic("ListAccountTransactions callback not set")
}

//===========================================================================
// CryptoAddress Store Methods
//===========================================================================

// Set a callback for when "ListCryptoAddresses()" is called on the mock store.
func (s *Store) OnListCryptoAddresses(fn func(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error)) {
	s.callbacks["ListCryptoAddresses"] = fn
}

// Calls the callback previously set for "ListCryptoAddresses()" (set the callback with "OnListCryptoAddresses()")
func (s *Store) ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	s.calls["ListCryptoAddresses"]++
	if fn, ok := s.callbacks["ListCryptoAddresses"]; ok {
		return fn.(func(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error))(ctx, accountID, page)
	}
	panic("ListCryptoAddresses callback not set")
}

// Set a callback for when "CreateCryptoAddress()" is called on the mock store.
func (s *Store) OnCreateCryptoAddress(fn func(ctx context.Context, in *models.CryptoAddress) error) {
	s.callbacks["CreateCryptoAddress"] = fn
}

// Calls the callback previously set for "CreateCryptoAddress()" (set the callback with "OnCreateCryptoAddress()")
func (s *Store) CreateCryptoAddress(ctx context.Context, in *models.CryptoAddress) error {
	s.calls["CreateCryptoAddress"]++
	if fn, ok := s.callbacks["CreateCryptoAddress"]; ok {
		return fn.(func(ctx context.Context, in *models.CryptoAddress) error)(ctx, in)
	}
	panic("CreateCryptoAddress callback not set")
}

// Set a callback for when "RetrieveCryptoAddress()" is called on the mock store.
func (s *Store) OnRetrieveCryptoAddress(fn func(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error)) {
	s.callbacks["RetrieveCryptoAddress"] = fn
}

// Calls the callback previously set for "RetrieveCryptoAddress()" (set the callback with "OnRetrieveCryptoAddress()")
func (s *Store) RetrieveCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error) {
	s.calls["RetrieveCryptoAddress"]++
	if fn, ok := s.callbacks["RetrieveCryptoAddress"]; ok {
		return fn.(func(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error))(ctx, accountID, cryptoAddressID)
	}
	panic("RetrieveCryptoAddress callback not set")
}

// Set a callback for when "UpdateCryptoAddress()" is called on the mock store.
func (s *Store) OnUpdateCryptoAddress(fn func(ctx context.Context, in *models.CryptoAddress) error) {
	s.callbacks["UpdateCryptoAddress"] = fn
}

// Calls the callback previously set for "UpdateCryptoAddress()" (set the callback with "OnUpdateCryptoAddress()")
func (s *Store) UpdateCryptoAddress(ctx context.Context, in *models.CryptoAddress) error {
	s.calls["UpdateCryptoAddress"]++
	if fn, ok := s.callbacks["UpdateCryptoAddress"]; ok {
		return fn.(func(ctx context.Context, in *models.CryptoAddress) error)(ctx, in)
	}
	panic("UpdateCryptoAddress callback not set")
}

// Set a callback for when "DeleteCryptoAddress()" is called on the mock store.
func (s *Store) OnDeleteCryptoAddress(fn func(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error) {
	s.callbacks["DeleteCryptoAddress"] = fn
}

// Calls the callback previously set for "DeleteCryptoAddress()" (set the callback with "OnDeleteCryptoAddress()")
func (s *Store) DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error {
	s.calls["DeleteCryptoAddress"]++
	if fn, ok := s.callbacks["DeleteCryptoAddress"]; ok {
		return fn.(func(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error)(ctx, accountID, cryptoAddressID)
	}
	panic("DeleteCryptoAddress callback not set")
}

//===========================================================================
// Counterparty Store Methods
//===========================================================================

// Set a callback for when "SearchCounterparties()" is called on the mock store.
func (s *Store) OnSearchCounterparties(fn func(ctx context.Context, query *models.SearchQuery) (*models.CounterpartyPage, error)) {
	s.callbacks["SearchCounterparties"] = fn
}

// Calls the callback previously set for "SearchCounterparties()" (set the callback with "OnSearchCounterparties()")
func (s *Store) SearchCounterparties(ctx context.Context, query *models.SearchQuery) (*models.CounterpartyPage, error) {
	s.calls["SearchCounterparties"]++
	if fn, ok := s.callbacks["SearchCounterparties"]; ok {
		return fn.(func(ctx context.Context, query *models.SearchQuery) (*models.CounterpartyPage, error))(ctx, query)
	}
	panic("SearchCounterparties callback not set")
}

// Set a callback for when "ListCounterparties()" is called on the mock store.
func (s *Store) OnListCounterparties(fn func(ctx context.Context, page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error)) {
	s.callbacks["ListCounterparties"] = fn
}

// Calls the callback previously set for "ListCounterparties()" (set the callback with "OnListCounterparties()")
func (s *Store) ListCounterparties(ctx context.Context, page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error) {
	s.calls["ListCounterparties"]++
	if fn, ok := s.callbacks["ListCounterparties"]; ok {
		return fn.(func(ctx context.Context, page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error))(ctx, page)
	}
	panic("ListCounterparties callback not set")
}

// Set a callback for when "ListCounterpartySourceInfo()" is called on the mock store.
func (s *Store) OnListCounterpartySourceInfo(fn func(ctx context.Context, source enum.Source) ([]*models.CounterpartySourceInfo, error)) {
	s.callbacks["ListCounterpartySourceInfo"] = fn
}

// Calls the callback previously set for "ListCounterpartySourceInfo()" (set the callback with "OnListCounterpartySourceInfo()")
func (s *Store) ListCounterpartySourceInfo(ctx context.Context, source enum.Source) ([]*models.CounterpartySourceInfo, error) {
	s.calls["ListCounterpartySourceInfo"]++
	if fn, ok := s.callbacks["ListCounterpartySourceInfo"]; ok {
		return fn.(func(ctx context.Context, source enum.Source) ([]*models.CounterpartySourceInfo, error))(ctx, source)
	}
	panic("ListCounterpartySourceInfo callback not set")
}

// Set a callback for when "CreateCounterparty()" is called on the mock store.
func (s *Store) OnCreateCounterparty(fn func(ctx context.Context, in *models.Counterparty) error) {
	s.callbacks["CreateCounterparty"] = fn
}

// Calls the callback previously set for "CreateCounterparty()" (set the callback with "OnCreateCounterparty()")
func (s *Store) CreateCounterparty(ctx context.Context, in *models.Counterparty) error {
	s.calls["CreateCounterparty"]++
	if fn, ok := s.callbacks["CreateCounterparty"]; ok {
		return fn.(func(ctx context.Context, in *models.Counterparty) error)(ctx, in)
	}
	panic("CreateCounterparty callback not set")
}

// Set a callback for when "RetrieveCounterparty()" is called on the mock store.
func (s *Store) OnRetrieveCounterparty(fn func(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error)) {
	s.callbacks["RetrieveCounterparty"] = fn
}

// Calls the callback previously set for "RetrieveCounterparty()" (set the callback with "OnRetrieveCounterparty()")
func (s *Store) RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error) {
	s.calls["RetrieveCounterparty"]++
	if fn, ok := s.callbacks["RetrieveCounterparty"]; ok {
		return fn.(func(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error))(ctx, counterpartyID)
	}
	panic("RetrieveCounterparty callback not set")
}

// Set a callback for when "LookupCounterparty()" is called on the mock store.
func (s *Store) OnLookupCounterparty(fn func(ctx context.Context, field, value string) (*models.Counterparty, error)) {
	s.callbacks["LookupCounterparty"] = fn
}

// Calls the callback previously set for "LookupCounterparty()" (set the callback with "OnLookupCounterparty()")
func (s *Store) LookupCounterparty(ctx context.Context, field, value string) (*models.Counterparty, error) {
	s.calls["LookupCounterparty"]++
	if fn, ok := s.callbacks["LookupCounterparty"]; ok {
		return fn.(func(ctx context.Context, field, value string) (*models.Counterparty, error))(ctx, field, value)
	}
	panic("LookupCounterparty callback not set")
}

// Set a callback for when "UpdateCounterparty()" is called on the mock store.
func (s *Store) OnUpdateCounterparty(fn func(ctx context.Context, in *models.Counterparty) error) {
	s.callbacks["UpdateCounterparty"] = fn
}

// Calls the callback previously set for "UpdateCounterparty()" (set the callback with "OnUpdateCounterparty()")
func (s *Store) UpdateCounterparty(ctx context.Context, in *models.Counterparty) error {
	s.calls["UpdateCounterparty"]++
	if fn, ok := s.callbacks["UpdateCounterparty"]; ok {
		return fn.(func(ctx context.Context, in *models.Counterparty) error)(ctx, in)
	}
	panic("UpdateCounterparty callback not set")
}

// Set a callback for when "DeleteCounterparty()" is called on the mock store.
func (s *Store) OnDeleteCounterparty(fn func(ctx context.Context, counterpartyID ulid.ULID) error) {
	s.callbacks["DeleteCounterparty"] = fn
}

// Calls the callback previously set for "DeleteCounterparty()" (set the callback with "OnDeleteCounterparty()")
func (s *Store) DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error {
	s.calls["DeleteCounterparty"]++
	if fn, ok := s.callbacks["DeleteCounterparty"]; ok {
		return fn.(func(ctx context.Context, counterpartyID ulid.ULID) error)(ctx, counterpartyID)
	}
	panic("DeleteCounterparty callback not set")
}

//===========================================================================
// Contact Store Methods
//===========================================================================

// Set a callback for when "ListContacts()" is called on the mock store.
func (s *Store) OnListContacts(fn func(ctx context.Context, counterparty any, page *models.PageInfo) (*models.ContactsPage, error)) {
	s.callbacks["ListContacts"] = fn
}

// Calls the callback previously set for "ListContacts()" (set the callback with "OnListContacts()")
func (s *Store) ListContacts(ctx context.Context, counterparty any, page *models.PageInfo) (*models.ContactsPage, error) {
	s.calls["ListContacts"]++
	if fn, ok := s.callbacks["ListContacts"]; ok {
		return fn.(func(ctx context.Context, counterparty any, page *models.PageInfo) (*models.ContactsPage, error))(ctx, counterparty, page)
	}
	panic("ListContacts callback not set")
}

// Set a callback for when "CreateContact()" is called on the mock store.
func (s *Store) OnCreateContact(fn func(ctx context.Context, in *models.Contact) error) {
	s.callbacks["CreateContact"] = fn
}

// Calls the callback previously set for "CreateContact()" (set the callback with "OnCreateContact()")
func (s *Store) CreateContact(ctx context.Context, in *models.Contact) error {
	s.calls["CreateContact"]++
	if fn, ok := s.callbacks["CreateContact"]; ok {
		return fn.(func(ctx context.Context, in *models.Contact) error)(ctx, in)
	}
	panic("CreateContact callback not set")
}

// Set a callback for when "RetrieveContact()" is called on the mock store.
func (s *Store) OnRetrieveContact(fn func(ctx context.Context, contactID, counterparty any) (*models.Contact, error)) {
	s.callbacks["RetrieveContact"] = fn
}

// Calls the callback previously set for "RetrieveContact()" (set the callback with "OnRetrieveContact()")
func (s *Store) RetrieveContact(ctx context.Context, contactID, counterparty any) (*models.Contact, error) {
	s.calls["RetrieveContact"]++
	if fn, ok := s.callbacks["RetrieveContact"]; ok {
		return fn.(func(ctx context.Context, contactID, counterparty any) (*models.Contact, error))(ctx, counterparty, contactID)
	}
	panic("RetrieveContact callback not set")
}

// Set a callback for when "UpdateContact()" is called on the mock store.
func (s *Store) OnUpdateContact(fn func(ctx context.Context, in *models.Contact) error) {
	s.callbacks["UpdateContact"] = fn
}

// Calls the callback previously set for "UpdateContact()" (set the callback with "OnUpdateContact()")
func (s *Store) UpdateContact(ctx context.Context, in *models.Contact) error {
	s.calls["UpdateContact"]++
	if fn, ok := s.callbacks["UpdateContact"]; ok {
		return fn.(func(ctx context.Context, in *models.Contact) error)(ctx, in)
	}
	panic("UpdateContact callback not set")
}

// Set a callback for when "DeleteContact()" is called on the mock store.
func (s *Store) OnDeleteContact(fn func(ctx context.Context, contactID, counterparty any) error) {
	s.callbacks["DeleteContact"] = fn
}

// Calls the callback previously set for "DeleteContact()" (set the callback with "OnDeleteContact()")
func (s *Store) DeleteContact(ctx context.Context, contactID, counterparty any) error {
	s.calls["DeleteContact"]++
	if fn, ok := s.callbacks["DeleteContact"]; ok {
		return fn.(func(ctx context.Context, contactID, counterparty any) error)(ctx, contactID, counterparty)
	}
	panic("DeleteContact callback not set")
}

//===========================================================================
// Travel Address Factory
//===========================================================================

// Set a callback for when "UseTravelAddressFactory()" is called on the mock store.
func (s *Store) OnUseTravelAddressFactory(fn func(models.TravelAddressFactory)) {
	s.callbacks["UseTravelAddressFactory"] = fn
}

// Calls the callback previously set for "UseTravelAddressFactory()" (set the callback with "OnUseTravelAddressFactory()")
func (s *Store) UseTravelAddressFactory(f models.TravelAddressFactory) {
	s.calls["UseTravelAddressFactory"]++
	if fn, ok := s.callbacks["UseTravelAddressFactory"]; ok {
		fn.(func(models.TravelAddressFactory))(f)
	}
	panic("UseTravelAddressFactory callback not set")
}

//===========================================================================
// Sunrise Store Methods
//===========================================================================

// Set a callback for when "ListSunrise()" is called on the mock store.
func (s *Store) OnListSunrise(fn func(ctx context.Context, page *models.PageInfo) (*models.SunrisePage, error)) {
	s.callbacks["ListSunrise"] = fn
}

// Calls the callback previously set for "ListSunrise()" (set the callback with "OnListSunrise()")
func (s *Store) ListSunrise(ctx context.Context, page *models.PageInfo) (*models.SunrisePage, error) {
	s.calls["ListSunrise"]++
	if fn, ok := s.callbacks["ListSunrise"]; ok {
		return fn.(func(ctx context.Context, page *models.PageInfo) (*models.SunrisePage, error))(ctx, page)
	}
	panic("ListSunrise callback not set")
}

// Set a callback for when "CreateSunrise()" is called on the mock store.
func (s *Store) OnCreateSunrise(fn func(ctx context.Context, msg *models.Sunrise) error) {
	s.callbacks["CreateSunrise"] = fn
}

// Calls the callback previously set for "CreateSunrise()" (set the callback with "OnCreateSunrise()")
func (s *Store) CreateSunrise(ctx context.Context, msg *models.Sunrise) error {
	s.calls["CreateSunrise"]++
	if fn, ok := s.callbacks["CreateSunrise"]; ok {
		return fn.(func(ctx context.Context, msg *models.Sunrise) error)(ctx, msg)
	}
	panic("CreateSunrise callback not set")
}

// Set a callback for when "RetrieveSunrise()" is called on the mock store.
func (s *Store) OnRetrieveSunrise(fn func(ctx context.Context, id ulid.ULID) (*models.Sunrise, error)) {
	s.callbacks["RetrieveSunrise"] = fn
}

// Calls the callback previously set for "RetrieveSunrise()" (set the callback with "OnRetrieveSunrise()")
func (s *Store) RetrieveSunrise(ctx context.Context, id ulid.ULID) (*models.Sunrise, error) {
	s.calls["RetrieveSunrise"]++
	if fn, ok := s.callbacks["RetrieveSunrise"]; ok {
		return fn.(func(ctx context.Context, id ulid.ULID) (*models.Sunrise, error))(ctx, id)
	}
	panic("RetrieveSunrise callback not set")
}

// Set a callback for when "UpdateSunrise()" is called on the mock store.
func (s *Store) OnUpdateSunrise(fn func(ctx context.Context, msg *models.Sunrise) error) {
	s.callbacks["UpdateSunrise"] = fn
}

// Calls the callback previously set for "UpdateSunrise()" (set the callback with "OnUpdateSunrise()")
func (s *Store) UpdateSunrise(ctx context.Context, msg *models.Sunrise) error {
	s.calls["UpdateSunrise"]++
	if fn, ok := s.callbacks["UpdateSunrise"]; ok {
		return fn.(func(ctx context.Context, msg *models.Sunrise) error)(ctx, msg)
	}
	panic("UpdateSunrise callback not set")
}

// Set a callback for when "UpdateSunriseStatus()" is called on the mock store.
func (s *Store) OnUpdateSunriseStatus(fn func(ctx context.Context, txID uuid.UUID, status enum.Status) error) {
	s.callbacks["UpdateSunriseStatus"] = fn
}

// Calls the callback previously set for "UpdateSunriseStatus()" (set the callback with "OnUpdateSunriseStatus()")
func (s *Store) UpdateSunriseStatus(ctx context.Context, txID uuid.UUID, status enum.Status) error {
	s.calls["UpdateSunriseStatus"]++
	if fn, ok := s.callbacks["UpdateSunriseStatus"]; ok {
		return fn.(func(ctx context.Context, txID uuid.UUID, status enum.Status) error)(ctx, txID, status)
	}
	panic("UpdateSunriseStatus callback not set")
}

// Set a callback for when "DeleteSunrise()" is called on the mock store.
func (s *Store) OnDeleteSunrise(fn func(ctx context.Context, id ulid.ULID) error) {
	s.callbacks["DeleteSunrise"] = fn
}

// Calls the callback previously set for "DeleteSunrise()" (set the callback with "OnDeleteSunrise()")
func (s *Store) DeleteSunrise(ctx context.Context, id ulid.ULID) error {
	s.calls["DeleteSunrise"]++
	if fn, ok := s.callbacks["DeleteSunrise"]; ok {
		return fn.(func(ctx context.Context, id ulid.ULID) error)(ctx, id)
	}
	panic("DeleteSunrise callback not set")
}

// Set a callback for when "GetOrCreateSunriseCounterparty()" is called on the mock store.
func (s *Store) OnGetOrCreateSunriseCounterparty(fn func(ctx context.Context, email, name string) (*models.Counterparty, error)) {
	s.callbacks["GetOrCreateSunriseCounterparty"] = fn
}

// Calls the callback previously set for "GetOrCreateSunriseCounterparty()" (set the callback with "OnGetOrCreateSunriseCounterparty()")
func (s *Store) GetOrCreateSunriseCounterparty(ctx context.Context, email, name string) (*models.Counterparty, error) {
	s.calls["GetOrCreateSunriseCounterparty"]++
	if fn, ok := s.callbacks["GetOrCreateSunriseCounterparty"]; ok {
		return fn.(func(ctx context.Context, email, name string) (*models.Counterparty, error))(ctx, email, name)
	}
	panic("GetOrCreateSunriseCounterparty callback not set")
}

//===========================================================================
// User Store Methods
//===========================================================================

// Set a callback for when "ListUsers()" is called on the mock store.
func (s *Store) OnListUsers(fn func(ctx context.Context, page *models.UserPageInfo) (*models.UserPage, error)) {
	s.callbacks["ListUsers"] = fn
}

// Calls the callback previously set for "ListUsers()" (set the callback with "OnListUsers()")
func (s *Store) ListUsers(ctx context.Context, page *models.UserPageInfo) (*models.UserPage, error) {
	s.calls["ListUsers"]++
	if fn, ok := s.callbacks["ListUsers"]; ok {
		return fn.(func(ctx context.Context, page *models.UserPageInfo) (*models.UserPage, error))(ctx, page)
	}
	panic("ListUsers callback not set")
}

// Set a callback for when "CreateUser()" is called on the mock store.
func (s *Store) OnCreateUser(fn func(ctx context.Context, in *models.User) error) {
	s.callbacks["CreateUser"] = fn
}

// Calls the callback previously set for "CreateUser()" (set the callback with "OnCreateUser()")
func (s *Store) CreateUser(ctx context.Context, in *models.User) error {
	s.calls["CreateUser"]++
	if fn, ok := s.callbacks["CreateUser"]; ok {
		return fn.(func(ctx context.Context, in *models.User) error)(ctx, in)
	}
	panic("CreateUser callback not set")
}

// Set a callback for when "RetrieveUser()" is called on the mock store.
func (s *Store) OnRetrieveUser(fn func(ctx context.Context, emailOrUserID any) (*models.User, error)) {
	s.callbacks["RetrieveUser"] = fn
}

// Calls the callback previously set for "RetrieveUser()" (set the callback with "OnRetrieveUser()")
func (s *Store) RetrieveUser(ctx context.Context, emailOrUserID any) (*models.User, error) {
	s.calls["RetrieveUser"]++
	if fn, ok := s.callbacks["RetrieveUser"]; ok {
		return fn.(func(ctx context.Context, emailOrUserID any) (*models.User, error))(ctx, emailOrUserID)
	}
	panic("RetrieveUser callback not set")
}

// Set a callback for when "UpdateUser()" is called on the mock store.
func (s *Store) OnUpdateUser(fn func(ctx context.Context, in *models.User) error) {
	s.callbacks["UpdateUser"] = fn
}

// Calls the callback previously set for "UpdateUser()" (set the callback with "OnUpdateUser()")
func (s *Store) UpdateUser(ctx context.Context, in *models.User) error {
	s.calls["UpdateUser"]++
	if fn, ok := s.callbacks["UpdateUser"]; ok {
		return fn.(func(ctx context.Context, in *models.User) error)(ctx, in)
	}
	panic("UpdateUser callback not set")
}

// Set a callback for when "SetUserPassword()" is called on the mock store.
func (s *Store) OnSetUserPassword(fn func(ctx context.Context, userID ulid.ULID, password string) (err error)) {
	s.callbacks["SetUserPassword"] = fn
}

// Calls the callback previously set for "SetUserPassword()" (set the callback with "OnSetUserPassword()")
func (s *Store) SetUserPassword(ctx context.Context, userID ulid.ULID, password string) (err error) {
	s.calls["SetUserPassword"]++
	if fn, ok := s.callbacks["SetUserPassword"]; ok {
		return fn.(func(ctx context.Context, userID ulid.ULID, password string) (err error))(ctx, userID, password)
	}
	panic("SetUserPassword callback not set")
}

// Set a callback for when "SetUserLastLogin()" is called on the mock store.
func (s *Store) OnSetUserLastLogin(fn func(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error)) {
	s.callbacks["SetUserLastLogin"] = fn
}

// Calls the callback previously set for "SetUserLastLogin()" (set the callback with "OnSetUserLastLogin()")
func (s *Store) SetUserLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error) {
	s.calls["SetUserLastLogin"]++
	if fn, ok := s.callbacks["SetUserLastLogin"]; ok {
		return fn.(func(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error))(ctx, userID, lastLogin)
	}
	panic("SetUserLastLogin callback not set")
}

// Set a callback for when "DeleteUser()" is called on the mock store.
func (s *Store) OnDeleteUser(fn func(ctx context.Context, userID ulid.ULID) error) {
	s.callbacks["DeleteUser"] = fn
}

// Calls the callback previously set for "DeleteUser()" (set the callback with "OnDeleteUser()")
func (s *Store) DeleteUser(ctx context.Context, userID ulid.ULID) error {
	s.calls["DeleteUser"]++
	if fn, ok := s.callbacks["DeleteUser"]; ok {
		return fn.(func(ctx context.Context, userID ulid.ULID) error)(ctx, userID)
	}
	panic("DeleteUser callback not set")
}

// Set a callback for when "LookupRole()" is called on the mock store.
func (s *Store) OnLookupRole(fn func(ctx context.Context, role string) (*models.Role, error)) {
	s.callbacks["LookupRole"] = fn
}

// Calls the callback previously set for "LookupRole()" (set the callback with "OnLookupRole()")
func (s *Store) LookupRole(ctx context.Context, role string) (*models.Role, error) {
	s.calls["LookupRole"]++
	if fn, ok := s.callbacks["LookupRole"]; ok {
		return fn.(func(ctx context.Context, role string) (*models.Role, error))(ctx, role)
	}
	panic("LookupRole callback not set")
}

//===========================================================================
// API Key Store Methods
//===========================================================================

// Set a callback for when "ListAPIKeys()" is called on the mock store.
func (s *Store) OnListAPIKeys(fn func(ctx context.Context, in *models.PageInfo) (*models.APIKeyPage, error)) {
	s.callbacks["ListAPIKeys"] = fn
}

// Calls the callback previously set for "ListAPIKeys()" (set the callback with "OnListAPIKeys()")
func (s *Store) ListAPIKeys(ctx context.Context, in *models.PageInfo) (*models.APIKeyPage, error) {
	s.calls["ListAPIKeys"]++
	if fn, ok := s.callbacks["ListAPIKeys"]; ok {
		return fn.(func(ctx context.Context, in *models.PageInfo) (*models.APIKeyPage, error))(ctx, in)
	}
	panic("ListAPIKeys callback not set")
}

// Set a callback for when "CreateAPIKey()" is called on the mock store.
func (s *Store) OnCreateAPIKey(fn func(ctx context.Context, in *models.APIKey) error) {
	s.callbacks["CreateAPIKey"] = fn
}

// Calls the callback previously set for "CreateAPIKey()" (set the callback with "OnCreateAPIKey()")
func (s *Store) CreateAPIKey(ctx context.Context, in *models.APIKey) error {
	s.calls["CreateAPIKey"]++
	if fn, ok := s.callbacks["CreateAPIKey"]; ok {
		return fn.(func(ctx context.Context, in *models.APIKey) error)(ctx, in)
	}
	panic("CreateAPIKey callback not set")
}

// Set a callback for when "RetrieveAPIKey()" is called on the mock store.
func (s *Store) OnRetrieveAPIKey(fn func(ctx context.Context, clientIDOrKeyID any) (*models.APIKey, error)) {
	s.callbacks["RetrieveAPIKey"] = fn
}

// Calls the callback previously set for "RetrieveAPIKey()" (set the callback with "OnRetrieveAPIKey()")
func (s *Store) RetrieveAPIKey(ctx context.Context, clientIDOrKeyID any) (*models.APIKey, error) {
	s.calls["RetrieveAPIKey"]++
	if fn, ok := s.callbacks["RetrieveAPIKey"]; ok {
		return fn.(func(ctx context.Context, clientIDOrKeyID any) (*models.APIKey, error))(ctx, clientIDOrKeyID)
	}
	panic("RetrieveAPIKey callback not set")
}

// Set a callback for when "UpdateAPIKey()" is called on the mock store.
func (s *Store) OnUpdateAPIKey(fn func(ctx context.Context, in *models.APIKey) error) {
	s.callbacks["UpdateAPIKey"] = fn
}

// Calls the callback previously set for "UpdateAPIKey()" (set the callback with "OnUpdateAPIKey()")
func (s *Store) UpdateAPIKey(ctx context.Context, in *models.APIKey) error {
	s.calls["UpdateAPIKey"]++
	if fn, ok := s.callbacks["UpdateAPIKey"]; ok {
		return fn.(func(ctx context.Context, in *models.APIKey) error)(ctx, in)
	}
	panic("UpdateAPIKey callback not set")
}

// Set a callback for when "DeleteAPIKey()" is called on the mock store.
func (s *Store) OnDeleteAPIKey(fn func(ctx context.Context, keyID ulid.ULID) error) {
	s.callbacks["DeleteAPIKey"] = fn
}

// Calls the callback previously set for "DeleteAPIKey()" (set the callback with "OnDeleteAPIKey()")
func (s *Store) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) error {
	s.calls["DeleteAPIKey"]++
	if fn, ok := s.callbacks["DeleteAPIKey"]; ok {
		return fn.(func(ctx context.Context, keyID ulid.ULID) error)(ctx, keyID)
	}
	panic("DeleteAPIKey callback not set")
}

//===========================================================================
// Reset Password Link Store Methods
//===========================================================================

// Set a callback for when "ListResetPasswordLinks()" is called on the mock store.
func (s *Store) OnListResetPasswordLinks(fn func(ctx context.Context, page *models.PageInfo) (*models.ResetPasswordLinkPage, error)) {
	s.callbacks["ListResetPasswordLinks"] = fn
}

// Calls the callback previously set for "ListResetPasswordLinks()" (set the callback with "OnListResetPasswordLinks()")
func (s *Store) ListResetPasswordLinks(ctx context.Context, page *models.PageInfo) (*models.ResetPasswordLinkPage, error) {
	s.calls["ListResetPasswordLinks"]++
	if fn, ok := s.callbacks["ListResetPasswordLinks"]; ok {
		return fn.(func(ctx context.Context, page *models.PageInfo) (*models.ResetPasswordLinkPage, error))(ctx, page)
	}
	panic("ListResetPasswordLinks callback not set")
}

// Set a callback for when "CreateResetPasswordLink()" is called on the mock store.
func (s *Store) OnCreateResetPasswordLink(fn func(ctx context.Context, link *models.ResetPasswordLink) error) {
	s.callbacks["CreateResetPasswordLink"] = fn
}

// Calls the callback previously set for "CreateResetPasswordLink()" (set the callback with "OnCreateResetPasswordLink()")
func (s *Store) CreateResetPasswordLink(ctx context.Context, link *models.ResetPasswordLink) error {
	s.calls["CreateResetPasswordLink"]++
	if fn, ok := s.callbacks["CreateResetPasswordLink"]; ok {
		return fn.(func(ctx context.Context, link *models.ResetPasswordLink) error)(ctx, link)
	}
	panic("CreateResetPasswordLink callback not set")
}

// Set a callback for when "RetrieveResetPasswordLink()" is called on the mock store.
func (s *Store) OnRetrieveResetPasswordLink(fn func(ctx context.Context, linkID ulid.ULID) (*models.ResetPasswordLink, error)) {
	s.callbacks["RetrieveResetPasswordLink"] = fn
}

// Calls the callback previously set for "RetrieveResetPasswordLink()" (set the callback with "OnRetrieveResetPasswordLink()")
func (s *Store) RetrieveResetPasswordLink(ctx context.Context, linkID ulid.ULID) (*models.ResetPasswordLink, error) {
	s.calls["RetrieveResetPasswordLink"]++
	if fn, ok := s.callbacks["RetrieveResetPasswordLink"]; ok {
		return fn.(func(ctx context.Context, linkID ulid.ULID) (*models.ResetPasswordLink, error))(ctx, linkID)
	}
	panic("RetrieveResetPasswordLink callback not set")
}

// Set a callback for when "UpdateResetPasswordLink()" is called on the mock store.
func (s *Store) OnUpdateResetPasswordLink(fn func(ctx context.Context, link *models.ResetPasswordLink) error) {
	s.callbacks["UpdateResetPasswordLink"] = fn
}

// Calls the callback previously set for "UpdateResetPasswordLink()" (set the callback with "OnUpdateResetPasswordLink()")
func (s *Store) UpdateResetPasswordLink(ctx context.Context, link *models.ResetPasswordLink) error {
	s.calls["UpdateResetPasswordLink"]++
	if fn, ok := s.callbacks["UpdateResetPasswordLink"]; ok {
		return fn.(func(ctx context.Context, link *models.ResetPasswordLink) error)(ctx, link)
	}
	panic("UpdateResetPasswordLink callback not set")
}

// Set a callback for when "DeleteResetPasswordLink()" is called on the mock store.
func (s *Store) OnDeleteResetPasswordLink(fn func(ctx context.Context, linkID ulid.ULID) (err error)) {
	s.callbacks["DeleteResetPasswordLink"] = fn
}

// Calls the callback previously set for "DeleteResetPasswordLink()" (set the callback with "OnDeleteResetPasswordLink()")
func (s *Store) DeleteResetPasswordLink(ctx context.Context, linkID ulid.ULID) (err error) {
	s.calls["DeleteResetPasswordLink"]++
	if fn, ok := s.callbacks["DeleteResetPasswordLink"]; ok {
		return fn.(func(ctx context.Context, linkID ulid.ULID) (err error))(ctx, linkID)
	}
	panic("DeleteResetPasswordLink callback not set")
}
