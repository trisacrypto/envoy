package mock

import "github.com/trisacrypto/envoy/pkg/store/models"

type PreparedTransaction struct {
	callbacks map[string]any
	calls     map[string]int
}

// Sets a callback for when "Created()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnCreated(fn func() bool) {
	p.callbacks["Created"] = fn
}

// Calls the callback previously set with "OnCreated()".
func (p *PreparedTransaction) Created() bool {
	p.calls["Created"]++
	if fn, ok := p.callbacks["Created"]; ok {
		return fn.(func() bool)()
	}
	panic("No callback set: Created()")
}

// Sets a callback for when "Fetch()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnFetch(fn func() (*models.Transaction, error)) {
	p.callbacks["Fetch"] = fn
}

// Calls the callback previously set with "OnFetch()".
func (p *PreparedTransaction) Fetch() (*models.Transaction, error) {
	p.calls["Fetch"]++
	if fn, ok := p.callbacks["Fetch"]; ok {
		return fn.(func() (*models.Transaction, error))()
	}
	panic("No callback set: Fetch()")
}

// Sets a callback for when "Update()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnUpdate(fn func(*models.Transaction) error) {
	p.callbacks["Update"] = fn
}

// Calls the callback previously set with "OnUpdate()".
func (p *PreparedTransaction) Update(model *models.Transaction) error {
	p.calls["Update"]++
	if fn, ok := p.callbacks["Update"]; ok {
		return fn.(func(*models.Transaction) error)(model)
	}
	panic("No callback set: Update()")
}

// Sets a callback for when "AddCounterparty()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnAddCounterparty(fn func(*models.Counterparty) error) {
	p.callbacks["AddCounterparty"] = fn
}

// Calls the callback previously set with "OnAddCounterparty()".
func (p *PreparedTransaction) AddCounterparty(model *models.Counterparty) error {
	p.calls["AddCounterparty"]++
	if fn, ok := p.callbacks["AddCounterparty"]; ok {
		return fn.(func(*models.Counterparty) error)(model)
	}
	panic("No callback set: AddCounterparty()")
}

// Sets a callback for when "AddEnvelope()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnAddEnvelope(fn func(*models.SecureEnvelope) error) {
	p.callbacks["AddEnvelope"] = fn
}

// Calls the callback previously set with "OnAddEnvelope()".
func (p *PreparedTransaction) AddEnvelope(model *models.SecureEnvelope) error {
	p.calls["AddEnvelope"]++
	if fn, ok := p.callbacks["AddEnvelope"]; ok {
		return fn.(func(*models.SecureEnvelope) error)(model)
	}
	panic("No callback set: AddEnvelope()")
}

// Sets a callback for when "Rollback()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnRollback(fn func() error) {
	p.callbacks["Rollback"] = fn
}

// Calls the callback previously set with "OnRollback()".
func (p *PreparedTransaction) Rollback() error {
	p.calls["Rollback"]++
	if fn, ok := p.callbacks["Rollback"]; ok {
		return fn.(func() error)()
	}
	panic("No callback set: Rollback()")
}

// Sets a callback for when "Commit()" is called on the mock PreparedTransaction.
func (p *PreparedTransaction) OnCommit(fn func() error) {
	p.callbacks["Commit"] = fn
}

// Calls the callback previously set with "OnCommit()".
func (p *PreparedTransaction) Commit() error {
	p.calls["Commit"]++
	if fn, ok := p.callbacks["Commit"]; ok {
		return fn.(func() error)()
	}
	panic("No callback set: Commit()")
}
