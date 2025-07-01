package mock

import (
	"context"
	"database/sql"
	"reflect"
	"strings"
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
// mock store should be used per test. To set a callback for `Store.Function()`, set
// the `Store.OnFunction` stub to your desired callback function.
type Store struct {
	calls    map[string]int
	readonly bool

	OnClose                          func() error
	OnBegin                          func(context.Context, *sql.TxOptions) (txn.Txn, error)
	OnListTransactions               func(ctx context.Context, in *models.TransactionPageInfo) (*models.TransactionPage, error)
	OnCreateTransaction              func(ctx context.Context, in *models.Transaction) error
	OnRetrieveTransaction            func(ctx context.Context, id uuid.UUID) (*models.Transaction, error)
	OnUpdateTransaction              func(ctx context.Context, in *models.Transaction) error
	OnDeleteTransaction              func(ctx context.Context, id uuid.UUID) error
	OnArchiveTransaction             func(ctx context.Context, id uuid.UUID) error
	OnUnarchiveTransaction           func(ctx context.Context, id uuid.UUID) error
	OnCountTransactions              func(ctx context.Context) (*models.TransactionCounts, error)
	OnPrepareTransaction             func(ctx context.Context, id uuid.UUID) (models.PreparedTransaction, error)
	OnTransactionState               func(ctx context.Context, id uuid.UUID) (bool, enum.Status, error)
	OnListSecureEnvelopes            func(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error)
	OnCreateSecureEnvelope           func(ctx context.Context, in *models.SecureEnvelope) error
	OnRetrieveSecureEnvelope         func(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error)
	OnUpdateSecureEnvelope           func(ctx context.Context, in *models.SecureEnvelope) error
	OnDeleteSecureEnvelope           func(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error
	OnLatestSecureEnvelope           func(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
	OnLatestPayloadEnvelope          func(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
	OnListAccounts                   func(ctx context.Context, in *models.PageInfo) (*models.AccountsPage, error)
	OnCreateAccount                  func(ctx context.Context, in *models.Account) error
	OnLookupAccount                  func(ctx context.Context, cryptoAddress string) (*models.Account, error)
	OnRetrieveAccount                func(ctx context.Context, id ulid.ULID) (*models.Account, error)
	OnUpdateAccount                  func(ctx context.Context, in *models.Account) error
	OnDeleteAccount                  func(ctx context.Context, id ulid.ULID) error
	OnListAccountTransactions        func(ctx context.Context, accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error)
	OnListCryptoAddresses            func(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error)
	OnCreateCryptoAddress            func(ctx context.Context, in *models.CryptoAddress) error
	OnRetrieveCryptoAddress          func(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error)
	OnUpdateCryptoAddress            func(ctx context.Context, in *models.CryptoAddress) error
	OnDeleteCryptoAddress            func(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error
	OnSearchCounterparties           func(ctx context.Context, query *models.SearchQuery) (*models.CounterpartyPage, error)
	OnListCounterparties             func(ctx context.Context, page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error)
	OnListCounterpartySourceInfo     func(ctx context.Context, source enum.Source) ([]*models.CounterpartySourceInfo, error)
	OnCreateCounterparty             func(ctx context.Context, in *models.Counterparty) error
	OnRetrieveCounterparty           func(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error)
	OnLookupCounterparty             func(ctx context.Context, field, value string) (*models.Counterparty, error)
	OnUpdateCounterparty             func(ctx context.Context, in *models.Counterparty) error
	OnDeleteCounterparty             func(ctx context.Context, counterpartyID ulid.ULID) error
	OnListContacts                   func(ctx context.Context, counterparty any, page *models.PageInfo) (*models.ContactsPage, error)
	OnCreateContact                  func(ctx context.Context, in *models.Contact) error
	OnRetrieveContact                func(ctx context.Context, contactID, counterparty any) (*models.Contact, error)
	OnUpdateContact                  func(ctx context.Context, in *models.Contact) error
	OnDeleteContact                  func(ctx context.Context, contactID, counterparty any) error
	OnUseTravelAddressFactory        func(models.TravelAddressFactory)
	OnListSunrise                    func(ctx context.Context, page *models.PageInfo) (*models.SunrisePage, error)
	OnCreateSunrise                  func(ctx context.Context, msg *models.Sunrise) error
	OnRetrieveSunrise                func(ctx context.Context, id ulid.ULID) (*models.Sunrise, error)
	OnUpdateSunrise                  func(ctx context.Context, msg *models.Sunrise) error
	OnUpdateSunriseStatus            func(ctx context.Context, txID uuid.UUID, status enum.Status) error
	OnDeleteSunrise                  func(ctx context.Context, id ulid.ULID) error
	OnGetOrCreateSunriseCounterparty func(ctx context.Context, email, name string) (*models.Counterparty, error)
	OnListUsers                      func(ctx context.Context, page *models.UserPageInfo) (*models.UserPage, error)
	OnCreateUser                     func(ctx context.Context, in *models.User) error
	OnRetrieveUser                   func(ctx context.Context, emailOrUserID any) (*models.User, error)
	OnUpdateUser                     func(ctx context.Context, in *models.User) error
	OnSetUserPassword                func(ctx context.Context, userID ulid.ULID, password string) (err error)
	OnSetUserLastLogin               func(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error)
	OnDeleteUser                     func(ctx context.Context, userID ulid.ULID) error
	OnLookupRole                     func(ctx context.Context, role string) (*models.Role, error)
	OnListAPIKeys                    func(ctx context.Context, in *models.PageInfo) (*models.APIKeyPage, error)
	OnCreateAPIKey                   func(ctx context.Context, in *models.APIKey) error
	OnRetrieveAPIKey                 func(ctx context.Context, clientIDOrKeyID any) (*models.APIKey, error)
	OnUpdateAPIKey                   func(ctx context.Context, in *models.APIKey) error
	OnDeleteAPIKey                   func(ctx context.Context, keyID ulid.ULID) error
	OnListResetPasswordLinks         func(ctx context.Context, page *models.PageInfo) (*models.ResetPasswordLinkPage, error)
	OnCreateResetPasswordLink        func(ctx context.Context, link *models.ResetPasswordLink) error
	OnRetrieveResetPasswordLink      func(ctx context.Context, linkID ulid.ULID) (*models.ResetPasswordLink, error)
	OnUpdateResetPasswordLink        func(ctx context.Context, link *models.ResetPasswordLink) error
	OnDeleteResetPasswordLink        func(ctx context.Context, linkID ulid.ULID) (err error)
	OnListComplianceAuditLogs        func(ctx context.Context, page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error)
	OnCreateComplianceAuditLog       func(ctx context.Context, log *models.ComplianceAuditLog) error
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
		calls:    make(map[string]int),
		readonly: uri.ReadOnly,
	}, nil
}

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Reset all the calls and callbacks in the store.
func (s *Store) Reset() {
	// reset the call counts
	s.calls = nil
	s.calls = make(map[string]int)

	// reset the callbacks using reflection
	v := reflect.ValueOf(s)
	v = v.Elem() // s is a pointer
	t := v.Type()
	for _, f := range reflect.VisibleFields(t) {
		// only reset functions named `OnSomething`
		if strings.HasPrefix(f.Name, "On") && f.Type.Kind() == reflect.Func {
			fv := v.FieldByIndex(f.Index)
			fv.SetZero()
		}
	}
}

// Assert that the expected number of calls were made to the given method.
func (s *Store) AssertCalls(t testing.TB, method string, expected int) {
	require.Equal(t, expected, s.calls[method], "expected %d calls to %s, got %d", expected, method, s.calls[method])
}

//===========================================================================
// Store Interface Methods
//===========================================================================

// If present, calls the callback previously set for "Close()" (set the callback
// with "OnClose()"), otherwise returns `nil` indicating success.
func (s *Store) Close() error {
	s.calls["Close"]++

	// perform callback if there is one
	if s.OnClose != nil {
		return s.OnClose()
	}

	// return no error
	return nil
}

// If present, calls the callback previously set for "Begin()" (set the callback
// with "OnBegin()"), otherwise returns a `Txn` mock.
func (s *Store) Begin(ctx context.Context, opts *sql.TxOptions) (txn.Txn, error) {
	s.calls["Begin"]++

	// perform callback if there is one
	if s.OnBegin != nil {
		return s.OnBegin(ctx, opts)
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

// Calls the callback previously set with `s.OnListTransactions = ...`
func (s *Store) ListTransactions(ctx context.Context, in *models.TransactionPageInfo) (*models.TransactionPage, error) {
	s.calls["ListTransactions"]++
	if s.OnListTransactions != nil {
		return s.OnListTransactions(ctx, in)
	}
	panic("ListTransactions callback not set")
}

// Calls the callback previously set with `s.OnCreateTransaction = ...`
func (s *Store) CreateTransaction(ctx context.Context, in *models.Transaction) error {
	s.calls["CreateTransaction"]++
	if s.OnCreateTransaction != nil {
		return s.OnCreateTransaction(ctx, in)
	}
	panic("CreateTransaction callback not set")
}

// Calls the callback previously set with `s.OnRetrieveTransaction = ...`
func (s *Store) RetrieveTransaction(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	s.calls["RetrieveTransaction"]++
	if s.OnRetrieveTransaction != nil {
		return s.OnRetrieveTransaction(ctx, id)
	}
	panic("RetrieveTransaction callback not set")
}

// Calls the callback previously set with `s.OnUpdateTransaction = ...`
func (s *Store) UpdateTransaction(ctx context.Context, in *models.Transaction) error {
	s.calls["UpdateTransaction"]++
	if s.OnUpdateTransaction != nil {
		return s.OnUpdateTransaction(ctx, in)
	}
	panic("UpdateTransaction callback not set")
}

// Calls the callback previously set with `s.OnDeleteTransaction = ...`
func (s *Store) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["DeleteTransaction"]++
	if s.OnDeleteTransaction != nil {
		return s.OnDeleteTransaction(ctx, id)
	}
	panic("DeleteTransaction callback not set")
}

// Calls the callback previously set with `s.OnArchiveTransaction = ...`
func (s *Store) ArchiveTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["ArchiveTransaction"]++
	if s.OnArchiveTransaction != nil {
		return s.OnArchiveTransaction(ctx, id)
	}
	panic("ArchiveTransaction callback not set")
}

// Calls the callback previously set with `s.OnUnarchiveTransaction = ...`
func (s *Store) UnarchiveTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["UnarchiveTransaction"]++
	if s.OnUnarchiveTransaction != nil {
		return s.OnUnarchiveTransaction(ctx, id)
	}
	panic("UnarchiveTransaction callback not set")
}

// Calls the callback previously set with `s.OnCountTransactions = ...`
func (s *Store) CountTransactions(ctx context.Context) (*models.TransactionCounts, error) {
	s.calls["CountTransactions"]++
	if s.OnCountTransactions != nil {
		return s.OnCountTransactions(ctx)
	}
	panic("CountTransactions callback not set")
}

// Calls the callback previously set with `s.OnPrepareTransaction = ...`
func (s *Store) PrepareTransaction(ctx context.Context, id uuid.UUID) (models.PreparedTransaction, error) {
	s.calls["PrepareTransaction"]++
	if s.OnPrepareTransaction != nil {
		return s.OnPrepareTransaction(ctx, id)
	}
	// if no callback is set, return a mock PreparedTransaction
	return &PreparedTransaction{
		callbacks: make(map[string]any),
		calls:     make(map[string]int),
	}, nil
}

// Calls the callback previously set with `s.OnTransactionState = ...`
func (s *Store) TransactionState(ctx context.Context, id uuid.UUID) (bool, enum.Status, error) {
	s.calls["TransactionState"]++
	if s.OnTransactionState != nil {
		return s.OnTransactionState(ctx, id)
	}
	panic("TransactionState callback not set")
}

//===========================================================================
// SecureEnvelope Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListSecureEnvelopes = ...`
func (s *Store) ListSecureEnvelopes(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	s.calls["ListSecureEnvelopes"]++
	if s.OnListSecureEnvelopes != nil {
		return s.OnListSecureEnvelopes(ctx, txID, page)
	}
	panic("ListSecureEnvelopes callback not set")
}

// Calls the callback previously set with `s.OnCreateSecureEnvelope = ...`
func (s *Store) CreateSecureEnvelope(ctx context.Context, in *models.SecureEnvelope) error {
	s.calls["CreateSecureEnvelope"]++
	if s.OnCreateSecureEnvelope != nil {
		return s.OnCreateSecureEnvelope(ctx, in)
	}
	panic("CreateSecureEnvelope callback not set")
}

// Calls the callback previously set with `s.OnRetrieveSecureEnvelope = ...`
func (s *Store) RetrieveSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	s.calls["RetrieveSecureEnvelope"]++
	if s.OnRetrieveSecureEnvelope != nil {
		return s.OnRetrieveSecureEnvelope(ctx, txID, envID)
	}
	panic("RetrieveSecureEnvelope callback not set")
}

// Calls the callback previously set with `s.OnUpdateSecureEnvelope = ...`
func (s *Store) UpdateSecureEnvelope(ctx context.Context, in *models.SecureEnvelope) error {
	s.calls["UpdateSecureEnvelope"]++
	if s.OnUpdateSecureEnvelope != nil {
		return s.OnUpdateSecureEnvelope(ctx, in)
	}
	panic("UpdateSecureEnvelope callback not set")
}

// Calls the callback previously set with `s.OnDeleteSecureEnvelope = ...`
func (s *Store) DeleteSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error {
	s.calls["DeleteSecureEnvelope"]++
	if s.OnDeleteSecureEnvelope != nil {
		return s.OnDeleteSecureEnvelope(ctx, txID, envID)
	}
	panic("DeleteSecureEnvelope callback not set")
}

// Calls the callback previously set with `s.OnLatestSecureEnvelope = ...`
func (s *Store) LatestSecureEnvelope(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	s.calls["LatestSecureEnvelope"]++
	if s.OnLatestSecureEnvelope != nil {
		return s.OnLatestSecureEnvelope(ctx, txID, direction)
	}
	panic("LatestSecureEnvelope callback not set")
}

// Calls the callback previously set with `s.OnLatestPayloadEnvelope = ...`
func (s *Store) LatestPayloadEnvelope(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	s.calls["LatestPayloadEnvelope"]++
	if s.OnLatestPayloadEnvelope != nil {
		return s.OnLatestPayloadEnvelope(ctx, txID, direction)
	}
	panic("LatestPayloadEnvelope callback not set")
}

//===========================================================================
// Account Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListAccounts = ...`
func (s *Store) ListAccounts(ctx context.Context, in *models.PageInfo) (*models.AccountsPage, error) {
	s.calls["ListAccounts"]++
	if s.OnListAccounts != nil {
		return s.OnListAccounts(ctx, in)
	}
	panic("ListAccounts callback not set")
}

// Calls the callback previously set with `s.OnCreateAccount = ...`
func (s *Store) CreateAccount(ctx context.Context, in *models.Account) error {
	s.calls["CreateAccount"]++
	if s.OnCreateAccount != nil {
		return s.OnCreateAccount(ctx, in)
	}
	panic("CreateAccount callback not set")
}

// Calls the callback previously set with `s.OnLookupAccount = ...`
func (s *Store) LookupAccount(ctx context.Context, cryptoAddress string) (*models.Account, error) {
	s.calls["LookupAccount"]++
	if s.OnLookupAccount != nil {
		return s.OnLookupAccount(ctx, cryptoAddress)
	}
	panic("LookupAccount callback not set")
}

// Calls the callback previously set with `s.OnRetrieveAccount = ...`
func (s *Store) RetrieveAccount(ctx context.Context, id ulid.ULID) (*models.Account, error) {
	s.calls["RetrieveAccount"]++
	if s.OnRetrieveAccount != nil {
		return s.OnRetrieveAccount(ctx, id)
	}
	panic("RetrieveAccount callback not set")
}

// Calls the callback previously set with `s.OnUpdateAccount = ...`
func (s *Store) UpdateAccount(ctx context.Context, in *models.Account) error {
	s.calls["UpdateAccount"]++
	if s.OnUpdateAccount != nil {
		return s.OnUpdateAccount(ctx, in)
	}
	panic("UpdateAccount callback not set")
}

// Calls the callback previously set with `s.OnDeleteAccount = ...`
func (s *Store) DeleteAccount(ctx context.Context, id ulid.ULID) error {
	s.calls["DeleteAccount"]++
	if s.OnDeleteAccount != nil {
		return s.OnDeleteAccount(ctx, id)
	}
	panic("DeleteAccount callback not set")
}

// Calls the callback previously set with `s.OnListAccountTransactions = ...`
func (s *Store) ListAccountTransactions(ctx context.Context, accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error) {
	s.calls["ListAccountTransactions"]++
	if s.OnListAccountTransactions != nil {
		return s.OnListAccountTransactions(ctx, accountID, page)
	}
	panic("ListAccountTransactions callback not set")
}

//===========================================================================
// CryptoAddress Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListCryptoAddresses = ...`
func (s *Store) ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	s.calls["ListCryptoAddresses"]++
	if s.OnListCryptoAddresses != nil {
		return s.OnListCryptoAddresses(ctx, accountID, page)
	}
	panic("ListCryptoAddresses callback not set")
}

// Calls the callback previously set with `s.OnCreateCryptoAddress = ...`
func (s *Store) CreateCryptoAddress(ctx context.Context, in *models.CryptoAddress) error {
	s.calls["CreateCryptoAddress"]++
	if s.OnCreateCryptoAddress != nil {
		return s.OnCreateCryptoAddress(ctx, in)
	}
	panic("CreateCryptoAddress callback not set")
}

// Calls the callback previously set with `s.OnRetrieveCryptoAddress = ...`
func (s *Store) RetrieveCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error) {
	s.calls["RetrieveCryptoAddress"]++
	if s.OnRetrieveCryptoAddress != nil {
		return s.OnRetrieveCryptoAddress(ctx, accountID, cryptoAddressID)
	}
	panic("RetrieveCryptoAddress callback not set")
}

// Calls the callback previously set with `s.OnUpdateCryptoAddress = ...`
func (s *Store) UpdateCryptoAddress(ctx context.Context, in *models.CryptoAddress) error {
	s.calls["UpdateCryptoAddress"]++
	if s.OnUpdateCryptoAddress != nil {
		return s.OnUpdateCryptoAddress(ctx, in)
	}
	panic("UpdateCryptoAddress callback not set")
}

// Calls the callback previously set with `s.OnDeleteCryptoAddress = ...`
func (s *Store) DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error {
	s.calls["DeleteCryptoAddress"]++
	if s.OnDeleteCryptoAddress != nil {
		return s.OnDeleteCryptoAddress(ctx, accountID, cryptoAddressID)
	}
	panic("DeleteCryptoAddress callback not set")
}

//===========================================================================
// Counterparty Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnSearchCounterparties = ...`
func (s *Store) SearchCounterparties(ctx context.Context, query *models.SearchQuery) (*models.CounterpartyPage, error) {
	s.calls["SearchCounterparties"]++
	if s.OnSearchCounterparties != nil {
		return s.OnSearchCounterparties(ctx, query)
	}
	panic("SearchCounterparties callback not set")
}

// Calls the callback previously set with `s.OnListCounterparties = ...`
func (s *Store) ListCounterparties(ctx context.Context, page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error) {
	s.calls["ListCounterparties"]++
	if s.OnListCounterparties != nil {
		return s.OnListCounterparties(ctx, page)
	}
	panic("ListCounterparties callback not set")
}

// Calls the callback previously set with `s.OnListCounterpartySourceInfo = ...`
func (s *Store) ListCounterpartySourceInfo(ctx context.Context, source enum.Source) ([]*models.CounterpartySourceInfo, error) {
	s.calls["ListCounterpartySourceInfo"]++
	if s.OnListCounterpartySourceInfo != nil {
		return s.OnListCounterpartySourceInfo(ctx, source)
	}
	panic("ListCounterpartySourceInfo callback not set")
}

// Calls the callback previously set with `s.OnCreateCounterparty = ...`
func (s *Store) CreateCounterparty(ctx context.Context, in *models.Counterparty) error {
	s.calls["CreateCounterparty"]++
	if s.OnCreateCounterparty != nil {
		return s.OnCreateCounterparty(ctx, in)
	}
	panic("CreateCounterparty callback not set")
}

// Calls the callback previously set with `s.OnRetrieveCounterparty = ...`
func (s *Store) RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error) {
	s.calls["RetrieveCounterparty"]++
	if s.OnRetrieveCounterparty != nil {
		return s.OnRetrieveCounterparty(ctx, counterpartyID)
	}
	panic("RetrieveCounterparty callback not set")
}

// Calls the callback previously set with `s.OnLookupCounterparty = ...`
func (s *Store) LookupCounterparty(ctx context.Context, field, value string) (*models.Counterparty, error) {
	s.calls["LookupCounterparty"]++
	if s.OnLookupCounterparty != nil {
		return s.OnLookupCounterparty(ctx, field, value)
	}
	panic("LookupCounterparty callback not set")
}

// Calls the callback previously set with `s.OnUpdateCounterparty = ...`
func (s *Store) UpdateCounterparty(ctx context.Context, in *models.Counterparty) error {
	s.calls["UpdateCounterparty"]++
	if s.OnUpdateCounterparty != nil {
		return s.OnUpdateCounterparty(ctx, in)
	}
	panic("UpdateCounterparty callback not set")
}

// Calls the callback previously set with `s.OnDeleteCounterparty = ...`
func (s *Store) DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error {
	s.calls["DeleteCounterparty"]++
	if s.OnDeleteCounterparty != nil {
		return s.OnDeleteCounterparty(ctx, counterpartyID)
	}
	panic("DeleteCounterparty callback not set")
}

//===========================================================================
// Contact Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListContacts = ...`
func (s *Store) ListContacts(ctx context.Context, counterparty any, page *models.PageInfo) (*models.ContactsPage, error) {
	s.calls["ListContacts"]++
	if s.OnListContacts != nil {
		return s.OnListContacts(ctx, counterparty, page)
	}
	panic("ListContacts callback not set")
}

// Calls the callback previously set with `s.OnCreateContact = ...`
func (s *Store) CreateContact(ctx context.Context, in *models.Contact) error {
	s.calls["CreateContact"]++
	if s.OnCreateContact != nil {
		return s.OnCreateContact(ctx, in)
	}
	panic("CreateContact callback not set")
}

// Calls the callback previously set with `s.OnRetrieveContact = ...`
func (s *Store) RetrieveContact(ctx context.Context, contactID, counterparty any) (*models.Contact, error) {
	s.calls["RetrieveContact"]++
	if s.OnRetrieveContact != nil {
		return s.OnRetrieveContact(ctx, counterparty, contactID)
	}
	panic("RetrieveContact callback not set")
}

// Calls the callback previously set with `s.OnUpdateContact = ...`
func (s *Store) UpdateContact(ctx context.Context, in *models.Contact) error {
	s.calls["UpdateContact"]++
	if s.OnUpdateContact != nil {
		return s.OnUpdateContact(ctx, in)
	}
	panic("UpdateContact callback not set")
}

// Calls the callback previously set with `s.OnDeleteContact = ...`
func (s *Store) DeleteContact(ctx context.Context, contactID, counterparty any) error {
	s.calls["DeleteContact"]++
	if s.OnDeleteContact != nil {
		return s.OnDeleteContact(ctx, contactID, counterparty)
	}
	panic("DeleteContact callback not set")
}

//===========================================================================
// Travel Address Factory
//===========================================================================

// Calls the callback previously set with `s.OnUseTravelAddressFactory = ...`
func (s *Store) UseTravelAddressFactory(f models.TravelAddressFactory) {
	s.calls["UseTravelAddressFactory"]++
	if s.OnUseTravelAddressFactory != nil {
		s.OnUseTravelAddressFactory(f)
	}
	panic("UseTravelAddressFactory callback not set")
}

//===========================================================================
// Sunrise Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListSunrise = ...`
func (s *Store) ListSunrise(ctx context.Context, page *models.PageInfo) (*models.SunrisePage, error) {
	s.calls["ListSunrise"]++
	if s.OnListSunrise != nil {
		return s.OnListSunrise(ctx, page)
	}
	panic("ListSunrise callback not set")
}

// Calls the callback previously set with `s.OnCreateSunrise = ...`
func (s *Store) CreateSunrise(ctx context.Context, msg *models.Sunrise) error {
	s.calls["CreateSunrise"]++
	if s.OnCreateSunrise != nil {
		return s.OnCreateSunrise(ctx, msg)
	}
	panic("CreateSunrise callback not set")
}

// Calls the callback previously set with `s.OnRetrieveSunrise = ...`
func (s *Store) RetrieveSunrise(ctx context.Context, id ulid.ULID) (*models.Sunrise, error) {
	s.calls["RetrieveSunrise"]++
	if s.OnRetrieveSunrise != nil {
		return s.OnRetrieveSunrise(ctx, id)
	}
	panic("RetrieveSunrise callback not set")
}

// Calls the callback previously set with `s.OnUpdateSunrise = ...`
func (s *Store) UpdateSunrise(ctx context.Context, msg *models.Sunrise) error {
	s.calls["UpdateSunrise"]++
	if s.OnUpdateSunrise != nil {
		return s.OnUpdateSunrise(ctx, msg)
	}
	panic("UpdateSunrise callback not set")
}

// Calls the callback previously set with `s.OnUpdateSunriseStatus = ...`
func (s *Store) UpdateSunriseStatus(ctx context.Context, txID uuid.UUID, status enum.Status) error {
	s.calls["UpdateSunriseStatus"]++
	if s.OnUpdateSunriseStatus != nil {
		return s.OnUpdateSunriseStatus(ctx, txID, status)
	}
	panic("UpdateSunriseStatus callback not set")
}

// Calls the callback previously set with `s.OnDeleteSunrise = ...`
func (s *Store) DeleteSunrise(ctx context.Context, id ulid.ULID) error {
	s.calls["DeleteSunrise"]++
	if s.OnDeleteSunrise != nil {
		return s.OnDeleteSunrise(ctx, id)
	}
	panic("DeleteSunrise callback not set")
}

// Calls the callback previously set with `s.OnGetOrCreateSunriseCounterparty = ...`
func (s *Store) GetOrCreateSunriseCounterparty(ctx context.Context, email, name string) (*models.Counterparty, error) {
	s.calls["GetOrCreateSunriseCounterparty"]++
	if s.OnGetOrCreateSunriseCounterparty != nil {
		return s.OnGetOrCreateSunriseCounterparty(ctx, email, name)
	}
	panic("GetOrCreateSunriseCounterparty callback not set")
}

//===========================================================================
// User Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListUsers = ...`
func (s *Store) ListUsers(ctx context.Context, page *models.UserPageInfo) (*models.UserPage, error) {
	s.calls["ListUsers"]++
	if s.OnListUsers != nil {
		return s.OnListUsers(ctx, page)
	}
	panic("ListUsers callback not set")
}

// Calls the callback previously set with `s.OnCreateUser = ...`
func (s *Store) CreateUser(ctx context.Context, in *models.User) error {
	s.calls["CreateUser"]++
	if s.OnCreateUser != nil {
		return s.OnCreateUser(ctx, in)
	}
	panic("CreateUser callback not set")
}

// Calls the callback previously set with `s.OnRetrieveUser = ...`
func (s *Store) RetrieveUser(ctx context.Context, emailOrUserID any) (*models.User, error) {
	s.calls["RetrieveUser"]++
	if s.OnRetrieveUser != nil {
		return s.OnRetrieveUser(ctx, emailOrUserID)
	}
	panic("RetrieveUser callback not set")
}

// Calls the callback previously set with `s.OnUpdateUser = ...`
func (s *Store) UpdateUser(ctx context.Context, in *models.User) error {
	s.calls["UpdateUser"]++
	if s.OnUpdateUser != nil {
		return s.OnUpdateUser(ctx, in)
	}
	panic("UpdateUser callback not set")
}

// Calls the callback previously set with `s.OnSetUserPassword = ...`
func (s *Store) SetUserPassword(ctx context.Context, userID ulid.ULID, password string) (err error) {
	s.calls["SetUserPassword"]++
	if s.OnSetUserPassword != nil {
		return s.OnSetUserPassword(ctx, userID, password)
	}
	panic("SetUserPassword callback not set")
}

// Calls the callback previously set with `s.OnSetUserLastLogin = ...`
func (s *Store) SetUserLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error) {
	s.calls["SetUserLastLogin"]++
	if s.OnSetUserLastLogin != nil {
		return s.OnSetUserLastLogin(ctx, userID, lastLogin)
	}
	panic("SetUserLastLogin callback not set")
}

// Calls the callback previously set with `s.OnDeleteUser = ...`
func (s *Store) DeleteUser(ctx context.Context, userID ulid.ULID) error {
	s.calls["DeleteUser"]++
	if s.OnDeleteUser != nil {
		return s.OnDeleteUser(ctx, userID)
	}
	panic("DeleteUser callback not set")
}

// Calls the callback previously set with `s.OnLookupRole = ...`
func (s *Store) LookupRole(ctx context.Context, role string) (*models.Role, error) {
	s.calls["LookupRole"]++
	if s.OnLookupRole != nil {
		return s.OnLookupRole(ctx, role)
	}
	panic("LookupRole callback not set")
}

//===========================================================================
// API Key Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListAPIKeys = ...`
func (s *Store) ListAPIKeys(ctx context.Context, in *models.PageInfo) (*models.APIKeyPage, error) {
	s.calls["ListAPIKeys"]++
	if s.OnListAPIKeys != nil {
		return s.OnListAPIKeys(ctx, in)
	}
	panic("ListAPIKeys callback not set")
}

// Calls the callback previously set with `s.OnCreateAPIKey = ...`
func (s *Store) CreateAPIKey(ctx context.Context, in *models.APIKey) error {
	s.calls["CreateAPIKey"]++
	if s.OnCreateAPIKey != nil {
		return s.OnCreateAPIKey(ctx, in)
	}
	panic("CreateAPIKey callback not set")
}

// Calls the callback previously set with `s.OnRetrieveAPIKey = ...`
func (s *Store) RetrieveAPIKey(ctx context.Context, clientIDOrKeyID any) (*models.APIKey, error) {
	s.calls["RetrieveAPIKey"]++
	if s.OnRetrieveAPIKey != nil {
		return s.OnRetrieveAPIKey(ctx, clientIDOrKeyID)
	}
	panic("RetrieveAPIKey callback not set")
}

// Calls the callback previously set with `s.OnUpdateAPIKey = ...`
func (s *Store) UpdateAPIKey(ctx context.Context, in *models.APIKey) error {
	s.calls["UpdateAPIKey"]++
	if s.OnUpdateAPIKey != nil {
		return s.OnUpdateAPIKey(ctx, in)
	}
	panic("UpdateAPIKey callback not set")
}

// Calls the callback previously set with `s.OnDeleteAPIKey = ...`
func (s *Store) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) error {
	s.calls["DeleteAPIKey"]++
	if s.OnDeleteAPIKey != nil {
		return s.OnDeleteAPIKey(ctx, keyID)
	}
	panic("DeleteAPIKey callback not set")
}

//===========================================================================
// Reset Password Link Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListResetPasswordLinks = ...`
func (s *Store) ListResetPasswordLinks(ctx context.Context, page *models.PageInfo) (*models.ResetPasswordLinkPage, error) {
	s.calls["ListResetPasswordLinks"]++
	if s.OnListResetPasswordLinks != nil {
		return s.OnListResetPasswordLinks(ctx, page)
	}
	panic("ListResetPasswordLinks callback not set")
}

// Calls the callback previously set with `s.OnCreateResetPasswordLink = ...`
func (s *Store) CreateResetPasswordLink(ctx context.Context, link *models.ResetPasswordLink) error {
	s.calls["CreateResetPasswordLink"]++
	if s.OnCreateResetPasswordLink != nil {
		return s.OnCreateResetPasswordLink(ctx, link)
	}
	panic("CreateResetPasswordLink callback not set")
}

// Calls the callback previously set with `s.OnRetrieveResetPasswordLink = ...`
func (s *Store) RetrieveResetPasswordLink(ctx context.Context, linkID ulid.ULID) (*models.ResetPasswordLink, error) {
	s.calls["RetrieveResetPasswordLink"]++
	if s.OnRetrieveResetPasswordLink != nil {
		return s.OnRetrieveResetPasswordLink(ctx, linkID)
	}
	panic("RetrieveResetPasswordLink callback not set")
}

// Calls the callback previously set with `s.OnUpdateResetPasswordLink = ...`
func (s *Store) UpdateResetPasswordLink(ctx context.Context, link *models.ResetPasswordLink) error {
	s.calls["UpdateResetPasswordLink"]++
	if s.OnUpdateResetPasswordLink != nil {
		return s.OnUpdateResetPasswordLink(ctx, link)
	}
	panic("UpdateResetPasswordLink callback not set")
}

// Calls the callback previously set with `s.OnDeleteResetPasswordLink = ...`
func (s *Store) DeleteResetPasswordLink(ctx context.Context, linkID ulid.ULID) (err error) {
	s.calls["DeleteResetPasswordLink"]++
	if s.OnDeleteResetPasswordLink != nil {
		return s.OnDeleteResetPasswordLink(ctx, linkID)
	}
	panic("DeleteResetPasswordLink callback not set")
}

//===========================================================================
// Compliance Audit Log Store Methods
//===========================================================================

// Calls the callback previously set with `s.OnListComplianceAuditLog = ...`
func (s *Store) ListComplianceAuditLogs(ctx context.Context, page *models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error) {
	s.calls["ListComplianceAuditLogs"]++
	if s.OnListComplianceAuditLogs != nil {
		return s.OnListComplianceAuditLogs(ctx, page)
	}
	panic("ListComplianceAuditLogs callback not set")
}

// Calls the callback previously set with `s.OnCreateComplianceAuditLog = ...`
func (s *Store) CreateComplianceAuditLog(ctx context.Context, log *models.ComplianceAuditLog) error {
	s.calls["CreateComplianceAuditLog"]++
	if s.OnCreateComplianceAuditLog != nil {
		return s.OnCreateComplianceAuditLog(ctx, log)
	}
	panic("CreateComplianceAuditLog callback not set")
}
