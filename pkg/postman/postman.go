/*
 * Postman providers helper functionality for managing TRISA and TRP transfers and
 * sorting them and storing them in the database. This package is intended to unify the
 * functionality across the TRISA node, the TRP node, and the Web API/UI.
 */
package postman

// Postman is a factory that creates transfer helpers for incoming and outgoing
// messages. It acts as a glue, unifying the behavior of sending messages from the web
// API or user interface, receiving messages via the TRISA node and receiving messages
// via the TRP node. It ensures that all transactions are handled consistently and that
// the database is updated appropriately in all directions.
type Postman struct {
}
