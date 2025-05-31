package mock

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

type PreparedTransaction struct {
	callbacks map[string]any
	calls     map[string]int
	commit    bool
	rollback  bool
}

//===========================================================================
// Mock Helper Methods
//===========================================================================

// Reset all the calls and callbacks in the PreparedTransaction, if you don't
// want to create a new one.
func (p *PreparedTransaction) Reset() {
	// Set maps to nil to free up memory
	p.calls = nil
	p.callbacks = nil

	// Create new calls and callbacks maps
	p.calls = make(map[string]int)
	p.callbacks = make(map[string]any)

	// Reset transaction commit/rollback
	p.commit = false
	p.rollback = false
}

// Assert that the expected number of calls were made to the given method.
func (p *PreparedTransaction) AssertCalls(t testing.TB, method string, expected int) {
	require.Equal(t, expected, p.calls[method], "expected %d calls to %s, got %d", expected, method, p.calls[method])
}

// Assert that Commit has been called on the PreparedTransaction.
func (p *PreparedTransaction) AssertCommit(t testing.TB) {
	require.True(t, p.commit, "expected Commit to be called")
}

// Assert that Rollback has been called on the PreparedTransaction without commit.
func (p *PreparedTransaction) AssertRollback(t testing.TB) {
	require.True(t, p.rollback && !p.commit, "expected Rollback to be called but not Commit")
}

// Check is a helper method that determines if the PreparedTransaction is committed
// or rolled back. If so it returns ErrpDone no matter if there is a callback set.
// Check will record the calls to the method on the transaction, and finally, if
// the method is not set in callbacks, it panics.
func (p *PreparedTransaction) check(method string) (any, error) {
	p.calls[method]++

	if p.commit || p.rollback {
		return nil, sql.ErrTxDone
	}

	if fn, ok := p.callbacks[method]; ok {
		return fn, nil
	}

	panic(fmt.Errorf("%q callback not set", method))
}

//===========================================================================
// Prepared Transaction Store Methods
//===========================================================================

// Sets a callback for when "Created()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnCreated(fn func() bool) {
	p.callbacks["Created"] = fn
}

// Calls the callback previously set with "OnCreated()".
func (p *PreparedTransaction) Created() bool {
	// this one doesn't need to go through `p.check()`
	p.calls["Created"]++
	if fn, ok := p.callbacks["Created"]; ok {
		return fn.(func() bool)()
	}
	panic(fmt.Errorf("Created callback not set"))

}

// Sets a callback for when "Fetch()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnFetch(fn func() (*models.Transaction, error)) {
	p.callbacks["Fetch"] = fn
}

// Calls the callback previously set with "OnFetch()".
func (p *PreparedTransaction) Fetch() (*models.Transaction, error) {
	fn, err := p.check("Fetch")
	if err != nil {
		return nil, err
	}

	return fn.(func() (*models.Transaction, error))()
}

// Sets a callback for when "Update()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnUpdate(fn func(*models.Transaction) error) {
	p.callbacks["Update"] = fn
}

// Calls the callback previously set with "OnUpdate()".
func (p *PreparedTransaction) Update(model *models.Transaction) error {
	fn, err := p.check("Update")
	if err != nil {
		return err
	}

	return fn.(func(*models.Transaction) error)(model)
}

// Sets a callback for when "AddCounterparty()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnAddCounterparty(fn func(*models.Counterparty) error) {
	p.callbacks["AddCounterparty"] = fn
}

// Calls the callback previously set with "OnAddCounterparty()".
func (p *PreparedTransaction) AddCounterparty(model *models.Counterparty) error {
	fn, err := p.check("AddCounterparty")
	if err != nil {
		return err
	}

	return fn.(func(*models.Counterparty) error)(model)
}

// Sets a callback for when "UpdateCounterparty()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnUpdateCounterparty(fn func(counterparty *models.Counterparty) error) {
	p.callbacks["UpdateCounterparty"] = fn
}

// Calls the callback previously set with "OnUpdateCounterparty()".
func (p *PreparedTransaction) UpdateCounterparty(counterparty *models.Counterparty) error {
	fn, err := p.check("UpdateCounterparty")
	if err != nil {
		return err
	}

	return fn.(func(counterparty *models.Counterparty) error)(counterparty)
}

// Sets a callback for when "LookupCounterparty()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnLookupCounterparty(fn func(field, value string) (*models.Counterparty, error)) {
	p.callbacks["LookupCounterparty"] = fn
}

// Calls the callback previously set with "OnLookupCounterparty()".
func (p *PreparedTransaction) LookupCounterparty(field, value string) (*models.Counterparty, error) {
	fn, err := p.check("LookupCounterparty")
	if err != nil {
		return nil, err
	}

	return fn.(func(field, value string) (*models.Counterparty, error))(field, value)
}

// Sets a callback for when "AddEnvelope()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnAddEnvelope(fn func(*models.SecureEnvelope) error) {
	p.callbacks["AddEnvelope"] = fn
}

// Calls the callback previously set with "OnAddEnvelope()".
func (p *PreparedTransaction) AddEnvelope(model *models.SecureEnvelope) error {
	fn, err := p.check("AddEnvelope")
	if err != nil {
		return err
	}

	return fn.(func(*models.SecureEnvelope) error)(model)
}

// Sets a callback for when "CreateSunrise()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnCreateSunrise(fn func(sunrise *models.Sunrise) error) {
	p.callbacks["CreateSunrise"] = fn
}

// Calls the callback previously set with "OnCreateSunrise()".
func (p *PreparedTransaction) CreateSunrise(sunrise *models.Sunrise) error {
	fn, err := p.check("CreateSunrise")
	if err != nil {
		return err
	}

	return fn.(func(*models.Sunrise) error)(sunrise)
}

// Sets a callback for when "UpdateSunrise()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnUpdateSunrise(fn func(sunrise *models.Sunrise) error) {
	p.callbacks["UpdateSunrise"] = fn
}

// Calls the callback previously set with "OnUpdateSunrise()".
func (p *PreparedTransaction) UpdateSunrise(sunrise *models.Sunrise) error {
	fn, err := p.check("UpdateSunrise")
	if err != nil {
		return err
	}

	return fn.(func(*models.Sunrise) error)(sunrise)
}

// Sets a callback for when "UpdateSunriseStatus()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnUpdateSunriseStatus(fn func(id uuid.UUID, status enum.Status) error) {
	p.callbacks["UpdateSunriseStatus"] = fn
}

// Calls the callback previously set with "OnUpdateSunriseStatus()".
func (p *PreparedTransaction) UpdateSunriseStatus(id uuid.UUID, status enum.Status) error {
	fn, err := p.check("UpdateSunriseStatus")
	if err != nil {
		return err
	}

	return fn.(func(uuid.UUID, enum.Status) error)(id, status)
}

// Sets a callback for when "Rollback()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnRollback(fn func() error) {
	p.callbacks["Rollback"] = fn
}

// Calls the callback previously set with "OnRollback()". If no callback is set, it
// will "rollback" the transaction.
func (p *PreparedTransaction) Rollback() error {
	p.calls["Rollback"]++

	// do callback if present (the user can manage checking txn status)
	if fn, ok := p.callbacks["Rollback"]; ok {
		return fn.(func() error)()
	}

	// ensure the transaction is still active
	if p.commit || p.rollback {
		return sql.ErrTxDone
	}

	// complete the rollback
	p.rollback = true
	return nil
}

// Sets a callback for when "Commit()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnCommit(fn func() error) {
	p.callbacks["Commit"] = fn
}

// Calls the callback previously set with "OnCommit()". If no callback is set, it
// will "commit" the transaction.
func (p *PreparedTransaction) Commit() error {
	p.calls["Commit"]++

	// do callback if present (the user can manage checking txn status)
	if fn, ok := p.callbacks["Commit"]; ok {
		return fn.(func() error)()
	}

	// ensure the transaction is still active
	if p.commit || p.rollback {
		return sql.ErrTxDone
	}

	// complete the commit
	p.commit = true
	return nil
}
