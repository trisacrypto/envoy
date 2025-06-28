package mock

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

// Tx implements the store.Txn interface using a mock store. The test user can specify
// any callbacks to simulate specific behaviors. By default the Tx struct will respect
// the readonly option so long as no overriding method is provided.
type Tx struct {
	opts      *sql.TxOptions
	callbacks map[string]any
	calls     map[string]int
	commit    bool
	rollback  bool
}

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Reset all the calls and callbacks in the transaction, if you don't want to
// create a new one.
func (tx *Tx) Reset() {
	// Set maps to nil to free up memory
	tx.calls = nil
	tx.callbacks = nil

	// Create new calls and callbacks maps
	tx.calls = make(map[string]int)
	tx.callbacks = make(map[string]any)

	// Reset transaction commit/rollback
	tx.commit = false
	tx.rollback = false
}

// Assert that the expected number of calls were made to the given method.
func (tx *Tx) AssertCalls(t testing.TB, method string, expected int) {
	require.Equal(t, expected, tx.calls[method], "expected %d calls to %s, got %d", expected, method, tx.calls[method])
}

// Assert that Commit has been called on the transaction without rollback.
func (tx *Tx) AssertCommit(t testing.TB) {
	require.True(t, tx.commit && !tx.rollback, "expected Commit to be called but not Rollback")
}

// Assert that Rollback has been called on the transaction without commit.
func (tx *Tx) AssertRollback(t testing.TB) {
	require.True(t, tx.rollback && !tx.commit, "expected Rollback to be called but not Commit")
}

// Assert that Commit has not been called on the transaction.
func (tx *Tx) AssertNoCommit(t testing.TB) {
	require.False(t, tx.commit, "did not expect Commit to be called")
}

// Assert that Rollback has not been called on the transaction.
func (tx *Tx) AssertNoRollback(t testing.TB) {
	require.False(t, tx.rollback, "did not expect Rollback to be called")
}

// Check is a helper method that determines if the transaction is committed or rolled
// back. If so it returns ErrTxDone no matter if there is a callback set. Additionally,
// if the writeable option is set to true, it will return ErrReadOnly if the transaction
// is read-only. Check will record the calls to the method on the transaction, and
// finally, if the method is not set in callbacks, it panics.
func (tx *Tx) check(method string, writable bool) (any, error) {
	tx.calls[method]++

	if tx.commit || tx.rollback {
		return nil, sql.ErrTxDone
	}

	if writable && tx.opts != nil && tx.opts.ReadOnly {
		return nil, errors.ErrReadOnly
	}

	if fn, ok := tx.callbacks[method]; ok {
		return fn, nil
	}

	panic(fmt.Errorf("%q callback not set", method))
}

//===========================================================================
// Txn Interface Methods
//===========================================================================

// Set a callback for when "Commit()" is called on the mock Txn.
func (tx *Tx) OnCommit(fn func() error) {
	tx.callbacks["Commit"] = fn
}

// Calls the callback previously set with "OnCommit()", or completes the
// "commit" for the transaction.
func (tx *Tx) Commit() error {
	tx.calls["Commit"]++
	if fn, ok := tx.callbacks["Commit"]; ok {
		return fn.(func() error)()
	}

	// ensure the transaction is still active
	if tx.commit || tx.rollback {
		return sql.ErrTxDone
	}

	tx.commit = true
	return nil
}

// Set a callback for when "Rollback()" is called on the mock Txn.
func (tx *Tx) OnRollback(fn func() error) {
	tx.callbacks["Rollback"] = fn
}

// Calls the callback previously set with "OnRollback()", or completes the
// "rollback" for the transaction.
func (tx *Tx) Rollback() error {
	tx.calls["Rollback"]++
	if fn, ok := tx.callbacks["Rollback"]; ok {
		return fn.(func() error)()
	}

	// ensure the transaction is still active
	if tx.commit || tx.rollback {
		return sql.ErrTxDone
	}

	tx.rollback = true
	return nil
}

//===========================================================================
// Transaction Interface Methods
//===========================================================================

// Set a callback for when "ListTransactions()" is called on the mock Txn.
func (tx *Tx) OnListTransactions(fn func(in *models.TransactionPageInfo) (*models.TransactionPage, error)) {
	tx.callbacks["ListTransactions"] = fn
}

// Calls the callback previously set with "OnListTransactions()".
func (tx *Tx) ListTransactions(in *models.TransactionPageInfo) (*models.TransactionPage, error) {
	fn, err := tx.check("ListTransactions", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(in *models.TransactionPageInfo) (*models.TransactionPage, error))(in)
}

// Set a callback for when "CreateTransaction()" is called on the mock Txn.
func (tx *Tx) OnCreateTransaction(fn func(in *models.Transaction) error) {
	tx.callbacks["CreateTransaction"] = fn
}

// Calls the callback previously set with "OnCreateTransaction()".
func (tx *Tx) CreateTransaction(in *models.Transaction) error {
	fn, err := tx.check("CreateTransaction", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Transaction) error)(in)
}

// Set a callback for when "RetrieveTransaction()" is called on the mock Txn.
func (tx *Tx) OnRetrieveTransaction(fn func(id uuid.UUID) (*models.Transaction, error)) {
	tx.callbacks["RetrieveTransaction"] = fn
}

// Calls the callback previously set with "OnRetrieveTransaction()".
func (tx *Tx) RetrieveTransaction(id uuid.UUID) (*models.Transaction, error) {
	fn, err := tx.check("RetrieveTransaction", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(id uuid.UUID) (*models.Transaction, error))(id)
}

// Set a callback for when "UpdateTransaction()" is called on the mock Txn.
func (tx *Tx) OnUpdateTransaction(fn func(in *models.Transaction) error) {
	tx.callbacks["UpdateTransaction"] = fn
}

// Calls the callback previously set with "OnUpdateTransaction()".
func (tx *Tx) UpdateTransaction(in *models.Transaction) error {
	fn, err := tx.check("UpdateTransaction", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Transaction) error)(in)
}

// Set a callback for when "DeleteTransaction()" is called on the mock Txn.
func (tx *Tx) OnDeleteTransaction(fn func(id uuid.UUID) error) {
	tx.callbacks["DeleteTransaction"] = fn
}

// Calls the callback previously set with "OnDeleteTransaction()".
func (tx *Tx) DeleteTransaction(id uuid.UUID) error {
	fn, err := tx.check("DeleteTransaction", true)
	if err != nil {
		return err
	}

	return fn.(func(id uuid.UUID) error)(id)
}

// Set a callback for when "ArchiveTransaction()" is called on the mock Txn.
func (tx *Tx) OnArchiveTransaction(fn func(id uuid.UUID) error) {
	tx.callbacks["ArchiveTransaction"] = fn
}

// Calls the callback previously set with "OnArchiveTransaction()".
func (tx *Tx) ArchiveTransaction(id uuid.UUID) error {
	fn, err := tx.check("ArchiveTransaction", true)
	if err != nil {
		return err
	}

	return fn.(func(id uuid.UUID) error)(id)
}

// Set a callback for when "UnarchiveTransaction()" is called on the mock Txn.
func (tx *Tx) OnUnarchiveTransaction(fn func(id uuid.UUID) error) {
	tx.callbacks["UnarchiveTransaction"] = fn
}

// Calls the callback previously set with "OnUnarchiveTransaction()".
func (tx *Tx) UnarchiveTransaction(id uuid.UUID) error {
	fn, err := tx.check("UnarchiveTransaction", true)
	if err != nil {
		return err
	}

	return fn.(func(id uuid.UUID) error)(id)
}

// Set a callback for when "CountTransactions()" is called on the mock Txn.
func (tx *Tx) OnCountTransactions(fn func() (*models.TransactionCounts, error)) {
	tx.callbacks["CountTransactions"] = fn
}

// Calls the callback previously set with "OnCountTransactions()".
func (tx *Tx) CountTransactions() (*models.TransactionCounts, error) {
	fn, err := tx.check("CountTransactions", false)
	if err != nil {
		return nil, err
	}

	return fn.(func() (*models.TransactionCounts, error))()
}

// Set a callback for when "TransactionState()" is called on the mock Txn.
func (tx *Tx) OnTransactionState(fn func(id uuid.UUID) (bool, enum.Status, error)) {
	tx.callbacks["TransactionState"] = fn
}

// Calls the callback previously set with "OnTransactionState()".
func (tx *Tx) TransactionState(id uuid.UUID) (bool, enum.Status, error) {
	fn, err := tx.check("TransactionState", false)
	if err != nil {
		return false, enum.StatusUnspecified, err
	}

	return fn.(func(id uuid.UUID) (bool, enum.Status, error))(id)
}

//===========================================================================
// SecureEnvelope Interface Methods
//===========================================================================

// Set a callback for when "ListSecureEnvelopes()" is called on the mock Txn.
func (tx *Tx) OnListSecureEnvelopes(fn func(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error)) {
	tx.callbacks["ListSecureEnvelopes"] = fn
}

// Calls the callback previously set with "OnListSecureEnvelopes()".
func (tx *Tx) ListSecureEnvelopes(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	fn, err := tx.check("ListSecureEnvelopes", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error))(txID, page)
}

// Set a callback for when "CreateSecureEnvelope()" is called on the mock Txn.
func (tx *Tx) OnCreateSecureEnvelope(fn func(in *models.SecureEnvelope) error) {
	tx.callbacks["CreateSecureEnvelope"] = fn
}

// Calls the callback previously set with "OnCreateSecureEnvelope()".
func (tx *Tx) CreateSecureEnvelope(in *models.SecureEnvelope) error {
	fn, err := tx.check("CreateSecureEnvelope", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.SecureEnvelope) error)(in)
}

// Set a callback for when "RetrieveSecureEnvelope()" is called on the mock Txn.
func (tx *Tx) OnRetrieveSecureEnvelope(fn func(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error)) {
	tx.callbacks["RetrieveSecureEnvelope"] = fn
}

// Calls the callback previously set with "OnRetrieveSecureEnvelope()".
func (tx *Tx) RetrieveSecureEnvelope(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	fn, err := tx.check("RetrieveSecureEnvelope", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error))(txID, envID)
}

// Set a callback for when "UpdateSecureEnvelope()" is called on the mock Txn.
func (tx *Tx) OnUpdateSecureEnvelope(fn func(in *models.SecureEnvelope) error) {
	tx.callbacks["UpdateSecureEnvelope"] = fn
}

// Calls the callback previously set with "OnUpdateSecureEnvelope()".
func (tx *Tx) UpdateSecureEnvelope(in *models.SecureEnvelope) error {
	fn, err := tx.check("UpdateSecureEnvelope", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.SecureEnvelope) error)(in)
}

// Set a callback for when "DeleteSecureEnvelope()" is called on the mock Txn.
func (tx *Tx) OnDeleteSecureEnvelope(fn func(txID uuid.UUID, envID ulid.ULID) error) {
	tx.callbacks["DeleteSecureEnvelope"] = fn
}

// Calls the callback previously set with "OnDeleteSecureEnvelope()".
func (tx *Tx) DeleteSecureEnvelope(txID uuid.UUID, envID ulid.ULID) error {
	fn, err := tx.check("DeleteSecureEnvelope", true)
	if err != nil {
		return err
	}

	return fn.(func(txID uuid.UUID, envID ulid.ULID) error)(txID, envID)
}

// Set a callback for when "LatestSecureEnvelope()" is called on the mock Txn.
func (tx *Tx) OnLatestSecureEnvelope(fn func(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)) {
	tx.callbacks["LatestSecureEnvelope"] = fn
}

// Calls the callback previously set with "OnLatestSecureEnvelope()".
func (tx *Tx) LatestSecureEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	fn, err := tx.check("LatestSecureEnvelope", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error))(txID, direction)
}

// Set a callback for when "LatestPayloadEnvelope()" is called on the mock Txn.
func (tx *Tx) OnLatestPayloadEnvelope(fn func(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)) {
	tx.callbacks["LatestPayloadEnvelope"] = fn
}

// Calls the callback previously set with "OnLatestPayloadEnvelope()".
func (tx *Tx) LatestPayloadEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	fn, err := tx.check("LatestPayloadEnvelope", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error))(txID, direction)
}

//===========================================================================
// Account Interface Methods
//===========================================================================

// Set a callback for when "ListAccounts()" is called on the mock Txn.
func (tx *Tx) OnListAccounts(fn func(page *models.PageInfo) (*models.AccountsPage, error)) {
	tx.callbacks["ListAccounts"] = fn
}

// Calls the callback previously set with "OnListAccounts()".
func (tx *Tx) ListAccounts(page *models.PageInfo) (*models.AccountsPage, error) {
	fn, err := tx.check("ListAccounts", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(page *models.PageInfo) (*models.AccountsPage, error))(page)
}

// Set a callback for when "CreateAccount()" is called on the mock Txn.
func (tx *Tx) OnCreateAccount(fn func(in *models.Account) error) {
	tx.callbacks["CreateAccount"] = fn
}

// Calls the callback previously set with "OnCreateAccount()".
func (tx *Tx) CreateAccount(in *models.Account) error {
	fn, err := tx.check("CreateAccount", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Account) error)(in)
}

// Set a callback for when "LookupAccount()" is called on the mock Txn.
func (tx *Tx) OnLookupAccount(fn func(cryptoAddress string) (*models.Account, error)) {
	tx.callbacks["LookupAccount"] = fn
}

// Calls the callback previously set with "OnLookupAccount()".
func (tx *Tx) LookupAccount(cryptoAddress string) (*models.Account, error) {
	fn, err := tx.check("LookupAccount", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(cryptoAddress string) (*models.Account, error))(cryptoAddress)
}

// Set a callback for when "RetrieveAccount()" is called on the mock Txn.
func (tx *Tx) OnRetrieveAccount(fn func(id ulid.ULID) (*models.Account, error)) {
	tx.callbacks["RetrieveAccount"] = fn
}

// Calls the callback previously set with "OnRetrieveAccount()".
func (tx *Tx) RetrieveAccount(id ulid.ULID) (*models.Account, error) {
	fn, err := tx.check("RetrieveAccount", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(id ulid.ULID) (*models.Account, error))(id)
}

// Set a callback for when "UpdateAccount()" is called on the mock Txn.
func (tx *Tx) OnUpdateAccount(fn func(in *models.Account) error) {
	tx.callbacks["UpdateAccount"] = fn
}

// Calls the callback previously set with "OnUpdateAccount()".
func (tx *Tx) UpdateAccount(in *models.Account) error {
	fn, err := tx.check("UpdateAccount", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Account) error)(in)
}

// Set a callback for when "DeleteAccount()" is called on the mock Txn.
func (tx *Tx) OnDeleteAccount(fn func(id ulid.ULID) error) {
	tx.callbacks["DeleteAccount"] = fn
}

// Calls the callback previously set with "OnDeleteAccount()".
func (tx *Tx) DeleteAccount(id ulid.ULID) error {
	fn, err := tx.check("DeleteAccount", true)
	if err != nil {
		return err
	}

	return fn.(func(id ulid.ULID) error)(id)
}

// Set a callback for when "ListAccountTransactions()" is called on the mock Txn.
func (tx *Tx) OnListAccountTransactions(fn func(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error)) {
	tx.callbacks["ListAccountTransactions"] = fn
}

// Calls the callback previously set with "OnListAccountTransactions()".
func (tx *Tx) ListAccountTransactions(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error) {
	fn, err := tx.check("ListAccountTransactions", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error))(accountID, page)
}

//===========================================================================
// CryptoAddress Interface Methods
//===========================================================================

// Set a callback for when "ListCryptoAddresses()" is called on the mock Txn.
func (tx *Tx) OnListCryptoAddresses(fn func(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error)) {
	tx.callbacks["ListCryptoAddresses"] = fn
}

// Calls the callback previously set with "OnListCryptoAddresses()".
func (tx *Tx) ListCryptoAddresses(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	fn, err := tx.check("ListCryptoAddresses", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error))(accountID, page)
}

// Set a callback for when "CreateCryptoAddress()" is called on the mock Txn.
func (tx *Tx) OnCreateCryptoAddress(fn func(in *models.CryptoAddress) error) {
	tx.callbacks["CreateCryptoAddress"] = fn
}

// Calls the callback previously set with "OnCreateCryptoAddress()".
func (tx *Tx) CreateCryptoAddress(in *models.CryptoAddress) error {
	fn, err := tx.check("CreateCryptoAddress", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.CryptoAddress) error)(in)
}

// Set a callback for when "RetrieveCryptoAddress()" is called on the mock Txn.
func (tx *Tx) OnRetrieveCryptoAddress(fn func(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error)) {
	tx.callbacks["RetrieveCryptoAddress"] = fn
}

// Calls the callback previously set with "OnRetrieveCryptoAddress()".
func (tx *Tx) RetrieveCryptoAddress(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error) {
	fn, err := tx.check("RetrieveCryptoAddress", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error))(accountID, cryptoAddressID)
}

// Set a callback for when "UpdateCryptoAddress()" is called on the mock Txn.
func (tx *Tx) OnUpdateCryptoAddress(fn func(in *models.CryptoAddress) error) {
	tx.callbacks["UpdateCryptoAddress"] = fn
}

// Calls the callback previously set with "OnUpdateCryptoAddress()".
func (tx *Tx) UpdateCryptoAddress(in *models.CryptoAddress) error {
	fn, err := tx.check("UpdateCryptoAddress", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.CryptoAddress) error)(in)
}

// Set a callback for when "DeleteCryptoAddress()" is called on the mock Txn.
func (tx *Tx) OnDeleteCryptoAddress(fn func(accountID, cryptoAddressID ulid.ULID) error) {
	tx.callbacks["DeleteCryptoAddress"] = fn
}

// Calls the callback previously set with "OnDeleteCryptoAddress()".
func (tx *Tx) DeleteCryptoAddress(accountID, cryptoAddressID ulid.ULID) error {
	fn, err := tx.check("DeleteCryptoAddress", true)
	if err != nil {
		return err
	}

	return fn.(func(accountID, cryptoAddressID ulid.ULID) error)(accountID, cryptoAddressID)
}

//===========================================================================
// Counterparty Interface Methods
//===========================================================================

// Set a callback for when "SearchCounterparties()" is called on the mock Txn.
func (tx *Tx) OnSearchCounterparties(fn func(query *models.SearchQuery) (*models.CounterpartyPage, error)) {
	tx.callbacks["SearchCounterparties"] = fn
}

// Calls the callback previously set with "OnSearchCounterparties()".
func (tx *Tx) SearchCounterparties(query *models.SearchQuery) (*models.CounterpartyPage, error) {
	fn, err := tx.check("SearchCounterparties", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(query *models.SearchQuery) (*models.CounterpartyPage, error))(query)
}

// Set a callback for when "ListCounterparties()" is called on the mock Txn.
func (tx *Tx) OnListCounterparties(fn func(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error)) {
	tx.callbacks["ListCounterparties"] = fn
}

// Calls the callback previously set with "OnListCounterparties()".
func (tx *Tx) ListCounterparties(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error) {
	fn, err := tx.check("ListCounterparties", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error))(page)
}

// Set a callback for when "ListCounterpartySourceInfo()" is called on the mock Txn.
func (tx *Tx) OnListCounterpartySourceInfo(fn func(source enum.Source) ([]*models.CounterpartySourceInfo, error)) {
	tx.callbacks["ListCounterpartySourceInfo"] = fn
}

// Calls the callback previously set with "OnListCounterpartySourceInfo()".
func (tx *Tx) ListCounterpartySourceInfo(source enum.Source) ([]*models.CounterpartySourceInfo, error) {
	fn, err := tx.check("ListCounterpartySourceInfo", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(source enum.Source) ([]*models.CounterpartySourceInfo, error))(source)
}

// Set a callback for when "CreateCounterparty()" is called on the mock Txn.
func (tx *Tx) OnCreateCounterparty(fn func(in *models.Counterparty) error) {
	tx.callbacks["CreateCounterparty"] = fn
}

// Calls the callback previously set with "OnCreateCounterparty()".
func (tx *Tx) CreateCounterparty(in *models.Counterparty) error {
	fn, err := tx.check("CreateCounterparty", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Counterparty) error)(in)
}

// Set a callback for when "RetrieveCounterparty()" is called on the mock Txn.
func (tx *Tx) OnRetrieveCounterparty(fn func(counterpartyID ulid.ULID) (*models.Counterparty, error)) {
	tx.callbacks["RetrieveCounterparty"] = fn
}

// Calls the callback previously set with "OnRetrieveCounterparty()".
func (tx *Tx) RetrieveCounterparty(counterpartyID ulid.ULID) (*models.Counterparty, error) {
	fn, err := tx.check("RetrieveCounterparty", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(counterpartyID ulid.ULID) (*models.Counterparty, error))(counterpartyID)
}

// Set a callback for when "LookupCounterparty()" is called on the mock Txn.
func (tx *Tx) OnLookupCounterparty(fn func(field, value string) (*models.Counterparty, error)) {
	tx.callbacks["LookupCounterparty"] = fn
}

// Calls the callback previously set with "OnLookupCounterparty()".
func (tx *Tx) LookupCounterparty(field, value string) (*models.Counterparty, error) {
	fn, err := tx.check("LookupCounterparty", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(field, value string) (*models.Counterparty, error))(field, value)

}

// Set a callback for when "UpdateCounterparty()" is called on the mock Txn.
func (tx *Tx) OnUpdateCounterparty(fn func(in *models.Counterparty) error) {
	tx.callbacks["UpdateCounterparty"] = fn
}

// Calls the callback previously set with "OnUpdateCounterparty()".
func (tx *Tx) UpdateCounterparty(in *models.Counterparty) error {
	fn, err := tx.check("UpdateCounterparty", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Counterparty) error)(in)
}

// Set a callback for when "DeleteCounterparty()" is called on the mock Txn.
func (tx *Tx) OnDeleteCounterparty(fn func(counterpartyID ulid.ULID) error) {
	tx.callbacks["DeleteCounterparty"] = fn
}

// Calls the callback previously set with "OnDeleteCounterparty()".
func (tx *Tx) DeleteCounterparty(counterpartyID ulid.ULID) error {
	fn, err := tx.check("DeleteCounterparty", true)
	if err != nil {
		return err
	}

	return fn.(func(counterpartyID ulid.ULID) error)(counterpartyID)
}

//===========================================================================
// Contact Interface Methods
//===========================================================================

// Set a callback for when "ListContacts()" is called on the mock Txn.
func (tx *Tx) OnListContacts(fn func(counterparty any, page *models.PageInfo) (*models.ContactsPage, error)) {
	tx.callbacks["ListContacts"] = fn
}

// Calls the callback previously set with "OnListContacts()".
func (tx *Tx) ListContacts(counterparty any, page *models.PageInfo) (*models.ContactsPage, error) {
	fn, err := tx.check("ListContacts", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(counterparty any, page *models.PageInfo) (*models.ContactsPage, error))(counterparty, page)
}

// Set a callback for when "CreateContact()" is called on the mock Txn.
func (tx *Tx) OnCreateContact(fn func(in *models.Contact) error) {
	tx.callbacks["CreateContact"] = fn
}

// Calls the callback previously set with "OnCreateContact()".
func (tx *Tx) CreateContact(in *models.Contact) error {
	fn, err := tx.check("CreateContact", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Contact) error)(in)
}

// Set a callback for when "RetrieveContact()" is called on the mock Txn.
func (tx *Tx) OnRetrieveContact(fn func(contactID, counterpartyID any) (*models.Contact, error)) {
	tx.callbacks["RetrieveContact"] = fn
}

// Calls the callback previously set with "OnRetrieveContact()".
func (tx *Tx) RetrieveContact(contactID, counterpartyID any) (*models.Contact, error) {
	fn, err := tx.check("RetrieveContact", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(contactID, counterpartyID any) (*models.Contact, error))(contactID, counterpartyID)
}

// Set a callback for when "UpdateContact()" is called on the mock Txn.
func (tx *Tx) OnUpdateContact(fn func(in *models.Contact) error) {
	tx.callbacks["UpdateContact"] = fn
}

// Calls the callback previously set with "OnUpdateContact()".
func (tx *Tx) UpdateContact(in *models.Contact) error {
	fn, err := tx.check("UpdateContact", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Contact) error)(in)
}

// Set a callback for when "DeleteContact()" is called on the mock Txn.
func (tx *Tx) OnDeleteContact(fn func(contactID, counterpartyID any) error) {
	tx.callbacks["DeleteContact"] = fn
}

// Calls the callback previously set with "OnDeleteContact()".
func (tx *Tx) DeleteContact(contactID, counterpartyID any) error {
	fn, err := tx.check("DeleteContact", true)
	if err != nil {
		return err
	}

	return fn.(func(contactID, counterpartyID any) error)(contactID, counterpartyID)
}

//===========================================================================
// Sunrise Interface Methods
//===========================================================================

// Set a callback for when "ListSunrise()" is called on the mock Txn.
func (tx *Tx) OnListSunrise(fn func(in *models.PageInfo) (*models.SunrisePage, error)) {
	tx.callbacks["ListSunrise"] = fn
}

// Calls the callback previously set with "OnListSunrise()".
func (tx *Tx) ListSunrise(in *models.PageInfo) (*models.SunrisePage, error) {
	fn, err := tx.check("ListSunrise", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(in *models.PageInfo) (*models.SunrisePage, error))(in)
}

// Set a callback for when "CreateSunrise()" is called on the mock Txn.
func (tx *Tx) OnCreateSunrise(fn func(in *models.Sunrise) error) {
	tx.callbacks["CreateSunrise"] = fn
}

// Calls the callback previously set with "OnCreateSunrise()".
func (tx *Tx) CreateSunrise(in *models.Sunrise) error {
	fn, err := tx.check("CreateSunrise", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Sunrise) error)(in)
}

// Set a callback for when "RetrieveSunrise()" is called on the mock Txn.
func (tx *Tx) OnRetrieveSunrise(fn func(id ulid.ULID) (*models.Sunrise, error)) {
	tx.callbacks["RetrieveSunrise"] = fn
}

// Calls the callback previously set with "OnRetrieveSunrise()".
func (tx *Tx) RetrieveSunrise(id ulid.ULID) (*models.Sunrise, error) {
	fn, err := tx.check("RetrieveSunrise", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(id ulid.ULID) (*models.Sunrise, error))(id)
}

// Set a callback for when "UpdateSunrise()" is called on the mock Txn.
func (tx *Tx) OnUpdateSunrise(fn func(in *models.Sunrise) error) {
	tx.callbacks["UpdateSunrise"] = fn
}

// Calls the callback previously set with "OnUpdateSunrise()".
func (tx *Tx) UpdateSunrise(in *models.Sunrise) error {
	fn, err := tx.check("UpdateSunrise", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.Sunrise) error)(in)
}

// Set a callback for when "UpdateSunriseStatus()" is called on the mock Txn.
func (tx *Tx) OnUpdateSunriseStatus(fn func(id uuid.UUID, status enum.Status) error) {
	tx.callbacks["UpdateSunriseStatus"] = fn
}

// Calls the callback previously set with "OnUpdateSunriseStatus()".
func (tx *Tx) UpdateSunriseStatus(id uuid.UUID, status enum.Status) error {
	fn, err := tx.check("UpdateSunriseStatus", true)
	if err != nil {
		return err
	}

	return fn.(func(id uuid.UUID, status enum.Status) error)(id, status)
}

// Set a callback for when "DeleteSunrise()" is called on the mock Txn.
func (tx *Tx) OnDeleteSunrise(fn func(in ulid.ULID) error) {
	tx.callbacks["DeleteSunrise"] = fn
}

// Calls the callback previously set with "OnDeleteSunrise()".
func (tx *Tx) DeleteSunrise(in ulid.ULID) error {
	fn, err := tx.check("DeleteSunrise", true)
	if err != nil {
		return err
	}

	return fn.(func(in ulid.ULID) error)(in)
}

// Set a callback for when "GetOrCreateSunriseCounterparty()" is called on the mock Txn.
func (tx *Tx) OnGetOrCreateSunriseCounterparty(fn func(email, name string) (*models.Counterparty, error)) {
	tx.callbacks["GetOrCreateSunriseCounterparty"] = fn
}

// Calls the callback previously set with "OnGetOrCreateSunriseCounterparty()".
func (tx *Tx) GetOrCreateSunriseCounterparty(email, name string) (*models.Counterparty, error) {
	fn, err := tx.check("GetOrCreateSunriseCounterparty", true)
	if err != nil {
		return nil, err
	}

	return fn.(func(email, name string) (*models.Counterparty, error))(email, name)

}

//===========================================================================
// User Interface Methods
//===========================================================================

// Set a callback for when "ListUsers()" is called on the mock Txn.
func (tx *Tx) OnListUsers(fn func(page *models.UserPageInfo) (*models.UserPage, error)) {
	tx.callbacks["ListUsers"] = fn
}

// Calls the callback previously set with "OnListUsers()".
func (tx *Tx) ListUsers(page *models.UserPageInfo) (*models.UserPage, error) {
	fn, err := tx.check("ListUsers", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(page *models.UserPageInfo) (*models.UserPage, error))(page)
}

// Set a callback for when "CreateUser()" is called on the mock Txn.
func (tx *Tx) OnCreateUser(fn func(in *models.User) error) {
	tx.callbacks["CreateUser"] = fn
}

// Calls the callback previously set with "OnCreateUser()".
func (tx *Tx) CreateUser(in *models.User) error {
	fn, err := tx.check("CreateUser", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.User) error)(in)
}

// Set a callback for when "RetrieveUser()" is called on the mock Txn.
func (tx *Tx) OnRetrieveUser(fn func(emailOrUserID any) (*models.User, error)) {
	tx.callbacks["RetrieveUser"] = fn
}

// Calls the callback previously set with "OnRetrieveUser()".
func (tx *Tx) RetrieveUser(emailOrUserID any) (*models.User, error) {
	fn, err := tx.check("RetrieveUser", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(emailOrUserID any) (*models.User, error))(emailOrUserID)
}

// Set a callback for when "UpdateUser()" is called on the mock Txn.
func (tx *Tx) OnUpdateUser(fn func(in *models.User) error) {
	tx.callbacks["UpdateUser"] = fn
}

// Calls the callback previously set with "OnUpdateUser()".
func (tx *Tx) UpdateUser(in *models.User) error {
	fn, err := tx.check("UpdateUser", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.User) error)(in)
}

// Set a callback for when "SetUserPassword()" is called on the mock Txn.
func (tx *Tx) OnSetUserPassword(fn func(userID ulid.ULID, password string) error) {
	tx.callbacks["SetUserPassword"] = fn
}

// Calls the callback previously set with "OnSetUserPassword()".
func (tx *Tx) SetUserPassword(userID ulid.ULID, password string) error {
	fn, err := tx.check("SetUserPassword", true)
	if err != nil {
		return err
	}

	return fn.(func(userID ulid.ULID, password string) error)(userID, password)
}

// Set a callback for when "SetUserLastLogin()" is called on the mock Txn.
func (tx *Tx) OnSetUserLastLogin(fn func(userID ulid.ULID, lastLogin time.Time) error) {
	tx.callbacks["SetUserLastLogin"] = fn
}

// Calls the callback previously set with "OnSetUserLastLogin()".
func (tx *Tx) SetUserLastLogin(userID ulid.ULID, lastLogin time.Time) error {
	fn, err := tx.check("SetUserLastLogin", true)
	if err != nil {
		return err
	}

	return fn.(func(userID ulid.ULID, lastLogin time.Time) error)(userID, lastLogin)
}

// Set a callback for when "DeleteUser()" is called on the mock Txn.
func (tx *Tx) OnDeleteUser(fn func(userID ulid.ULID) error) {
	tx.callbacks["DeleteUser"] = fn
}

// Calls the callback previously set with "OnDeleteUser()".
func (tx *Tx) DeleteUser(userID ulid.ULID) error {
	fn, err := tx.check("DeleteUser", true)
	if err != nil {
		return err
	}

	return fn.(func(userID ulid.ULID) error)(userID)
}

// Set a callback for when "LookupRole()" is called on the mock Txn.
func (tx *Tx) OnLookupRole(fn func(role string) (*models.Role, error)) {
	tx.callbacks["LookupRole"] = fn
}

// Calls the callback previously set with "OnLookupRole()".
func (tx *Tx) LookupRole(role string) (*models.Role, error) {
	fn, err := tx.check("LookupRole", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(role string) (*models.Role, error))(role)
}

//===========================================================================
// APIKey Interface Methods
//===========================================================================

// Set a callback for when "ListAPIKeys()" is called on the mock Txn.
func (tx *Tx) OnListAPIKeys(fn func(page *models.PageInfo) (*models.APIKeyPage, error)) {
	tx.callbacks["ListAPIKeys"] = fn
}

// Calls the callback previously set with "OnListAPIKeys()".
func (tx *Tx) ListAPIKeys(page *models.PageInfo) (*models.APIKeyPage, error) {
	fn, err := tx.check("ListAPIKeys", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(page *models.PageInfo) (*models.APIKeyPage, error))(page)
}

// Set a callback for when "CreateAPIKey()" is called on the mock Txn.
func (tx *Tx) OnCreateAPIKey(fn func(in *models.APIKey) error) {
	tx.callbacks["CreateAPIKey"] = fn
}

// Calls the callback previously set with "OnCreateAPIKey()".
func (tx *Tx) CreateAPIKey(in *models.APIKey) error {
	fn, err := tx.check("CreateAPIKey", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.APIKey) error)(in)
}

// Set a callback for when "RetrieveAPIKey()" is called on the mock Txn.
func (tx *Tx) OnRetrieveAPIKey(fn func(clientIDOrKeyID any) (*models.APIKey, error)) {
	tx.callbacks["RetrieveAPIKey"] = fn
}

// Calls the callback previously set with "OnRetrieveAPIKey()".
func (tx *Tx) RetrieveAPIKey(clientIDOrKeyID any) (*models.APIKey, error) {
	fn, err := tx.check("RetrieveAPIKey", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(clientIDOrKeyID any) (*models.APIKey, error))(clientIDOrKeyID)
}

// Set a callback for when "UpdateAPIKey()" is called on the mock Txn.
func (tx *Tx) OnUpdateAPIKey(fn func(in *models.APIKey) error) {
	tx.callbacks["UpdateAPIKey"] = fn
}

// Calls the callback previously set with "OnUpdateAPIKey()".
func (tx *Tx) UpdateAPIKey(in *models.APIKey) error {
	fn, err := tx.check("UpdateAPIKey", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.APIKey) error)(in)
}

// Set a callback for when "DeleteAPIKey()" is called on the mock Txn.
func (tx *Tx) OnDeleteAPIKey(fn func(keyID ulid.ULID) error) {
	tx.callbacks["DeleteAPIKey"] = fn
}

// Calls the callback previously set with "OnDeleteAPIKey()".
func (tx *Tx) DeleteAPIKey(keyID ulid.ULID) error {
	fn, err := tx.check("DeleteAPIKey", true)
	if err != nil {
		return err
	}

	return fn.(func(keyID ulid.ULID) error)(keyID)
}

//===========================================================================
// ResetPasswordLink Interface Methods
//===========================================================================

// Set a callback for when "ListResetPasswordLinks()" is called on the mock Txn.
func (tx *Tx) OnListResetPasswordLinks(fn func(page *models.PageInfo) (*models.ResetPasswordLinkPage, error)) {
	tx.callbacks["ListResetPasswordLinks"] = fn
}

// Calls the callback previously set with "OnListResetPasswordLinks()".
func (tx *Tx) ListResetPasswordLinks(page *models.PageInfo) (*models.ResetPasswordLinkPage, error) {
	fn, err := tx.check("ListResetPasswordLinks", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(page *models.PageInfo) (*models.ResetPasswordLinkPage, error))(page)
}

// Set a callback for when "CreateResetPasswordLink()" is called on the mock Txn.
func (tx *Tx) OnCreateResetPasswordLink(fn func(in *models.ResetPasswordLink) error) {
	tx.callbacks["CreateResetPasswordLink"] = fn
}

// Calls the callback previously set with "OnCreateResetPasswordLink()".
func (tx *Tx) CreateResetPasswordLink(in *models.ResetPasswordLink) error {
	fn, err := tx.check("CreateResetPasswordLink", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.ResetPasswordLink) error)(in)
}

// Set a callback for when "RetrieveResetPasswordLink()" is called on the mock Txn.
func (tx *Tx) OnRetrieveResetPasswordLink(fn func(id ulid.ULID) (*models.ResetPasswordLink, error)) {
	tx.callbacks["RetrieveResetPasswordLink"] = fn
}

// Calls the callback previously set with "OnRetrieveResetPasswordLink()".
func (tx *Tx) RetrieveResetPasswordLink(id ulid.ULID) (*models.ResetPasswordLink, error) {
	fn, err := tx.check("RetrieveResetPasswordLink", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(id ulid.ULID) (*models.ResetPasswordLink, error))(id)
}

// Set a callback for when "UpdateResetPasswordLink()" is called on the mock Txn.
func (tx *Tx) OnUpdateResetPasswordLink(fn func(in *models.ResetPasswordLink) error) {
	tx.callbacks["UpdateResetPasswordLink"] = fn
}

// Calls the callback previously set with "OnUpdateResetPasswordLink()".
func (tx *Tx) UpdateResetPasswordLink(in *models.ResetPasswordLink) error {
	fn, err := tx.check("UpdateResetPasswordLink", true)
	if err != nil {
		return err
	}

	return fn.(func(in *models.ResetPasswordLink) error)(in)
}

// Set a callback for when "DeleteResetPasswordLink()" is called on the mock Txn.
func (tx *Tx) OnDeleteResetPasswordLink(fn func(id ulid.ULID) error) {
	tx.callbacks["DeleteResetPasswordLink"] = fn
}

// Calls the callback previously set with "OnDeleteResetPasswordLink()".
func (tx *Tx) DeleteResetPasswordLink(id ulid.ULID) error {
	fn, err := tx.check("DeleteResetPasswordLink", true)
	if err != nil {
		return err
	}

	return fn.(func(id ulid.ULID) error)(id)
}

//===========================================================================
// Compliance Audit Log Store Methods
//===========================================================================

// Set a callback for when "ListComplianceAuditLogs()" is called on the mock Txn.
func (tx *Tx) OnListComplianceAuditLogs(fn func(page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error)) {
	tx.callbacks["ListComplianceAuditLogs"] = fn
}

// Calls the callback previously set with "OnListComplianceAuditLogs()".
func (tx *Tx) ListComplianceAuditLogs(page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error) {
	fn, err := tx.check("ListComplianceAuditLogs", false)
	if err != nil {
		return nil, err
	}

	return fn.(func(page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error))(page)
}

// Set a callback for when "CreateComplianceAuditLog()" is called on the mock Txn.
func (tx *Tx) OnCreateComplianceAuditLog(fn func(log *models.ComplianceAuditLog) error) {
	tx.callbacks["CreateComplianceAuditLog"] = fn
}

// Calls the callback previously set with "OnCreateComplianceAuditLog()".
func (tx *Tx) CreateComplianceAuditLog(log *models.ComplianceAuditLog) error {
	fn, err := tx.check("CreateComplianceAuditLog", true)
	if err != nil {
		return err
	}

	return fn.(func(log *models.ComplianceAuditLog) error)(log)
}

//===========================================================================
// Daybreak Interface Methods
//===========================================================================

// Set a callback for when "ListDaybreak()" is called on the mock Txn.
func (tx *Tx) OnListDaybreak(fn func() (map[string]*models.CounterpartySourceInfo, error)) {
	tx.callbacks["ListDaybreak"] = fn
}

// Calls the callback previously set with "OnListDaybreak()".
func (tx *Tx) ListDaybreak() (map[string]*models.CounterpartySourceInfo, error) {
	fn, err := tx.check("ListDaybreak", false)
	if err != nil {
		return nil, err
	}

	return fn.(func() (map[string]*models.CounterpartySourceInfo, error))()
}

// Set a callback for when "CreateDaybreak()" is called on the mock Txn.
func (tx *Tx) OnCreateDaybreak(fn func(counterparty *models.Counterparty) error) {
	tx.callbacks["CreateDaybreak"] = fn
}

// Calls the callback previously set with "OnCreateDaybreak()".
func (tx *Tx) CreateDaybreak(counterparty *models.Counterparty) error {
	fn, err := tx.check("CreateDaybreak", true)
	if err != nil {
		return err
	}

	return fn.(func(counterparty *models.Counterparty) error)(counterparty)
}

// Set a callback for when "UpdateDaybreak()" is called on the mock Txn.
func (tx *Tx) OnUpdateDaybreak(fn func(counterparty *models.Counterparty) error) {
	tx.callbacks["UpdateDaybreak"] = fn
}

// Calls the callback previously set with "OnUpdateDaybreak()".
func (tx *Tx) UpdateDaybreak(counterparty *models.Counterparty) error {
	fn, err := tx.check("UpdateDaybreak", true)
	if err != nil {
		return err
	}

	return fn.(func(counterparty *models.Counterparty) error)(counterparty)
}

// Set a callback for when "DeleteDaybreak()" is called on the mock Txn.
func (tx *Tx) OnDeleteDaybreak(fn func(counterpartyID ulid.ULID, ignoreTxns bool) error) {
	tx.callbacks["DeleteDaybreak"] = fn
}

// Calls the callback previously set with "OnDeleteDaybreak()".
func (tx *Tx) DeleteDaybreak(counterpartyID ulid.ULID, ignoreTxns bool) error {
	fn, err := tx.check("DeleteDaybreak", true)
	if err != nil {
		return err
	}

	return fn.(func(counterpartyID ulid.ULID, ignoreTxns bool) error)(counterpartyID, ignoreTxns)
}
