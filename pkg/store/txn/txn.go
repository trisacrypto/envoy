/*
Unfortunately, to prevent import cycle, we have to put the transaction interface in a
subpackage of store so that store and other packages can import it. This means that
whenver a new database interface is created, we have to also implement the parallel
transaction interface as well.
*/
package txn

// Tx is a storage interface for executing multiple operations against the database so
// that if all operations succeed, the transaction can be committed. If any operation
// fails, the transaction can be rolled back to ensure that the database is not left in
// an inconsistent state. Tx should have similar methods to the Store interface, but
// without requiring the context (this is passed to the transaction when it is created).
type Tx interface {
	Rollback() error
	Commit() error
}
