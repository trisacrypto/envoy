package mock

import (
	"context"
	"database/sql"
	"fmt"
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
// Store Callback Types
//===========================================================================

type ErrorFn func() error
type ContextFn func(context.Context, ...any) (any, error)
type ListStoreFn func(context.Context, any) (any, error)
type CreateUpdateStoreFn func(context.Context, any) error
type RetrieveStoreAnyFn func(context.Context, any) (any, error)
type RetrieveStoreUUIDFn func(context.Context, uuid.UUID) (any, error)
type RetrieveStoreULIDFn func(context.Context, ulid.ULID) (any, error)
type DeleteStoreUUIDFn func(context.Context, uuid.UUID) error
type DeleteStoreULIDFn func(context.Context, ulid.ULID) error
type TransactionStateFn func(context.Context, uuid.UUID) (bool, enum.Status, error)
type ListRetrieveAssocFn func(context.Context, any, any) (any, error)
type ListRetrieveAssocUUIDFn func(context.Context, uuid.UUID, any) (any, error)
type ListRetrieveAssocULIDFn func(context.Context, ulid.ULID, any) (any, error)
type DeleteActionAssocFn func(context.Context, any, any) error

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

// Set a callback for the given method. Make sure to use one of the callback types and
// not the function signature itself. If a method is called without a callback set,
// the test will panic. The test will also panic if return type assertions aren't met.
// NOTE: if a callback is already set, it will be overwritten but the calls will not be.
func (s *Store) On(method string, fn any) {
	s.callbacks[method] = fn
}

// Have a method return the specified error.
// NOTE: you may have to add a case to this method if you're implementing a test that
// uses that call back for the first time. Try to group test cases together if possible.
func (s *Store) Err(method string, err error) {
	switch method {
	case "Close":
		s.callbacks["Close"] = func() error { return err }
	default:
		panic(fmt.Errorf("method %q not implemented yet", method))
	}
}

//===========================================================================
// Store Interface Methods
//===========================================================================

func (s *Store) Close() error {
	s.calls["Close"]++
	if fn, ok := s.callbacks["Close"]; ok {
		return fn.(ErrorFn)()
	}
	return nil
}

func (s *Store) Begin(ctx context.Context, opts *sql.TxOptions) (txn.Txn, error) {
	s.calls["Begin"]++

	if opts == nil {
		opts = &sql.TxOptions{ReadOnly: s.readonly}
	} else if s.readonly && !opts.ReadOnly {
		return nil, errors.ErrReadOnly
	}

	return &Tx{
		opts:      opts,
		callbacks: make(map[string]any),
		calls:     make(map[string]int),
	}, nil
}

//===========================================================================
// Transaction Store Methods
//===========================================================================

func (s *Store) ListTransactions(ctx context.Context, in *models.TransactionPageInfo) (*models.TransactionPage, error) {
	s.calls["ListTransactions"]++
	if fn, ok := s.callbacks["ListTransactions"]; ok {
		out, err := fn.(ListStoreFn)(ctx, in)
		return out.(*models.TransactionPage), err
	}
	panic("ListTransactions callback not set")
}

func (s *Store) CreateTransaction(ctx context.Context, in *models.Transaction) error {
	s.calls["CreateTransaction"]++
	if fn, ok := s.callbacks["CreateTransaction"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("CreateTransaction callback not set")
}

func (s *Store) RetrieveTransaction(ctx context.Context, id uuid.UUID) (*models.Transaction, error) {
	s.calls["RetrieveTransaction"]++
	if fn, ok := s.callbacks["RetrieveTransaction"]; ok {
		out, err := fn.(RetrieveStoreUUIDFn)(ctx, id)
		return out.(*models.Transaction), err
	}
	panic("RetrieveTransaction callback not set")
}

func (s *Store) UpdateTransaction(ctx context.Context, in *models.Transaction) error {
	s.calls["UpdateTransaction"]++
	if fn, ok := s.callbacks["UpdateTransaction"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("UpdateTransaction callback not set")
}

func (s *Store) DeleteTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["DeleteTransaction"]++
	if fn, ok := s.callbacks["DeleteTransaction"]; ok {
		return fn.(DeleteStoreUUIDFn)(ctx, id)
	}
	panic("DeleteTransaction callback not set")
}

func (s *Store) ArchiveTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["ArchiveTransaction"]++
	if fn, ok := s.callbacks["ArchiveTransaction"]; ok {
		return fn.(DeleteStoreUUIDFn)(ctx, id)
	}
	panic("ArchiveTransaction callback not set")
}

func (s *Store) UnarchiveTransaction(ctx context.Context, id uuid.UUID) error {
	s.calls["UnarchiveTransaction"]++
	if fn, ok := s.callbacks["UnarchiveTransaction"]; ok {
		return fn.(DeleteStoreUUIDFn)(ctx, id)
	}
	panic("UnarchiveTransaction callback not set")
}

func (s *Store) CountTransactions(ctx context.Context) (*models.TransactionCounts, error) {
	s.calls["CountTransactions"]++
	if fn, ok := s.callbacks["CountTransactions"]; ok {
		out, err := fn.(ContextFn)(ctx)
		return out.(*models.TransactionCounts), err
	}
	panic("CountTransactions callback not set")
}

func (s *Store) PrepareTransaction(ctx context.Context, id uuid.UUID) (models.PreparedTransaction, error) {
	s.calls["PrepareTransaction"]++
	if fn, ok := s.callbacks["PrepareTransaction"]; ok {
		out, err := fn.(RetrieveStoreUUIDFn)(ctx, id)
		return out.(models.PreparedTransaction), err
	}
	panic("PrepareTransaction callback not set")
}

func (s *Store) TransactionState(ctx context.Context, id uuid.UUID) (bool, enum.Status, error) {
	s.calls["TransactionState"]++
	if fn, ok := s.callbacks["TransactionState"]; ok {
		return fn.(TransactionStateFn)(ctx, id)
	}
	panic("TransactionState callback not set")
}

//===========================================================================
// SecureEnvelope Store Methods
//===========================================================================

func (s *Store) ListSecureEnvelopes(ctx context.Context, txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	s.calls["ListSecureEnvelopes"]++
	if fn, ok := s.callbacks["ListSecureEnvelopes"]; ok {
		out, err := fn.(ListRetrieveAssocUUIDFn)(ctx, txID, page)
		return out.(*models.SecureEnvelopePage), err
	}
	panic("ListSecureEnvelopes callback not set")
}

func (s *Store) CreateSecureEnvelope(ctx context.Context, in *models.SecureEnvelope) error {
	s.calls["CreateSecureEnvelope"]++
	if fn, ok := s.callbacks["CreateSecureEnvelope"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("CreateSecureEnvelope callback not set")
}

func (s *Store) RetrieveSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	s.calls["RetrieveSecureEnvelope"]++
	if fn, ok := s.callbacks["RetrieveSecureEnvelope"]; ok {
		out, err := fn.(ListRetrieveAssocUUIDFn)(ctx, txID, envID)
		return out.(*models.SecureEnvelope), err
	}
	panic("RetrieveSecureEnvelope callback not set")
}

func (s *Store) UpdateSecureEnvelope(ctx context.Context, in *models.SecureEnvelope) error {
	s.calls["UpdateSecureEnvelope"]++
	if fn, ok := s.callbacks["UpdateSecureEnvelope"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("UpdateSecureEnvelope callback not set")
}

func (s *Store) DeleteSecureEnvelope(ctx context.Context, txID uuid.UUID, envID ulid.ULID) error {
	s.calls["DeleteSecureEnvelope"]++
	if fn, ok := s.callbacks["DeleteSecureEnvelope"]; ok {
		return fn.(DeleteStoreULIDFn)(ctx, envID)
	}
	panic("DeleteSecureEnvelope callback not set")
}

func (s *Store) LatestSecureEnvelope(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	s.calls["LatestSecureEnvelope"]++
	if fn, ok := s.callbacks["LatestSecureEnvelope"]; ok {
		out, err := fn.(ListRetrieveAssocUUIDFn)(ctx, txID, direction)
		return out.(*models.SecureEnvelope), err
	}
	panic("LatestSecureEnvelope callback not set")
}

func (s *Store) LatestPayloadEnvelope(ctx context.Context, txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	s.calls["LatestPayloadEnvelope"]++
	if fn, ok := s.callbacks["LatestPayloadEnvelope"]; ok {
		out, err := fn.(ListRetrieveAssocUUIDFn)(ctx, txID, direction)
		return out.(*models.SecureEnvelope), err
	}
	panic("LatestPayloadEnvelope callback not set")
}

//===========================================================================
// Account Store Methods
//===========================================================================

func (s *Store) ListAccounts(ctx context.Context, in *models.PageInfo) (*models.AccountsPage, error) {
	s.calls["ListAccounts"]++
	if fn, ok := s.callbacks["ListAccounts"]; ok {
		out, err := fn.(ListStoreFn)(ctx, in)
		return out.(*models.AccountsPage), err
	}
	panic("ListAccounts callback not set")
}

func (s *Store) CreateAccount(ctx context.Context, in *models.Account) error {
	s.calls["CreateAccount"]++
	if fn, ok := s.callbacks["CreateAccount"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("CreateAccount callback not set")
}

func (s *Store) LookupAccount(ctx context.Context, cryptoAddress string) (*models.Account, error) {
	s.calls["LookupAccount"]++
	if fn, ok := s.callbacks["LookupAccount"]; ok {
		out, err := fn.(ContextFn)(ctx, cryptoAddress)
		return out.(*models.Account), err
	}
	panic("LookupAccount callback not set")
}

func (s *Store) RetrieveAccount(ctx context.Context, id ulid.ULID) (*models.Account, error) {
	s.calls["RetrieveAccount"]++
	if fn, ok := s.callbacks["RetrieveAccount"]; ok {
		out, err := fn.(RetrieveStoreULIDFn)(ctx, id)
		return out.(*models.Account), err
	}
	panic("RetrieveAccount callback not set")
}

func (s *Store) UpdateAccount(ctx context.Context, in *models.Account) error {
	s.calls["UpdateAccount"]++
	if fn, ok := s.callbacks["UpdateAccount"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("UpdateAccount callback not set")
}

func (s *Store) DeleteAccount(ctx context.Context, id ulid.ULID) error {
	s.calls["DeleteAccount"]++
	if fn, ok := s.callbacks["DeleteAccount"]; ok {
		return fn.(DeleteStoreULIDFn)(ctx, id)
	}
	panic("DeleteAccount callback not set")
}

func (s *Store) ListAccountTransactions(ctx context.Context, accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error) {
	s.calls["ListAccountTransactions"]++
	if fn, ok := s.callbacks["ListAccountTransactions"]; ok {
		out, err := fn.(ListRetrieveAssocULIDFn)(ctx, accountID, page)
		return out.(*models.TransactionPage), err
	}
	panic("ListAccountTransactions callback not set")
}

//===========================================================================
// CryptoAddress Store Methods
//===========================================================================

func (s *Store) ListCryptoAddresses(ctx context.Context, accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	s.calls["ListCryptoAddresses"]++
	if fn, ok := s.callbacks["ListCryptoAddresses"]; ok {
		out, err := fn.(ListRetrieveAssocULIDFn)(ctx, accountID, page)
		return out.(*models.CryptoAddressPage), err
	}
	panic("ListCryptoAddresses callback not set")
}

func (s *Store) CreateCryptoAddress(ctx context.Context, in *models.CryptoAddress) error {
	s.calls["CreateCryptoAddress"]++
	if fn, ok := s.callbacks["CreateCryptoAddress"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("CreateCryptoAddress callback not set")
}

func (s *Store) RetrieveCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error) {
	s.calls["RetrieveCryptoAddress"]++
	if fn, ok := s.callbacks["RetrieveCryptoAddress"]; ok {
		out, err := fn.(ListRetrieveAssocULIDFn)(ctx, accountID, cryptoAddressID)
		return out.(*models.CryptoAddress), err
	}
	panic("RetrieveCryptoAddress callback not set")
}

func (s *Store) UpdateCryptoAddress(ctx context.Context, in *models.CryptoAddress) error {
	s.calls["UpdateCryptoAddress"]++
	if fn, ok := s.callbacks["UpdateCryptoAddress"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("UpdateCryptoAddress callback not set")
}

func (s *Store) DeleteCryptoAddress(ctx context.Context, accountID, cryptoAddressID ulid.ULID) error {
	s.calls["DeleteCryptoAddress"]++
	if fn, ok := s.callbacks["DeleteCryptoAddress"]; ok {
		return fn.(DeleteStoreULIDFn)(ctx, cryptoAddressID)
	}
	panic("DeleteCryptoAddress callback not set")
}

//===========================================================================
// Counterparty Store Methods
//===========================================================================

func (s *Store) SearchCounterparties(ctx context.Context, query *models.SearchQuery) (*models.CounterpartyPage, error) {
	s.calls["SearchCounterparties"]++
	if fn, ok := s.callbacks["SearchCounterparties"]; ok {
		out, err := fn.(ListStoreFn)(ctx, query)
		return out.(*models.CounterpartyPage), err
	}
	panic("SearchCounterparties callback not set")
}

func (s *Store) ListCounterparties(ctx context.Context, page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error) {
	s.calls["ListCounterparties"]++
	if fn, ok := s.callbacks["ListCounterparties"]; ok {
		out, err := fn.(ListStoreFn)(ctx, page)
		return out.(*models.CounterpartyPage), err
	}
	panic("ListCounterparties callback not set")
}

func (s *Store) ListCounterpartySourceInfo(ctx context.Context, source enum.Source) ([]*models.CounterpartySourceInfo, error) {
	s.calls["ListCounterpartySourceInfo"]++
	if fn, ok := s.callbacks["ListCounterpartySourceInfo"]; ok {
		out, err := fn.(ListStoreFn)(ctx, source)
		if err != nil {
			return nil, err
		}
		return out.([]*models.CounterpartySourceInfo), nil
	}
	panic("ListCounterpartySourceInfo callback not set")
}

func (s *Store) CreateCounterparty(ctx context.Context, in *models.Counterparty) error {
	s.calls["CreateCounterparty"]++
	if fn, ok := s.callbacks["CreateCounterparty"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("CreateCounterparty callback not set")
}

func (s *Store) RetrieveCounterparty(ctx context.Context, counterpartyID ulid.ULID) (*models.Counterparty, error) {
	s.calls["RetrieveCounterparty"]++
	if fn, ok := s.callbacks["RetrieveCounterparty"]; ok {
		out, err := fn.(RetrieveStoreULIDFn)(ctx, counterpartyID)
		return out.(*models.Counterparty), err
	}
	panic("RetrieveCounterparty callback not set")
}

func (s *Store) LookupCounterparty(ctx context.Context, field, value string) (*models.Counterparty, error) {
	s.calls["LookupCounterparty"]++
	if fn, ok := s.callbacks["LookupCounterparty"]; ok {
		out, err := fn.(ContextFn)(ctx, field, value)
		return out.(*models.Counterparty), err
	}
	panic("LookupCounterparty callback not set")
}

func (s *Store) UpdateCounterparty(ctx context.Context, in *models.Counterparty) error {
	s.calls["UpdateCounterparty"]++
	if fn, ok := s.callbacks["UpdateCounterparty"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("UpdateCounterparty callback not set")
}

func (s *Store) DeleteCounterparty(ctx context.Context, counterpartyID ulid.ULID) error {
	s.calls["DeleteCounterparty"]++
	if fn, ok := s.callbacks["DeleteCounterparty"]; ok {
		return fn.(DeleteStoreULIDFn)(ctx, counterpartyID)
	}
	panic("DeleteCounterparty callback not set")
}

//===========================================================================
// Contact Store Methods
//===========================================================================

func (s *Store) ListContacts(ctx context.Context, counterparty any, page *models.PageInfo) (*models.ContactsPage, error) {
	s.calls["ListContacts"]++
	if fn, ok := s.callbacks["ListContacts"]; ok {
		out, err := fn.(ListRetrieveAssocFn)(ctx, counterparty, page)
		return out.(*models.ContactsPage), err
	}
	panic("ListContacts callback not set")
}

func (s *Store) CreateContact(ctx context.Context, in *models.Contact) error {
	s.calls["CreateContact"]++
	if fn, ok := s.callbacks["CreateContact"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("CreateContact callback not set")
}

func (s *Store) RetrieveContact(ctx context.Context, contactID, counterparty any) (*models.Contact, error) {
	s.calls["RetrieveContact"]++
	if fn, ok := s.callbacks["RetrieveContact"]; ok {
		out, err := fn.(ListRetrieveAssocFn)(ctx, counterparty, contactID)
		return out.(*models.Contact), err
	}
	panic("RetrieveContact callback not set")
}

func (s *Store) UpdateContact(ctx context.Context, in *models.Contact) error {
	s.calls["UpdateContact"]++
	if fn, ok := s.callbacks["UpdateContact"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("UpdateContact callback not set")
}

func (s *Store) DeleteContact(ctx context.Context, contactID, counterparty any) error {
	s.calls["DeleteContact"]++
	if fn, ok := s.callbacks["DeleteContact"]; ok {
		return fn.(DeleteActionAssocFn)(ctx, contactID, counterparty)
	}
	panic("DeleteContact callback not set")
}

//===========================================================================
// Travel Address Factory
//===========================================================================

func (s *Store) UseTravelAddressFactory(models.TravelAddressFactory) {
	s.calls["UseTravelAddressFactory"]++
	if fn, ok := s.callbacks["UseTravelAddressFactory"]; ok {
		fn.(ErrorFn)()
	}
	panic("UseTravelAddressFactory callback not set")
}

//===========================================================================
// Sunrise Store Methods
//===========================================================================

func (s *Store) ListSunrise(ctx context.Context, page *models.PageInfo) (*models.SunrisePage, error) {
	s.calls["ListSunrise"]++
	if fn, ok := s.callbacks["ListSunrise"]; ok {
		out, err := fn.(ListStoreFn)(ctx, page)
		return out.(*models.SunrisePage), err
	}
	panic("ListSunrise callback not set")
}

func (s *Store) CreateSunrise(ctx context.Context, msg *models.Sunrise) error {
	s.calls["CreateSunrise"]++
	if fn, ok := s.callbacks["CreateSunrise"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, msg)
	}
	panic("CreateSunrise callback not set")
}

func (s *Store) RetrieveSunrise(ctx context.Context, id ulid.ULID) (*models.Sunrise, error) {
	s.calls["RetrieveSunrise"]++
	if fn, ok := s.callbacks["RetrieveSunrise"]; ok {
		out, err := fn.(RetrieveStoreULIDFn)(ctx, id)
		return out.(*models.Sunrise), err
	}
	panic("RetrieveSunrise callback not set")
}

func (s *Store) UpdateSunrise(ctx context.Context, msg *models.Sunrise) error {
	s.calls["UpdateSunrise"]++
	if fn, ok := s.callbacks["UpdateSunrise"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, msg)
	}
	panic("UpdateSunrise callback not set")
}

func (s *Store) UpdateSunriseStatus(ctx context.Context, txID uuid.UUID, status enum.Status) error {
	s.calls["UpdateSunriseStatus"]++
	if fn, ok := s.callbacks["UpdateSunriseStatus"]; ok {
		return fn.(DeleteActionAssocFn)(ctx, txID, status)
	}
	panic("UpdateSunriseStatus callback not set")
}

func (s *Store) DeleteSunrise(ctx context.Context, id ulid.ULID) error {
	s.calls["DeleteSunrise"]++
	if fn, ok := s.callbacks["DeleteSunrise"]; ok {
		return fn.(DeleteStoreULIDFn)(ctx, id)
	}
	panic("DeleteSunrise callback not set")
}

func (s *Store) GetOrCreateSunriseCounterparty(ctx context.Context, email, name string) (*models.Counterparty, error) {
	s.calls["GetOrCreateSunriseCounterparty"]++
	if fn, ok := s.callbacks["GetOrCreateSunriseCounterparty"]; ok {
		out, err := fn.(ContextFn)(ctx, email, name)
		return out.(*models.Counterparty), err
	}
	panic("GetOrCreateSunriseCounterparty callback not set")
}

//===========================================================================
// User Store Methods
//===========================================================================

func (s *Store) ListUsers(ctx context.Context, page *models.UserPageInfo) (*models.UserPage, error) {
	s.calls["ListUsers"]++
	if fn, ok := s.callbacks["ListUsers"]; ok {
		out, err := fn.(ListStoreFn)(ctx, page)
		return out.(*models.UserPage), err
	}
	panic("ListUsers callback not set")
}

func (s *Store) CreateUser(ctx context.Context, in *models.User) error {
	s.calls["CreateUser"]++
	if fn, ok := s.callbacks["CreateUser"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("CreateUser callback not set")
}

func (s *Store) RetrieveUser(ctx context.Context, emailOrUserID any) (*models.User, error) {
	s.calls["RetrieveUser"]++
	if fn, ok := s.callbacks["RetrieveUser"]; ok {
		out, err := fn.(ContextFn)(ctx, emailOrUserID)
		return out.(*models.User), err
	}
	panic("RetrieveUser callback not set")
}

func (s *Store) UpdateUser(ctx context.Context, in *models.User) error {
	s.calls["UpdateUser"]++
	if fn, ok := s.callbacks["UpdateUser"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("UpdateUser callback not set")
}

func (s *Store) SetUserPassword(ctx context.Context, userID ulid.ULID, password string) (err error) {
	s.calls["SetUserPassword"]++
	if fn, ok := s.callbacks["SetUserPassword"]; ok {
		return fn.(DeleteActionAssocFn)(ctx, userID, password)
	}
	panic("SetUserPassword callback not set")
}

func (s *Store) SetUserLastLogin(ctx context.Context, userID ulid.ULID, lastLogin time.Time) (err error) {
	s.calls["SetUserLastLogin"]++
	if fn, ok := s.callbacks["SetUserLastLogin"]; ok {
		return fn.(DeleteActionAssocFn)(ctx, userID, lastLogin)
	}
	panic("SetUserLastLogin callback not set")
}

func (s *Store) DeleteUser(ctx context.Context, userID ulid.ULID) error {
	s.calls["DeleteUser"]++
	if fn, ok := s.callbacks["DeleteUser"]; ok {
		return fn.(DeleteStoreULIDFn)(ctx, userID)
	}
	panic("DeleteUser callback not set")
}

func (s *Store) LookupRole(ctx context.Context, role string) (*models.Role, error) {
	s.calls["LookupRole"]++
	if fn, ok := s.callbacks["LookupRole"]; ok {
		out, err := fn.(RetrieveStoreAnyFn)(ctx, role)
		return out.(*models.Role), err
	}
	panic("LookupRole callback not set")
}

//===========================================================================
// API Key Store Methods
//===========================================================================

func (s *Store) ListAPIKeys(ctx context.Context, in *models.PageInfo) (*models.APIKeyPage, error) {
	s.calls["ListAPIKeys"]++
	if fn, ok := s.callbacks["ListAPIKeys"]; ok {
		out, err := fn.(ListStoreFn)(ctx, in)
		return out.(*models.APIKeyPage), err
	}
	panic("ListAPIKeys callback not set")
}

func (s *Store) CreateAPIKey(ctx context.Context, in *models.APIKey) error {
	s.calls["CreateAPIKey"]++
	if fn, ok := s.callbacks["CreateAPIKey"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("CreateAPIKey callback not set")
}

func (s *Store) RetrieveAPIKey(ctx context.Context, clientIDOrKeyID any) (*models.APIKey, error) {
	s.calls["RetrieveAPIKey"]++
	if fn, ok := s.callbacks["RetrieveAPIKey"]; ok {
		out, err := fn.(RetrieveStoreAnyFn)(ctx, clientIDOrKeyID)
		return out.(*models.APIKey), err
	}
	panic("RetrieveAPIKey callback not set")
}

func (s *Store) UpdateAPIKey(ctx context.Context, in *models.APIKey) error {
	s.calls["UpdateAPIKey"]++
	if fn, ok := s.callbacks["UpdateAPIKey"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, in)
	}
	panic("UpdateAPIKey callback not set")
}

func (s *Store) DeleteAPIKey(ctx context.Context, keyID ulid.ULID) error {
	s.calls["DeleteAPIKey"]++
	if fn, ok := s.callbacks["DeleteAPIKey"]; ok {
		return fn.(DeleteStoreULIDFn)(ctx, keyID)
	}
	panic("DeleteAPIKey callback not set")
}

//===========================================================================
// Reset Password Link Store Methods
//===========================================================================

func (s *Store) ListResetPasswordLinks(ctx context.Context, page *models.PageInfo) (*models.ResetPasswordLinkPage, error) {
	s.calls["ListResetPasswordLinks"]++
	if fn, ok := s.callbacks["ListResetPasswordLinks"]; ok {
		out, err := fn.(ListStoreFn)(ctx, page)
		return out.(*models.ResetPasswordLinkPage), err
	}
	panic("ListResetPasswordLinks callback not set")
}

func (s *Store) CreateResetPasswordLink(ctx context.Context, link *models.ResetPasswordLink) error {
	s.calls["CreateResetPasswordLink"]++
	if fn, ok := s.callbacks["CreateResetPasswordLink"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, link)
	}
	panic("CreateResetPasswordLink callback not set")
}

func (s *Store) RetrieveResetPasswordLink(ctx context.Context, linkID ulid.ULID) (*models.ResetPasswordLink, error) {
	s.calls["RetrieveResetPasswordLink"]++
	if fn, ok := s.callbacks["RetrieveResetPasswordLink"]; ok {
		out, err := fn.(RetrieveStoreULIDFn)(ctx, linkID)
		return out.(*models.ResetPasswordLink), err
	}
	panic("RetrieveResetPasswordLink callback not set")
}

func (s *Store) UpdateResetPasswordLink(ctx context.Context, link *models.ResetPasswordLink) error {
	s.calls["UpdateResetPasswordLink"]++
	if fn, ok := s.callbacks["UpdateResetPasswordLink"]; ok {
		return fn.(CreateUpdateStoreFn)(ctx, link)
	}
	panic("UpdateResetPasswordLink callback not set")
}

func (s *Store) DeleteResetPasswordLink(ctx context.Context, linkID ulid.ULID) (err error) {
	s.calls["DeleteResetPasswordLink"]++
	if fn, ok := s.callbacks["DeleteResetPasswordLink"]; ok {
		return fn.(DeleteStoreULIDFn)(ctx, linkID)
	}
	panic("DeleteResetPasswordLink callback not set")
}
