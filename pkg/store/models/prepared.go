package models

// Standard interface for Prepared SQL Interfaces
type SQL interface {
	Rollback() error // Rollback the prepared transaction and conclude it
	Commit() error   // Commit the prepared transaction and conclude it
}

// Transaction allows you to manage the creation/modification of a transaction
// w.r.t a secure envelope. It is unified in a single interface to allow backend stores
// that have database transactions to perform all operations in a single transaction
// without concurrency issues.
type PreparedTransaction interface {
	SQL
	PreparedSunrise
	Created() bool                       // Returns true if the transaction was newly created, false if it already existed
	Fetch() (*Transaction, error)        // Fetches the current transaction record from the database
	Update(*Transaction) error           // Update the transaction with new information; e.g. data from decryption
	AddCounterparty(*Counterparty) error // Add counterparty by database ULID, counterparty name, or registered directory ID; if the counterparty doesn't exist, it is created
	AddEnvelope(*SecureEnvelope) error   // Associate a secure envelope with the prepared transaction
}

// Sunrise allows you to manage the creation/modification of sunrise messages within
// the scope of a single transaction.
type PreparedSunrise interface {
	CreateSunrise(*Sunrise) error // Create a sunrise message sent to the counterparty for the transaction
	UpdateSunrise(*Sunrise) error // Update the sunrise message associated with the transaction
}
