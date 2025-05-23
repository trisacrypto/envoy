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
// Tx Callback Types
//===========================================================================

type EmptyAnyFn func() (any, error)
type ListTxnFn func(any) (any, error)
type CreateUpdateTxnFn func(any) error
type RetrieveTxnAnyFn func(any) (any, error)
type RetrieveTxnUUIDFn func(uuid.UUID) (any, error)
type RetrieveTxnULIDFn func(ulid.ULID) (any, error)
type DeleteTxnUUIDFn func(uuid.UUID) error
type DeleteTxnULIDFn func(ulid.ULID) error
type TransactionStatusTxnFn func(uuid.UUID) (bool, enum.Status, error)
type ListRetrieveAssocTxnFn func(any, any) (any, error)
type ListRetrieveAssocTxnUUIDFn func(uuid.UUID, any) (any, error)
type ListRetrieveAssocTxnULIDFn func(ulid.ULID, any) (any, error)
type DeleteActionAssocTxnFn func(any, any) error
type LookupAssocTxnFn func(any, any) (any, error)

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Assert that the expected number of calls were made to the given method.
func (tx *Tx) AssertCalls(t testing.TB, method string, expected int) {
	require.Equal(t, expected, tx.calls[method], "expected %d calls to %s, got %d", expected, method, tx.calls[method])
}

// Assert that Commit has been called on the transaction.
func (tx *Tx) AssertCommit(t testing.TB) {
	require.True(t, tx.commit, "expected Commit to be called")
}

// Assert that Rollback has been called on the transaction without commit.
func (tx *Tx) AssertRollback(t testing.TB) {
	require.True(t, tx.rollback && !tx.commit, "expected Rollback to be called but not Commit")
}

// Set a callback for the given method. Make sure to use one of the callback types and
// not the function signature itself. If a method is called without a callback set,
// the test will panic. The test will also panic if return type assertions aren't met.
// NOTE: if a callback is already set, it will be overwritten but the calls will not be.
func (tx *Tx) On(method string, fn any) {
	tx.callbacks[method] = fn
}

// Have a method return the specified error.
// NOTE: you may have to add a case to this method if you're implementing a test that
// uses that call back for the first time. Try to group test cases together if possible.
func (tx *Tx) Err(method string, err error) {
	switch method {
	case "Commit", "Rollback":
		tx.callbacks[method] = func() error { return err }
	default:
		panic(fmt.Errorf("method %q not implemented yet", method))
	}
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

func (tx *Tx) Commit() error {
	tx.calls["Commit"]++
	if fn, ok := tx.callbacks["Commit"]; ok {
		return fn.(ErrorFn)()
	}

	// Prevent calling commit multiple times.
	if tx.commit {
		return sql.ErrTxDone
	}

	tx.commit = true
	return nil
}

func (tx *Tx) Rollback() error {
	tx.calls["Rollback"]++
	if fn, ok := tx.callbacks["Rollback"]; ok {
		return fn.(ErrorFn)()
	}

	// Prevent calling rollback multiple times.
	if tx.rollback {
		return sql.ErrTxDone
	}

	tx.rollback = true
	return nil
}

//===========================================================================
// Transaction Interface Methods
//===========================================================================

func (tx *Tx) ListTransactions(in *models.TransactionPageInfo) (*models.TransactionPage, error) {
	fn, err := tx.check("ListTransactions", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(in)
	return out.(*models.TransactionPage), err
}

func (tx *Tx) CreateTransaction(in *models.Transaction) error {
	fn, err := tx.check("CreateTransaction", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveTransaction(id uuid.UUID) (*models.Transaction, error) {
	fn, err := tx.check("RetrieveTransaction", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnUUIDFn)(id)
	if err != nil {
		return nil, err
	}

	return out.(*models.Transaction), nil
}

func (tx *Tx) UpdateTransaction(in *models.Transaction) error {
	fn, err := tx.check("UpdateTransaction", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) DeleteTransaction(id uuid.UUID) error {
	fn, err := tx.check("DeleteTransaction", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnUUIDFn)(id)
}

func (tx *Tx) ArchiveTransaction(id uuid.UUID) error {
	fn, err := tx.check("ArchiveTransaction", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnUUIDFn)(id)
}

func (tx *Tx) UnarchiveTransaction(id uuid.UUID) error {
	fn, err := tx.check("UnarchiveTransaction", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnUUIDFn)(id)
}

func (tx *Tx) CountTransactions() (*models.TransactionCounts, error) {
	fn, err := tx.check("CountTransactions", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(EmptyAnyFn)()
	if err != nil {
		return nil, err
	}
	return out.(*models.TransactionCounts), nil
}

func (tx *Tx) TransactionState(id uuid.UUID) (bool, enum.Status, error) {
	fn, err := tx.check("TransactionState", false)
	if err != nil {
		return false, enum.StatusUnspecified, err
	}

	archived, status, err := fn.(TransactionStatusTxnFn)(id)
	if err != nil {
		return false, enum.StatusUnspecified, err
	}
	return archived, status, nil
}

//===========================================================================
// SecureEnvelope Interface Methods
//===========================================================================

func (tx *Tx) ListSecureEnvelopes(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	fn, err := tx.check("ListSecureEnvelopes", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnUUIDFn)(txID, page)
	if err != nil {
		return nil, err
	}

	return out.(*models.SecureEnvelopePage), nil
}

func (tx *Tx) CreateSecureEnvelope(in *models.SecureEnvelope) error {
	fn, err := tx.check("CreateSecureEnvelope", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveSecureEnvelope(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	fn, err := tx.check("RetrieveSecureEnvelope", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnUUIDFn)(txID, envID)
	if err != nil {
		return nil, err
	}

	return out.(*models.SecureEnvelope), nil
}

func (tx *Tx) UpdateSecureEnvelope(in *models.SecureEnvelope) error {
	fn, err := tx.check("UpdateSecureEnvelope", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) DeleteSecureEnvelope(txID uuid.UUID, envID ulid.ULID) error {
	fn, err := tx.check("DeleteSecureEnvelope", true)
	if err != nil {
		return err
	}

	return fn.(DeleteActionAssocTxnFn)(txID, envID)
}

func (tx *Tx) LatestSecureEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	fn, err := tx.check("LatestSecureEnvelope", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnUUIDFn)(txID, direction)
	if err != nil {
		return nil, err
	}

	return out.(*models.SecureEnvelope), nil
}

func (tx *Tx) LatestPayloadEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	fn, err := tx.check("LatestPayloadEnvelope", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnUUIDFn)(txID, direction)
	if err != nil {
		return nil, err
	}

	return out.(*models.SecureEnvelope), nil
}

//===========================================================================
// Account Interface Methods
//===========================================================================

func (tx *Tx) ListAccounts(page *models.PageInfo) (*models.AccountsPage, error) {
	fn, err := tx.check("ListAccounts", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(page)
	if err != nil {
		return nil, err
	}

	return out.(*models.AccountsPage), nil
}

func (tx *Tx) CreateAccount(in *models.Account) error {
	fn, err := tx.check("CreateAccount", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) LookupAccount(cryptoAddress string) (*models.Account, error) {
	fn, err := tx.check("LookupAccount", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnAnyFn)(cryptoAddress)
	if err != nil {
		return nil, err
	}

	return out.(*models.Account), nil
}

func (tx *Tx) RetrieveAccount(id ulid.ULID) (*models.Account, error) {
	fn, err := tx.check("RetrieveAccount", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnULIDFn)(id)
	if err != nil {
		return nil, err
	}

	return out.(*models.Account), nil
}

func (tx *Tx) UpdateAccount(in *models.Account) error {
	fn, err := tx.check("UpdateAccount", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) DeleteAccount(id ulid.ULID) error {
	fn, err := tx.check("DeleteAccount", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnULIDFn)(id)
}

func (tx *Tx) ListAccountTransactions(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error) {
	fn, err := tx.check("ListAccountTransactions", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnULIDFn)(accountID, page)
	if err != nil {
		return nil, err
	}

	return out.(*models.TransactionPage), nil
}

//===========================================================================
// CryptoAddress Interface Methods
//===========================================================================

func (tx *Tx) ListCryptoAddresses(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	fn, err := tx.check("ListCryptoAddresses", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnULIDFn)(accountID, page)
	if err != nil {
		return nil, err
	}

	return out.(*models.CryptoAddressPage), nil
}

func (tx *Tx) CreateCryptoAddress(in *models.CryptoAddress) error {
	fn, err := tx.check("CreateCryptoAddress", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveCryptoAddress(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error) {
	fn, err := tx.check("RetrieveCryptoAddress", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnULIDFn)(accountID, cryptoAddressID)
	if err != nil {
		return nil, err
	}

	return out.(*models.CryptoAddress), nil
}

func (tx *Tx) UpdateCryptoAddress(in *models.CryptoAddress) error {
	fn, err := tx.check("UpdateCryptoAddress", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) DeleteCryptoAddress(accountID, cryptoAddressID ulid.ULID) error {
	fn, err := tx.check("DeleteCryptoAddress", true)
	if err != nil {
		return err
	}

	return fn.(DeleteActionAssocTxnFn)(accountID, cryptoAddressID)
}

//===========================================================================
// Counterparty Interface Methods
//===========================================================================

func (tx *Tx) SearchCounterparties(query *models.SearchQuery) (*models.CounterpartyPage, error) {
	fn, err := tx.check("SearchCounterparties", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(query)
	if err != nil {
		return nil, err
	}

	return out.(*models.CounterpartyPage), err
}

func (tx *Tx) ListCounterparties(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error) {
	fn, err := tx.check("ListCounterparties", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(page)
	if err != nil {
		return nil, err
	}

	return out.(*models.CounterpartyPage), err
}

func (tx *Tx) ListCounterpartySourceInfo(source enum.Source) ([]*models.CounterpartySourceInfo, error) {
	fn, err := tx.check("ListCounterpartySourceInfo", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(source)
	if err != nil {
		return nil, err
	}

	return out.([]*models.CounterpartySourceInfo), nil
}

func (tx *Tx) CreateCounterparty(in *models.Counterparty) error {
	fn, err := tx.check("CreateCounterparty", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveCounterparty(counterpartyID ulid.ULID) (*models.Counterparty, error) {
	fn, err := tx.check("RetrieveCounterparty", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnULIDFn)(counterpartyID)
	if err != nil {
		return nil, err
	}

	return out.(*models.Counterparty), nil
}

func (tx *Tx) LookupCounterparty(field, value string) (*models.Counterparty, error) {
	fn, err := tx.check("LookupCounterparty", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(LookupAssocTxnFn)(field, value)
	if err != nil {
		return nil, err
	}

	return out.(*models.Counterparty), nil
}

func (tx *Tx) UpdateCounterparty(in *models.Counterparty) error {
	fn, err := tx.check("UpdateCounterparty", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) DeleteCounterparty(counterpartyID ulid.ULID) error {
	fn, err := tx.check("DeleteCounterparty", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnULIDFn)(counterpartyID)
}

//===========================================================================
// Contact Interface Methods
//===========================================================================

func (tx *Tx) ListContacts(counterparty any, page *models.PageInfo) (*models.ContactsPage, error) {
	fn, err := tx.check("ListContacts", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnFn)(counterparty, page)
	if err != nil {
		return nil, err
	}

	return out.(*models.ContactsPage), nil
}

func (tx *Tx) CreateContact(in *models.Contact) error {
	fn, err := tx.check("CreateContact", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveContact(contactID, counterpartyID any) (*models.Contact, error) {
	fn, err := tx.check("RetrieveContact", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListRetrieveAssocTxnFn)(contactID, counterpartyID)
	if err != nil {
		return nil, err
	}

	return out.(*models.Contact), nil
}

func (tx *Tx) UpdateContact(in *models.Contact) error {
	fn, err := tx.check("UpdateContact", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) DeleteContact(contactID, counterpartyID any) error {
	fn, err := tx.check("DeleteContact", true)
	if err != nil {
		return err
	}

	return fn.(DeleteActionAssocTxnFn)(contactID, counterpartyID)
}

//===========================================================================
// Sunrise Interface Methods
//===========================================================================

func (tx *Tx) ListSunrise(in *models.PageInfo) (*models.SunrisePage, error) {
	fn, err := tx.check("ListSunrise", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(in)
	if err != nil {
		return nil, err
	}

	return out.(*models.SunrisePage), nil
}

func (tx *Tx) CreateSunrise(in *models.Sunrise) error {
	fn, err := tx.check("CreateSunrise", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveSunrise(id ulid.ULID) (*models.Sunrise, error) {
	fn, err := tx.check("RetrieveSunrise", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnULIDFn)(id)
	if err != nil {
		return nil, err
	}

	return out.(*models.Sunrise), nil
}

func (tx *Tx) UpdateSunrise(in *models.Sunrise) error {
	fn, err := tx.check("UpdateSunrise", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) UpdateSunriseStatus(id uuid.UUID, status enum.Status) error {
	fn, err := tx.check("UpdateSunriseStatus", true)
	if err != nil {
		return err
	}

	return fn.(DeleteActionAssocTxnFn)(id, status)
}

func (tx *Tx) DeleteSunrise(in ulid.ULID) error {
	fn, err := tx.check("DeleteSunrise", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnULIDFn)(in)
}

func (tx *Tx) GetOrCreateSunriseCounterparty(email, name string) (*models.Counterparty, error) {
	fn, err := tx.check("GetOrCreateSunriseCounterparty", true)
	if err != nil {
		return nil, err
	}

	out, err := fn.(LookupAssocTxnFn)(email, name)
	if err != nil {
		return nil, err
	}

	return out.(*models.Counterparty), nil
}

//===========================================================================
// User Interface Methods
//===========================================================================

func (tx *Tx) ListUsers(page *models.UserPageInfo) (*models.UserPage, error) {
	fn, err := tx.check("ListUsers", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(page)
	if err != nil {
		return nil, err
	}

	return out.(*models.UserPage), nil
}

func (tx *Tx) CreateUser(in *models.User) error {
	fn, err := tx.check("CreateUser", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveUser(emailOrUserID any) (*models.User, error) {
	fn, err := tx.check("RetrieveUser", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnAnyFn)(emailOrUserID)
	if err != nil {
		return nil, err
	}

	return out.(*models.User), nil
}

func (tx *Tx) UpdateUser(in *models.User) error {
	fn, err := tx.check("UpdateUser", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) SetUserPassword(userID ulid.ULID, password string) error {
	fn, err := tx.check("SetUserPassword", true)
	if err != nil {
		return err
	}

	return fn.(DeleteActionAssocTxnFn)(userID, password)
}

func (tx *Tx) SetUserLastLogin(userID ulid.ULID, lastLogin time.Time) error {
	fn, err := tx.check("SetUserLastLogin", true)
	if err != nil {
		return err
	}

	return fn.(DeleteActionAssocTxnFn)(userID, lastLogin)
}

func (tx *Tx) DeleteUser(userID ulid.ULID) error {
	fn, err := tx.check("DeleteUser", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnULIDFn)(userID)
}

func (tx *Tx) LookupRole(role string) (*models.Role, error) {
	fn, err := tx.check("LookupRole", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnAnyFn)(role)
	if err != nil {
		return nil, err
	}

	return out.(*models.Role), nil
}

//===========================================================================
// APIKey Interface Methods
//===========================================================================

func (tx *Tx) ListAPIKeys(page *models.PageInfo) (*models.APIKeyPage, error) {
	fn, err := tx.check("ListAPIKeys", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(page)
	if err != nil {
		return nil, err
	}

	return out.(*models.APIKeyPage), nil
}

func (tx *Tx) CreateAPIKey(in *models.APIKey) error {
	fn, err := tx.check("CreateAPIKey", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveAPIKey(clientIDOrKeyID any) (*models.APIKey, error) {
	fn, err := tx.check("RetrieveAPIKey", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnAnyFn)(clientIDOrKeyID)
	if err != nil {
		return nil, err
	}

	return out.(*models.APIKey), nil
}

func (tx *Tx) UpdateAPIKey(in *models.APIKey) error {
	fn, err := tx.check("UpdateAPIKey", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) DeleteAPIKey(keyID ulid.ULID) error {
	fn, err := tx.check("DeleteAPIKey", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnULIDFn)(keyID)
}

//===========================================================================
// ResetPasswordLink Interface Methods
//===========================================================================

func (tx *Tx) ListResetPasswordLinks(page *models.PageInfo) (*models.ResetPasswordLinkPage, error) {
	fn, err := tx.check("ListResetPasswordLinks", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(ListTxnFn)(page)
	if err != nil {
		return nil, err
	}

	return out.(*models.ResetPasswordLinkPage), nil
}

func (tx *Tx) CreateResetPasswordLink(in *models.ResetPasswordLink) error {
	fn, err := tx.check("CreateResetPasswordLink", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) RetrieveResetPasswordLink(id ulid.ULID) (*models.ResetPasswordLink, error) {
	fn, err := tx.check("RetrieveResetPasswordLink", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(RetrieveTxnULIDFn)(id)
	if err != nil {
		return nil, err
	}

	return out.(*models.ResetPasswordLink), nil
}

func (tx *Tx) UpdateResetPasswordLink(in *models.ResetPasswordLink) error {
	fn, err := tx.check("UpdateResetPasswordLink", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(in)
}

func (tx *Tx) DeleteResetPasswordLink(id ulid.ULID) error {
	fn, err := tx.check("DeleteResetPasswordLink", true)
	if err != nil {
		return err
	}

	return fn.(DeleteTxnULIDFn)(id)
}

//===========================================================================
// Daybreak Interface Methods
//===========================================================================

func (tx *Tx) ListDaybreak() (map[string]*models.CounterpartySourceInfo, error) {
	fn, err := tx.check("ListDaybreak", false)
	if err != nil {
		return nil, err
	}

	out, err := fn.(EmptyAnyFn)()
	if err != nil {
		return nil, err
	}

	return out.(map[string]*models.CounterpartySourceInfo), nil
}

func (tx *Tx) CreateDaybreak(counterparty *models.Counterparty) error {
	fn, err := tx.check("CreateDaybreak", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(counterparty)
}

func (tx *Tx) UpdateDaybreak(counterparty *models.Counterparty) error {
	fn, err := tx.check("UpdateDaybreak", true)
	if err != nil {
		return err
	}

	return fn.(CreateUpdateTxnFn)(counterparty)
}

func (tx *Tx) DeleteDaybreak(counterpartyID ulid.ULID, ignoreTxns bool) error {
	fn, err := tx.check("DeleteDaybreak", true)
	if err != nil {
		return err
	}

	return fn.(DeleteActionAssocTxnFn)(counterpartyID, ignoreTxns)
}
