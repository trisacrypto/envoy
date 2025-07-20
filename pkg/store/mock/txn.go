package mock

import (
	"database/sql"
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
	opts     *sql.TxOptions
	calls    map[string]int
	commit   bool
	rollback bool

	OnCommit                         func() error
	OnRollback                       func() error
	OnListTransactions               func(in *models.TransactionPageInfo) (*models.TransactionPage, error)
	OnCreateTransaction              func(in *models.Transaction, log *models.ComplianceAuditLog) error
	OnRetrieveTransaction            func(id uuid.UUID) (*models.Transaction, error)
	OnUpdateTransaction              func(in *models.Transaction, log *models.ComplianceAuditLog) error
	OnDeleteTransaction              func(id uuid.UUID, log *models.ComplianceAuditLog) error
	OnArchiveTransaction             func(id uuid.UUID, log *models.ComplianceAuditLog) error
	OnUnarchiveTransaction           func(id uuid.UUID, log *models.ComplianceAuditLog) error
	OnCountTransactions              func() (*models.TransactionCounts, error)
	OnTransactionState               func(id uuid.UUID) (bool, enum.Status, error)
	OnListSecureEnvelopes            func(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error)
	OnCreateSecureEnvelope           func(in *models.SecureEnvelope, log *models.ComplianceAuditLog) error
	OnRetrieveSecureEnvelope         func(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error)
	OnUpdateSecureEnvelope           func(in *models.SecureEnvelope, log *models.ComplianceAuditLog) error
	OnDeleteSecureEnvelope           func(txID uuid.UUID, envID ulid.ULID, log *models.ComplianceAuditLog) error
	OnLatestSecureEnvelope           func(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
	OnLatestPayloadEnvelope          func(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
	OnListAccounts                   func(page *models.PageInfo) (*models.AccountsPage, error)
	OnCreateAccount                  func(in *models.Account, log *models.ComplianceAuditLog) error
	OnLookupAccount                  func(cryptoAddress string) (*models.Account, error)
	OnRetrieveAccount                func(id ulid.ULID) (*models.Account, error)
	OnUpdateAccount                  func(in *models.Account, log *models.ComplianceAuditLog) error
	OnDeleteAccount                  func(id ulid.ULID, log *models.ComplianceAuditLog) error
	OnListAccountTransactions        func(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error)
	OnListCryptoAddresses            func(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error)
	OnCreateCryptoAddress            func(in *models.CryptoAddress, log *models.ComplianceAuditLog) error
	OnRetrieveCryptoAddress          func(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error)
	OnUpdateCryptoAddress            func(in *models.CryptoAddress, log *models.ComplianceAuditLog) error
	OnDeleteCryptoAddress            func(accountID, cryptoAddressID ulid.ULID, log *models.ComplianceAuditLog) error
	OnSearchCounterparties           func(query *models.SearchQuery) (*models.CounterpartyPage, error)
	OnListCounterparties             func(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error)
	OnListCounterpartySourceInfo     func(source enum.Source) ([]*models.CounterpartySourceInfo, error)
	OnCreateCounterparty             func(in *models.Counterparty, log *models.ComplianceAuditLog) error
	OnRetrieveCounterparty           func(counterpartyID ulid.ULID) (*models.Counterparty, error)
	OnLookupCounterparty             func(field, value string) (*models.Counterparty, error)
	OnUpdateCounterparty             func(in *models.Counterparty, log *models.ComplianceAuditLog) error
	OnDeleteCounterparty             func(counterpartyID ulid.ULID, log *models.ComplianceAuditLog) error
	OnListContacts                   func(counterparty any, page *models.PageInfo) (*models.ContactsPage, error)
	OnCreateContact                  func(in *models.Contact, log *models.ComplianceAuditLog) error
	OnRetrieveContact                func(contactID, counterpartyID any) (*models.Contact, error)
	OnUpdateContact                  func(in *models.Contact, log *models.ComplianceAuditLog) error
	OnDeleteContact                  func(contactID, counterpartyID any, log *models.ComplianceAuditLog) error
	OnListSunrise                    func(in *models.PageInfo) (*models.SunrisePage, error)
	OnCreateSunrise                  func(in *models.Sunrise, log *models.ComplianceAuditLog) error
	OnRetrieveSunrise                func(id ulid.ULID) (*models.Sunrise, error)
	OnUpdateSunrise                  func(in *models.Sunrise, log *models.ComplianceAuditLog) error
	OnUpdateSunriseStatus            func(id uuid.UUID, status enum.Status, log *models.ComplianceAuditLog) error
	OnDeleteSunrise                  func(in ulid.ULID, log *models.ComplianceAuditLog) error
	OnGetOrCreateSunriseCounterparty func(email, name string, log *models.ComplianceAuditLog) (*models.Counterparty, error)
	OnListUsers                      func(page *models.UserPageInfo) (*models.UserPage, error)
	OnCreateUser                     func(in *models.User, log *models.ComplianceAuditLog) error
	OnRetrieveUser                   func(emailOrUserID any) (*models.User, error)
	OnUpdateUser                     func(in *models.User, log *models.ComplianceAuditLog) error
	OnSetUserPassword                func(userID ulid.ULID, password string) error
	OnSetUserLastLogin               func(userID ulid.ULID, lastLogin time.Time) error
	OnDeleteUser                     func(userID ulid.ULID, log *models.ComplianceAuditLog) error
	OnLookupRole                     func(role string) (*models.Role, error)
	OnListAPIKeys                    func(page *models.PageInfo) (*models.APIKeyPage, error)
	OnCreateAPIKey                   func(in *models.APIKey, log *models.ComplianceAuditLog) error
	OnRetrieveAPIKey                 func(clientIDOrKeyID any) (*models.APIKey, error)
	OnUpdateAPIKey                   func(in *models.APIKey, log *models.ComplianceAuditLog) error
	OnDeleteAPIKey                   func(keyID ulid.ULID, log *models.ComplianceAuditLog) error
	OnListResetPasswordLinks         func(page *models.PageInfo) (*models.ResetPasswordLinkPage, error)
	OnCreateResetPasswordLink        func(in *models.ResetPasswordLink) error
	OnRetrieveResetPasswordLink      func(id ulid.ULID) (*models.ResetPasswordLink, error)
	OnUpdateResetPasswordLink        func(in *models.ResetPasswordLink) error
	OnDeleteResetPasswordLink        func(id ulid.ULID) error
	OnListComplianceAuditLogs        func(page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error)
	OnCreateComplianceAuditLog       func(log *models.ComplianceAuditLog) error
	OnRetrieveComplianceAuditLog     func(id ulid.ULID) (*models.ComplianceAuditLog, error)
	OnListDaybreak                   func() (map[string]*models.CounterpartySourceInfo, error)
	OnCreateDaybreak                 func(counterparty *models.Counterparty) error
	OnUpdateDaybreak                 func(counterparty *models.Counterparty) error
	OnDeleteDaybreak                 func(counterpartyID ulid.ULID, ignoreTxns bool) error
}

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Reset all the calls and callbacks in the transaction, if you don't want to
// create a new one.
func (tx *Tx) Reset() {
	// Set map to nil to free up memory
	tx.calls = nil

	// Create new calls map
	tx.calls = make(map[string]int)

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
// is read-only.
func (tx *Tx) check(writable bool) error {
	if tx.commit || tx.rollback {
		return sql.ErrTxDone
	}

	if writable && tx.opts != nil && tx.opts.ReadOnly {
		return errors.ErrReadOnly
	}

	return nil
}

//===========================================================================
// Txn Interface Methods
//===========================================================================

// Calls the callback previously set with "OnCommit()", or completes the
// "commit" for the transaction.
func (tx *Tx) Commit() error {
	tx.calls["Commit"]++
	if tx.OnCommit != nil {
		return tx.OnCommit()
	}

	// ensure the transaction is still active
	if tx.commit || tx.rollback {
		return sql.ErrTxDone
	}

	tx.commit = true
	return nil
}

// Calls the callback previously set with "OnRollback()", or completes the
// "rollback" for the transaction.
func (tx *Tx) Rollback() error {
	tx.calls["Rollback"]++
	if tx.OnRollback != nil {
		return tx.OnRollback()
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

// Calls the callback previously set with "OnListTransactions()".
func (tx *Tx) ListTransactions(in *models.TransactionPageInfo) (*models.TransactionPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListTransactions != nil {
		return tx.OnListTransactions(in)
	}
	panic("ListTransactions callback not set")
}

// Calls the callback previously set with "OnCreateTransaction()".
func (tx *Tx) CreateTransaction(in *models.Transaction, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateTransaction != nil {
		return tx.OnCreateTransaction(in, log)
	}
	panic("CreateTransaction callback not set")
}

// Calls the callback previously set with "OnRetrieveTransaction()".
func (tx *Tx) RetrieveTransaction(id uuid.UUID) (*models.Transaction, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveTransaction != nil {
		return tx.OnRetrieveTransaction(id)
	}
	panic("RetrieveTransaction callback not set")
}

// Calls the callback previously set with "OnUpdateTransaction()".
func (tx *Tx) UpdateTransaction(in *models.Transaction, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateTransaction != nil {
		return tx.OnUpdateTransaction(in, log)
	}
	panic("UpdateTransaction callback not set")
}

// Calls the callback previously set with "OnDeleteTransaction()".
func (tx *Tx) DeleteTransaction(id uuid.UUID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteTransaction != nil {
		return tx.OnDeleteTransaction(id, log)
	}
	panic("DeleteTransaction callback not set")
}

// Calls the callback previously set with "OnArchiveTransaction()".
func (tx *Tx) ArchiveTransaction(id uuid.UUID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnArchiveTransaction != nil {
		return tx.OnArchiveTransaction(id, log)
	}
	panic("ArchiveTransaction callback not set")
}

// Calls the callback previously set with "OnUnarchiveTransaction()".
func (tx *Tx) UnarchiveTransaction(id uuid.UUID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUnarchiveTransaction != nil {
		return tx.OnUnarchiveTransaction(id, log)
	}
	panic("UnarchiveTransaction callback not set")
}

// Calls the callback previously set with "OnCountTransactions()".
func (tx *Tx) CountTransactions() (*models.TransactionCounts, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnCountTransactions != nil {
		return tx.OnCountTransactions()
	}
	panic("CountTransactions callback not set")
}

// Calls the callback previously set with "OnTransactionState()".
func (tx *Tx) TransactionState(id uuid.UUID) (bool, enum.Status, error) {
	if err := tx.check(false); err != nil {
		return false, enum.StatusUnspecified, err
	}

	if tx.OnTransactionState != nil {
		return tx.OnTransactionState(id)
	}
	panic("TransactionState callback not set")
}

//===========================================================================
// SecureEnvelope Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListSecureEnvelopes()".
func (tx *Tx) ListSecureEnvelopes(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListSecureEnvelopes != nil {
		return tx.OnListSecureEnvelopes(txID, page)
	}
	panic("ListSecureEnvelopes callback not set")
}

// Calls the callback previously set with "OnCreateSecureEnvelope()".
func (tx *Tx) CreateSecureEnvelope(in *models.SecureEnvelope, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateSecureEnvelope != nil {
		return tx.OnCreateSecureEnvelope(in, log)
	}
	panic("CreateSecureEnvelope callback not set")
}

// Calls the callback previously set with "OnRetrieveSecureEnvelope()".
func (tx *Tx) RetrieveSecureEnvelope(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveSecureEnvelope != nil {
		return tx.OnRetrieveSecureEnvelope(txID, envID)
	}
	panic("RetrieveSecureEnvelope callback not set")
}

// Calls the callback previously set with "OnUpdateSecureEnvelope()".
func (tx *Tx) UpdateSecureEnvelope(in *models.SecureEnvelope, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateSecureEnvelope != nil {
		return tx.OnUpdateSecureEnvelope(in, log)
	}
	panic("UpdateSecureEnvelope callback not set")
}

// Calls the callback previously set with "OnDeleteSecureEnvelope()".
func (tx *Tx) DeleteSecureEnvelope(txID uuid.UUID, envID ulid.ULID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteSecureEnvelope != nil {
		return tx.OnDeleteSecureEnvelope(txID, envID, log)
	}
	panic("DeleteSecureEnvelope callback not set")
}

// Calls the callback previously set with "OnLatestSecureEnvelope()".
func (tx *Tx) LatestSecureEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnLatestSecureEnvelope != nil {
		return tx.OnLatestSecureEnvelope(txID, direction)
	}
	panic("LatestSecureEnvelope callback not set")
}

// Calls the callback previously set with "OnLatestPayloadEnvelope()".
func (tx *Tx) LatestPayloadEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnLatestPayloadEnvelope != nil {
		return tx.OnLatestPayloadEnvelope(txID, direction)
	}
	panic("LatestPayloadEnvelope callback not set")
}

//===========================================================================
// Account Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListAccounts()".
func (tx *Tx) ListAccounts(page *models.PageInfo) (*models.AccountsPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListAccounts != nil {
		return tx.OnListAccounts(page)
	}
	panic("ListAccounts callback not set")
}

// Calls the callback previously set with "OnCreateAccount()".
func (tx *Tx) CreateAccount(in *models.Account, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateAccount != nil {
		return tx.OnCreateAccount(in, log)
	}
	panic("CreateAccount callback not set")
}

// Calls the callback previously set with "OnLookupAccount()".
func (tx *Tx) LookupAccount(cryptoAddress string) (*models.Account, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnLookupAccount != nil {
		return tx.OnLookupAccount(cryptoAddress)
	}
	panic("LookupAccount callback not set")
}

// Calls the callback previously set with "OnRetrieveAccount()".
func (tx *Tx) RetrieveAccount(id ulid.ULID) (*models.Account, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveAccount != nil {
		return tx.OnRetrieveAccount(id)
	}
	panic("RetrieveAccount callback not set")
}

// Calls the callback previously set with "OnUpdateAccount()".
func (tx *Tx) UpdateAccount(in *models.Account, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateAccount != nil {
		return tx.OnUpdateAccount(in, log)
	}
	panic("UpdateAccount callback not set")
}

// Calls the callback previously set with "OnDeleteAccount()".
func (tx *Tx) DeleteAccount(id ulid.ULID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteAccount != nil {
		return tx.OnDeleteAccount(id, log)
	}
	panic("DeleteAccount callback not set")
}

// Calls the callback previously set with "OnListAccountTransactions()".
func (tx *Tx) ListAccountTransactions(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListAccountTransactions != nil {
		return tx.OnListAccountTransactions(accountID, page)
	}
	panic("ListAccountTransactions callback not set")
}

//===========================================================================
// CryptoAddress Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListCryptoAddresses()".
func (tx *Tx) ListCryptoAddresses(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListCryptoAddresses != nil {
		return tx.OnListCryptoAddresses(accountID, page)
	}
	panic("ListCryptoAddresses callback not set")
}

// Calls the callback previously set with "OnCreateCryptoAddress()".
func (tx *Tx) CreateCryptoAddress(in *models.CryptoAddress, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateCryptoAddress != nil {
		return tx.OnCreateCryptoAddress(in, log)
	}
	panic("CreateCryptoAddress callback not set")
}

// Calls the callback previously set with "OnRetrieveCryptoAddress()".
func (tx *Tx) RetrieveCryptoAddress(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveCryptoAddress != nil {
		return tx.OnRetrieveCryptoAddress(accountID, cryptoAddressID)
	}
	panic("RetrieveCryptoAddress callback not set")
}

// Calls the callback previously set with "OnUpdateCryptoAddress()".
func (tx *Tx) UpdateCryptoAddress(in *models.CryptoAddress, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateCryptoAddress != nil {
		return tx.OnUpdateCryptoAddress(in, log)
	}
	panic("UpdateCryptoAddress callback not set")
}

// Calls the callback previously set with "OnDeleteCryptoAddress()".
func (tx *Tx) DeleteCryptoAddress(accountID, cryptoAddressID ulid.ULID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteCryptoAddress != nil {
		return tx.OnDeleteCryptoAddress(accountID, cryptoAddressID, log)
	}
	panic("DeleteCryptoAddress callback not set")
}

//===========================================================================
// Counterparty Interface Methods
//===========================================================================

// Calls the callback previously set with "OnSearchCounterparties()".
func (tx *Tx) SearchCounterparties(query *models.SearchQuery) (*models.CounterpartyPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnSearchCounterparties != nil {
		return tx.OnSearchCounterparties(query)
	}
	panic("SearchCounterparties callback not set")
}

// Calls the callback previously set with "OnListCounterparties()".
func (tx *Tx) ListCounterparties(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListCounterparties != nil {
		return tx.OnListCounterparties(page)
	}
	panic("ListCounterparties callback not set")
}

// Calls the callback previously set with "OnListCounterpartySourceInfo()".
func (tx *Tx) ListCounterpartySourceInfo(source enum.Source) ([]*models.CounterpartySourceInfo, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListCounterpartySourceInfo != nil {
		return tx.OnListCounterpartySourceInfo(source)
	}
	panic("ListCounterpartySourceInfo callback not set")
}

// Calls the callback previously set with "OnCreateCounterparty()".
func (tx *Tx) CreateCounterparty(in *models.Counterparty, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateCounterparty != nil {
		return tx.OnCreateCounterparty(in, log)
	}
	panic("CreateCounterparty callback not set")
}

// Calls the callback previously set with "OnRetrieveCounterparty()".
func (tx *Tx) RetrieveCounterparty(counterpartyID ulid.ULID) (*models.Counterparty, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveCounterparty != nil {
		return tx.OnRetrieveCounterparty(counterpartyID)
	}
	panic("RetrieveCounterparty callback not set")
}

// Calls the callback previously set with "OnLookupCounterparty()".
func (tx *Tx) LookupCounterparty(field, value string) (*models.Counterparty, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnLookupCounterparty != nil {
		return tx.OnLookupCounterparty(field, value)
	}
	panic("LookupCounterparty callback not set")

}

// Calls the callback previously set with "OnUpdateCounterparty()".
func (tx *Tx) UpdateCounterparty(in *models.Counterparty, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateCounterparty != nil {
		return tx.OnUpdateCounterparty(in, log)
	}
	panic("UpdateCounterparty callback not set")
}

// Calls the callback previously set with "OnDeleteCounterparty()".
func (tx *Tx) DeleteCounterparty(counterpartyID ulid.ULID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteCounterparty != nil {
		return tx.OnDeleteCounterparty(counterpartyID, log)
	}
	panic("DeleteCounterparty callback not set")
}

//===========================================================================
// Contact Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListContacts()".
func (tx *Tx) ListContacts(counterparty any, page *models.PageInfo) (*models.ContactsPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListContacts != nil {
		return tx.OnListContacts(counterparty, page)
	}
	panic("ListContacts callback not set")
}

// Calls the callback previously set with "OnCreateContact()".
func (tx *Tx) CreateContact(in *models.Contact, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateContact != nil {
		return tx.OnCreateContact(in, log)
	}
	panic("CreateContact callback not set")
}

// Calls the callback previously set with "OnRetrieveContact()".
func (tx *Tx) RetrieveContact(contactID, counterpartyID any) (*models.Contact, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveContact != nil {
		return tx.OnRetrieveContact(contactID, counterpartyID)
	}
	panic("RetrieveContact callback not set")
}

// Calls the callback previously set with "OnUpdateContact()".
func (tx *Tx) UpdateContact(in *models.Contact, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateContact != nil {
		return tx.OnUpdateContact(in, log)
	}
	panic("UpdateContact callback not set")
}

// Calls the callback previously set with "OnDeleteContact()".
func (tx *Tx) DeleteContact(contactID, counterpartyID any, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteContact != nil {
		return tx.OnDeleteContact(contactID, counterpartyID, log)
	}
	panic("DeleteContact callback not set")
}

//===========================================================================
// Sunrise Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListSunrise()".
func (tx *Tx) ListSunrise(in *models.PageInfo) (*models.SunrisePage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListSunrise != nil {
		return tx.OnListSunrise(in)
	}
	panic("ListSunrise callback not set")
}

// Calls the callback previously set with "OnCreateSunrise()".
func (tx *Tx) CreateSunrise(in *models.Sunrise, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateSunrise != nil {
		return tx.OnCreateSunrise(in, log)
	}
	panic("CreateSunrise callback not set")
}

// Calls the callback previously set with "OnRetrieveSunrise()".
func (tx *Tx) RetrieveSunrise(id ulid.ULID) (*models.Sunrise, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveSunrise != nil {
		return tx.OnRetrieveSunrise(id)
	}
	panic("RetrieveSunrise callback not set")
}

// Calls the callback previously set with "OnUpdateSunrise()".
func (tx *Tx) UpdateSunrise(in *models.Sunrise, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateSunrise != nil {
		return tx.OnUpdateSunrise(in, log)
	}
	panic("UpdateSunrise callback not set")
}

// Calls the callback previously set with "OnUpdateSunriseStatus()".
func (tx *Tx) UpdateSunriseStatus(id uuid.UUID, status enum.Status, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateSunriseStatus != nil {
		return tx.OnUpdateSunriseStatus(id, status, log)
	}
	panic("UpdateSunriseStatus callback not set")
}

// Calls the callback previously set with "OnDeleteSunrise()".
func (tx *Tx) DeleteSunrise(in ulid.ULID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteSunrise != nil {
		return tx.OnDeleteSunrise(in, log)
	}
	panic("DeleteSunrise callback not set")
}

// Calls the callback previously set with "OnGetOrCreateSunriseCounterparty()".
func (tx *Tx) GetOrCreateSunriseCounterparty(email, name string, log *models.ComplianceAuditLog) (*models.Counterparty, error) {
	if err := tx.check(true); err != nil {
		return nil, err
	}

	if tx.OnGetOrCreateSunriseCounterparty != nil {
		return tx.OnGetOrCreateSunriseCounterparty(email, name, log)
	}
	panic("GetOrCreateSunriseCounterparty callback not set")

}

//===========================================================================
// User Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListUsers()".
func (tx *Tx) ListUsers(page *models.UserPageInfo) (*models.UserPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListUsers != nil {
		return tx.OnListUsers(page)
	}
	panic("ListUsers callback not set")
}

// Calls the callback previously set with "OnCreateUser()".
func (tx *Tx) CreateUser(in *models.User, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateUser != nil {
		return tx.OnCreateUser(in, log)
	}
	panic("CreateUser callback not set")
}

// Calls the callback previously set with "OnRetrieveUser()".
func (tx *Tx) RetrieveUser(emailOrUserID any) (*models.User, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveUser != nil {
		return tx.OnRetrieveUser(emailOrUserID)
	}
	panic("RetrieveUser callback not set")
}

// Calls the callback previously set with "OnUpdateUser()".
func (tx *Tx) UpdateUser(in *models.User, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateUser != nil {
		return tx.OnUpdateUser(in, log)
	}
	panic("UpdateUser callback not set")
}

// Calls the callback previously set with "OnSetUserPassword()".
func (tx *Tx) SetUserPassword(userID ulid.ULID, password string) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnSetUserPassword != nil {
		return tx.OnSetUserPassword(userID, password)
	}
	panic("SetUserPassword callback not set")
}

// Calls the callback previously set with "OnSetUserLastLogin()".
func (tx *Tx) SetUserLastLogin(userID ulid.ULID, lastLogin time.Time) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnSetUserLastLogin != nil {
		return tx.OnSetUserLastLogin(userID, lastLogin)
	}
	panic("SetUserLastLogin callback not set")
}

// Calls the callback previously set with "OnDeleteUser()".
func (tx *Tx) DeleteUser(userID ulid.ULID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteUser != nil {
		return tx.OnDeleteUser(userID, log)
	}
	panic("DeleteUser callback not set")
}

// Calls the callback previously set with "OnLookupRole()".
func (tx *Tx) LookupRole(role string) (*models.Role, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnLookupRole != nil {
		return tx.OnLookupRole(role)
	}
	panic("LookupRole callback not set")
}

//===========================================================================
// APIKey Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListAPIKeys()".
func (tx *Tx) ListAPIKeys(page *models.PageInfo) (*models.APIKeyPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListAPIKeys != nil {
		return tx.OnListAPIKeys(page)
	}
	panic("ListAPIKeys callback not set")
}

// Calls the callback previously set with "OnCreateAPIKey()".
func (tx *Tx) CreateAPIKey(in *models.APIKey, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateAPIKey != nil {
		return tx.OnCreateAPIKey(in, log)
	}
	panic("CreateAPIKey callback not set")
}

// Calls the callback previously set with "OnRetrieveAPIKey()".
func (tx *Tx) RetrieveAPIKey(clientIDOrKeyID any) (*models.APIKey, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveAPIKey != nil {
		return tx.OnRetrieveAPIKey(clientIDOrKeyID)
	}
	panic("RetrieveAPIKey callback not set")
}

// Calls the callback previously set with "OnUpdateAPIKey()".
func (tx *Tx) UpdateAPIKey(in *models.APIKey, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateAPIKey != nil {
		return tx.OnUpdateAPIKey(in, log)
	}
	panic("UpdateAPIKey callback not set")
}

// Calls the callback previously set with "OnDeleteAPIKey()".
func (tx *Tx) DeleteAPIKey(keyID ulid.ULID, log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteAPIKey != nil {
		return tx.OnDeleteAPIKey(keyID, log)
	}
	panic("DeleteAPIKey callback not set")
}

//===========================================================================
// ResetPasswordLink Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListResetPasswordLinks()".
func (tx *Tx) ListResetPasswordLinks(page *models.PageInfo) (*models.ResetPasswordLinkPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListResetPasswordLinks != nil {
		return tx.OnListResetPasswordLinks(page)
	}
	panic("ListResetPasswordLinks callback not set")
}

// Calls the callback previously set with "OnCreateResetPasswordLink()".
func (tx *Tx) CreateResetPasswordLink(in *models.ResetPasswordLink) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateResetPasswordLink != nil {
		return tx.OnCreateResetPasswordLink(in)
	}
	panic("CreateResetPasswordLink callback not set")
}

// Calls the callback previously set with "OnRetrieveResetPasswordLink()".
func (tx *Tx) RetrieveResetPasswordLink(id ulid.ULID) (*models.ResetPasswordLink, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnRetrieveResetPasswordLink != nil {
		return tx.OnRetrieveResetPasswordLink(id)
	}
	panic("RetrieveResetPasswordLink callback not set")
}

// Calls the callback previously set with "OnUpdateResetPasswordLink()".
func (tx *Tx) UpdateResetPasswordLink(in *models.ResetPasswordLink) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateResetPasswordLink != nil {
		return tx.OnUpdateResetPasswordLink(in)
	}
	panic("UpdateResetPasswordLink callback not set")
}

// Calls the callback previously set with "OnDeleteResetPasswordLink()".
func (tx *Tx) DeleteResetPasswordLink(id ulid.ULID) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteResetPasswordLink != nil {
		return tx.OnDeleteResetPasswordLink(id)
	}
	panic("DeleteResetPasswordLink callback not set")
}

//===========================================================================
// Compliance Audit Log Store Methods
//===========================================================================

// Calls the callback previously set with "OnListComplianceAuditLogs()".
func (tx *Tx) ListComplianceAuditLogs(page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListComplianceAuditLogs != nil {
		return tx.OnListComplianceAuditLogs(page)
	}
	panic("ListComplianceAuditLogs callback not set")
}

// Calls the callback previously set with "OnCreateComplianceAuditLog()".
func (tx *Tx) CreateComplianceAuditLog(log *models.ComplianceAuditLog) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateComplianceAuditLog != nil {
		return tx.OnCreateComplianceAuditLog(log)
	}
	panic("CreateComplianceAuditLog callback not set")
}

// Calls the callback previously set with "OnCreateComplianceAuditLog()".
func (tx *Tx) RetrieveComplianceAuditLog(id ulid.ULID) (*models.ComplianceAuditLog, error) {
	if err := tx.check(true); err != nil {
		return nil, err
	}

	if tx.OnRetrieveComplianceAuditLog != nil {
		return tx.OnRetrieveComplianceAuditLog(id)
	}
	panic("RetrieveComplianceAuditLog callback not set")
}

//===========================================================================
// Daybreak Interface Methods
//===========================================================================

// Calls the callback previously set with "OnListDaybreak()".
func (tx *Tx) ListDaybreak() (map[string]*models.CounterpartySourceInfo, error) {
	if err := tx.check(false); err != nil {
		return nil, err
	}

	if tx.OnListDaybreak != nil {
		return tx.OnListDaybreak()
	}
	panic("ListDaybreak callback not set")
}

// Calls the callback previously set with "OnCreateDaybreak()".
func (tx *Tx) CreateDaybreak(counterparty *models.Counterparty) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnCreateDaybreak != nil {
		return tx.OnCreateDaybreak(counterparty)
	}
	panic("CreateDaybreak callback not set")
}

// Calls the callback previously set with "OnUpdateDaybreak()".
func (tx *Tx) UpdateDaybreak(counterparty *models.Counterparty) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnUpdateDaybreak != nil {
		return tx.OnUpdateDaybreak(counterparty)
	}
	panic("UpdateDaybreak callback not set")
}

// Calls the callback previously set with "OnDeleteDaybreak()".
func (tx *Tx) DeleteDaybreak(counterpartyID ulid.ULID, ignoreTxns bool) error {
	if err := tx.check(true); err != nil {
		return err
	}

	if tx.OnDeleteDaybreak != nil {
		return tx.OnDeleteDaybreak(counterpartyID, ignoreTxns)
	}
	panic("DeleteDaybreak callback not set")
}
