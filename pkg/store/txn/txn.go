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

	// Returns actor metadata for compliance audit logging.
	SetActor(actorID []byte, actorType enum.Actor)
	GetActor() (actorID []byte, actorType enum.Actor)

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
	CreateTransaction(*models.Transaction, *models.ComplianceAuditLog) error
	RetrieveTransaction(uuid.UUID) (*models.Transaction, error)
	UpdateTransaction(*models.Transaction, *models.ComplianceAuditLog) error
	DeleteTransaction(uuid.UUID, *models.ComplianceAuditLog) error
	ArchiveTransaction(uuid.UUID, *models.ComplianceAuditLog) error
	UnarchiveTransaction(uuid.UUID, *models.ComplianceAuditLog) error
	CountTransactions() (*models.TransactionCounts, error)
	TransactionState(uuid.UUID) (archived bool, status enum.Status, err error)
}

// SecureEnvelopes are associated with individual transactions.
type SecureEnvelopeTxn interface {
	ListSecureEnvelopes(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error)
	CreateSecureEnvelope(*models.SecureEnvelope, *models.ComplianceAuditLog) error
	RetrieveSecureEnvelope(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error)
	UpdateSecureEnvelope(*models.SecureEnvelope, *models.ComplianceAuditLog) error
	DeleteSecureEnvelope(txID uuid.UUID, envID ulid.ULID, auditLog *models.ComplianceAuditLog) error
	LatestSecureEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
	LatestPayloadEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error)
}

// AccountTxn provides CRUD interactions with Account models.
type AccountTxn interface {
	CryptoAddressTxn
	ListAccounts(page *models.PageInfo) (*models.AccountsPage, error)
	CreateAccount(*models.Account, *models.ComplianceAuditLog) error
	LookupAccount(cryptoAddress string) (*models.Account, error)
	RetrieveAccount(id ulid.ULID) (*models.Account, error)
	UpdateAccount(*models.Account, *models.ComplianceAuditLog) error
	DeleteAccount(id ulid.ULID, auditLog *models.ComplianceAuditLog) error
	ListAccountTransactions(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error)
}

// CryptoAddressTxn provides CRUD interactions with CryptoAddress models and their
// associated Account model.
type CryptoAddressTxn interface {
	ListCryptoAddresses(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error)
	CreateCryptoAddress(*models.CryptoAddress, *models.ComplianceAuditLog) error
	RetrieveCryptoAddress(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error)
	UpdateCryptoAddress(*models.CryptoAddress, *models.ComplianceAuditLog) error
	DeleteCryptoAddress(accountID, cryptoAddressID ulid.ULID, auditLog *models.ComplianceAuditLog) error
}

// Counterparty store provides CRUD interactions with Counterparty models.
type CounterpartyTxn interface {
	SearchCounterparties(query *models.SearchQuery) (*models.CounterpartyPage, error)
	ListCounterparties(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error)
	ListCounterpartySourceInfo(source enum.Source) ([]*models.CounterpartySourceInfo, error)
	CreateCounterparty(*models.Counterparty, *models.ComplianceAuditLog) error
	RetrieveCounterparty(counterpartyID ulid.ULID) (*models.Counterparty, error)
	LookupCounterparty(field, value string) (*models.Counterparty, error)
	UpdateCounterparty(*models.Counterparty, *models.ComplianceAuditLog) error
	DeleteCounterparty(counterpartyID ulid.ULID, auditLog *models.ComplianceAuditLog) error
}

type ContactTxn interface {
	ListContacts(counterparty any, page *models.PageInfo) (*models.ContactsPage, error)
	CreateContact(*models.Contact, *models.ComplianceAuditLog) error
	RetrieveContact(contactID, counterpartyID any) (*models.Contact, error)
	UpdateContact(*models.Contact, *models.ComplianceAuditLog) error
	DeleteContact(contactID, counterpartyID any, auditLog *models.ComplianceAuditLog) error
}

// Sunrise store manages both contacts and counterparties.
type SunriseTxn interface {
	ListSunrise(*models.PageInfo) (*models.SunrisePage, error)
	CreateSunrise(*models.Sunrise, *models.ComplianceAuditLog) error
	RetrieveSunrise(ulid.ULID) (*models.Sunrise, error)
	UpdateSunrise(*models.Sunrise, *models.ComplianceAuditLog) error
	UpdateSunriseStatus(uuid.UUID, enum.Status, *models.ComplianceAuditLog) error
	DeleteSunrise(ulid.ULID, *models.ComplianceAuditLog) error
	GetOrCreateSunriseCounterparty(email, name string, auditLog *models.ComplianceAuditLog) (*models.Counterparty, error)
}

type UserTxn interface {
	ListUsers(page *models.UserPageInfo) (*models.UserPage, error)
	CreateUser(*models.User, *models.ComplianceAuditLog) error
	RetrieveUser(emailOrUserID any) (*models.User, error)
	UpdateUser(*models.User, *models.ComplianceAuditLog) error
	// NOTE: password update does not require an audit log entry:
	SetUserPassword(userID ulid.ULID, password string) error
	// NOTE: last login time update does not require an audit log entry:
	SetUserLastLogin(userID ulid.ULID, lastLogin time.Time) error
	DeleteUser(userID ulid.ULID, auditLog *models.ComplianceAuditLog) error
	LookupRole(role string) (*models.Role, error)
}

type APIKeyTxn interface {
	ListAPIKeys(*models.PageInfo) (*models.APIKeyPage, error)
	CreateAPIKey(*models.APIKey, *models.ComplianceAuditLog) error
	RetrieveAPIKey(clientIDOrKeyID any) (*models.APIKey, error)
	UpdateAPIKey(*models.APIKey, *models.ComplianceAuditLog) error
	DeleteAPIKey(keyID ulid.ULID, auditLog *models.ComplianceAuditLog) error
}

type ResetPasswordLinkTxn interface {
	// NOTE: no audit logs required for ResetPasswordLinkTxn resource
	ListResetPasswordLinks(*models.PageInfo) (*models.ResetPasswordLinkPage, error)
	CreateResetPasswordLink(*models.ResetPasswordLink) error
	RetrieveResetPasswordLink(ulid.ULID) (*models.ResetPasswordLink, error)
	UpdateResetPasswordLink(*models.ResetPasswordLink) error
	DeleteResetPasswordLink(ulid.ULID) error
}

type ComplianceAuditLogTxn interface {
	ListComplianceAuditLogs(*models.ComplianceAuditLogPageInfo) (*models.ComplianceAuditLogPage, error)
	CreateComplianceAuditLog(*models.ComplianceAuditLog) error
	RetrieveComplianceAuditLog(ulid.ULID) (*models.ComplianceAuditLog, error)
	// NOTE: ComplianceAuditLogs are required to be immutable; do not create Update or Delete functions
}

// Methods required for managing Daybreak records in the database. This interface allows
// us to have a single transaction open for a daybreak operation so that with respect
// to a single counterparty we completely create the record or rollback on failure.
//
// NOTE: this is not part of the Txn interface since it is not required for the
// server to function but is useful for Daybreak-specific operations.
type DaybreakTxn interface {
	// NOTE: no audit logs required for DaybreakTxn resource
	ListDaybreak() (map[string]*models.CounterpartySourceInfo, error)
	CreateDaybreak(counterparty *models.Counterparty) error
	UpdateDaybreak(counterparty *models.Counterparty) error
	DeleteDaybreak(counterpartyID ulid.ULID, ignoreTxns bool) error
}
