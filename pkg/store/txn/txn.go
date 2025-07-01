/*
Unfortunately, to prevent import cycle, we have to put the transaction interface in a
subpackage of store so that store and other packages can import it. This means that
whenver a new database interface is created, we have to also implement the parallel
transaction interface as well.
*/
package txn

import (
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

// Txn is a storage interface for executing multiple operations against the database so
// that if all operations succeed, the transaction can be committed. If any operation
// fails, the transaction can be rolled back to ensure that the database is not left in
// an inconsistent state. Txn should have similar methods to the Store interface, but
// without requiring the context (this is passed to the transaction when it is created).
type Txn interface {
	Rollback() error
	Commit() error

	TransactionTxn
	AccountTxn
	CounterpartyTxn
	ContactTxn
	SunriseTxn
	UserTxn
	APIKeyTxn
	ResetPasswordLinkTxn
	ComplianceAuditLogTxn
}

// TransactionTxn stores some lightweight information about specific transactions
// stored in the database (most of which is not sensitive and is used for indexing).
// It also maintains an association with all secure envelopes sent and received as
// part of completing a travel rule exchange for the transaction.
type TransactionTxn interface {
	SecureEnvelopeTxn
	ListTransactions(*models.TransactionPageInfo) (*models.TransactionPage, error)
	CreateTransaction(*models.Transaction) error
	RetrieveTransaction(uuid.UUID) (*models.Transaction, error)
	UpdateTransaction(*models.Transaction) error
	DeleteTransaction(uuid.UUID) error
	ArchiveTransaction(uuid.UUID) error
	UnarchiveTransaction(uuid.UUID) error
	CountTransactions() (*models.TransactionCounts, error)
	TransactionState(uuid.UUID) (archived bool, status enum.Status, err error)
}

// SecureEnvelopes are associated with individual transactions.
type SecureEnvelopeTxn interface {
	ListSecureEnvelopes(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error)
	CreateSecureEnvelope(*models.SecureEnvelope) error
	RetrieveSecureEnvelope(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error)
	UpdateSecureEnvelope(*models.SecureEnvelope) error
	DeleteSecureEnvelope(txID uuid.UUID, envID ulid.ULID) error
	LatestSecureEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
	LatestPayloadEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
}

// AccountTxn provides CRUD interactions with Account models.
type AccountTxn interface {
	CryptoAddressTxn
	ListAccounts(page *models.PageInfo) (*models.AccountsPage, error)
	CreateAccount(*models.Account) error
	LookupAccount(cryptoAddress string) (*models.Account, error)
	RetrieveAccount(id ulid.ULID) (*models.Account, error)
	UpdateAccount(*models.Account) error
	DeleteAccount(id ulid.ULID) error
	ListAccountTransactions(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error)
}

// CryptoAddressTxn provides CRUD interactions with CryptoAddress models and their
// associated Account model.
type CryptoAddressTxn interface {
	ListCryptoAddresses(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error)
	CreateCryptoAddress(*models.CryptoAddress) error
	RetrieveCryptoAddress(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error)
	UpdateCryptoAddress(*models.CryptoAddress) error
	DeleteCryptoAddress(accountID, cryptoAddressID ulid.ULID) error
}

// Counterparty store provides CRUD interactions with Counterparty models.
type CounterpartyTxn interface {
	SearchCounterparties(query *models.SearchQuery) (*models.CounterpartyPage, error)
	ListCounterparties(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error)
	ListCounterpartySourceInfo(source enum.Source) ([]*models.CounterpartySourceInfo, error)
	CreateCounterparty(*models.Counterparty) error
	RetrieveCounterparty(counterpartyID ulid.ULID) (*models.Counterparty, error)
	LookupCounterparty(field, value string) (*models.Counterparty, error)
	UpdateCounterparty(*models.Counterparty) error
	DeleteCounterparty(counterpartyID ulid.ULID) error
}

type ContactTxn interface {
	ListContacts(counterparty any, page *models.PageInfo) (*models.ContactsPage, error)
	CreateContact(*models.Contact) error
	RetrieveContact(contactID, counterpartyID any) (*models.Contact, error)
	UpdateContact(*models.Contact) error
	DeleteContact(contactID, counterpartyID any) error
}

// Sunrise store manages both contacts and counterparties.
type SunriseTxn interface {
	ListSunrise(*models.PageInfo) (*models.SunrisePage, error)
	CreateSunrise(*models.Sunrise) error
	RetrieveSunrise(ulid.ULID) (*models.Sunrise, error)
	UpdateSunrise(*models.Sunrise) error
	UpdateSunriseStatus(uuid.UUID, enum.Status) error
	DeleteSunrise(ulid.ULID) error
	GetOrCreateSunriseCounterparty(email, name string) (*models.Counterparty, error)
}

type UserTxn interface {
	ListUsers(page *models.UserPageInfo) (*models.UserPage, error)
	CreateUser(*models.User) error
	RetrieveUser(emailOrUserID any) (*models.User, error)
	UpdateUser(*models.User) error
	SetUserPassword(userID ulid.ULID, password string) error
	SetUserLastLogin(userID ulid.ULID, lastLogin time.Time) error
	DeleteUser(userID ulid.ULID) error
	LookupRole(role string) (*models.Role, error)
}

type APIKeyTxn interface {
	ListAPIKeys(*models.PageInfo) (*models.APIKeyPage, error)
	CreateAPIKey(*models.APIKey) error
	RetrieveAPIKey(clientIDOrKeyID any) (*models.APIKey, error)
	UpdateAPIKey(*models.APIKey) error
	DeleteAPIKey(keyID ulid.ULID) error
}

type ResetPasswordLinkTxn interface {
	ListResetPasswordLinks(*models.PageInfo) (*models.ResetPasswordLinkPage, error)
	CreateResetPasswordLink(*models.ResetPasswordLink) error
	RetrieveResetPasswordLink(ulid.ULID) (*models.ResetPasswordLink, error)
	UpdateResetPasswordLink(*models.ResetPasswordLink) error
	DeleteResetPasswordLink(ulid.ULID) error
}

type ComplianceAuditLogTxn interface {
	ListComplianceAuditLogs(*models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error)
	CreateComplianceAuditLog(*models.ComplianceAuditLog) error
	// NOTE: ComplianceAuditLogs are required to be immutable; do not create Update or Delete functions
}

// Methods required for managing Daybreak records in the database. This interface allows
// us to have a single transaction open for a daybreak operation so that with respect
// to a single counterparty we completely create the record or rollback on failure.
//
// NOTE: this is not part of the Txn interface since it is not required for the
// server to function but is useful for Daybreak-specific operations.
type DaybreakTxn interface {
	ListDaybreak() (map[string]*models.CounterpartySourceInfo, error)
	CreateDaybreak(counterparty *models.Counterparty) error
	UpdateDaybreak(counterparty *models.Counterparty) error
	DeleteDaybreak(counterpartyID ulid.ULID, ignoreTxns bool) error
}
