package mock

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"go.rtnl.ai/ulid"
)

type Tx struct{}

func (tx *Tx) Commit() error {
	return nil
}

func (tx *Tx) Rollback() error {
	return nil
}

func (tx *Tx) ListTransactions(*models.TransactionPageInfo) (*models.TransactionPage, error) {
	return nil, nil
}

func (tx *Tx) CreateTransaction(*models.Transaction) error {
	return nil
}

func (tx *Tx) RetrieveTransaction(uuid.UUID) (*models.Transaction, error) {
	return nil, nil
}

func (tx *Tx) UpdateTransaction(*models.Transaction) error {
	return nil
}

func (tx *Tx) DeleteTransaction(uuid.UUID) error {
	return nil
}

func (tx *Tx) ArchiveTransaction(uuid.UUID) error {
	return nil
}

func (tx *Tx) UnarchiveTransaction(uuid.UUID) error {
	return nil
}

func (tx *Tx) CountTransactions() (*models.TransactionCounts, error) {
	return nil, nil
}

func (tx *Tx) TransactionState(uuid.UUID) (archived bool, status enum.Status, err error) {
	return false, enum.StatusUnspecified, nil
}

func (tx *Tx) ListSecureEnvelopes(txID uuid.UUID, page *models.PageInfo) (*models.SecureEnvelopePage, error) {
	return nil, nil
}

func (tx *Tx) CreateSecureEnvelope(*models.SecureEnvelope) error {
	return nil
}

func (tx *Tx) RetrieveSecureEnvelope(txID uuid.UUID, envID ulid.ULID) (*models.SecureEnvelope, error) {
	return nil, nil
}

func (tx *Tx) UpdateSecureEnvelope(*models.SecureEnvelope) error {
	return nil
}

func (tx *Tx) DeleteSecureEnvelope(txID uuid.UUID, envID ulid.ULID) error {
	return nil
}

func (tx *Tx) LatestSecureEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	return nil, nil
}

func (tx *Tx) LatestPayloadEnvelope(txID uuid.UUID, direction enum.Direction) (*models.SecureEnvelope, error) {
	return nil, nil
}

func (tx *Tx) ListAccounts(page *models.PageInfo) (*models.AccountsPage, error) {
	return nil, nil
}

func (tx *Tx) CreateAccount(*models.Account) error {
	return nil
}

func (tx *Tx) LookupAccount(cryptoAddress string) (*models.Account, error) {
	return nil, nil
}

func (tx *Tx) RetrieveAccount(id ulid.ULID) (*models.Account, error) {
	return nil, nil
}

func (tx *Tx) UpdateAccount(*models.Account) error {
	return nil
}

func (tx *Tx) DeleteAccount(id ulid.ULID) error {
	return nil
}

func (tx *Tx) ListAccountTransactions(accountID ulid.ULID, page *models.TransactionPageInfo) (*models.TransactionPage, error) {
	return nil, nil
}

func (tx *Tx) ListCryptoAddresses(accountID ulid.ULID, page *models.PageInfo) (*models.CryptoAddressPage, error) {
	return nil, nil
}

func (tx *Tx) CreateCryptoAddress(*models.CryptoAddress) error {
	return nil
}

func (tx *Tx) RetrieveCryptoAddress(accountID, cryptoAddressID ulid.ULID) (*models.CryptoAddress, error) {
	return nil, nil
}

func (tx *Tx) UpdateCryptoAddress(*models.CryptoAddress) error {
	return nil
}

func (tx *Tx) DeleteCryptoAddress(accountID, cryptoAddressID ulid.ULID) error {
	return nil
}

func (tx *Tx) SearchCounterparties(query *models.SearchQuery) (*models.CounterpartyPage, error) {
	return nil, nil
}

func (tx *Tx) ListCounterparties(page *models.CounterpartyPageInfo) (*models.CounterpartyPage, error) {
	return nil, nil
}

func (tx *Tx) ListCounterpartySourceInfo(source enum.Source) ([]*models.CounterpartySourceInfo, error) {
	return nil, nil
}

func (tx *Tx) CreateCounterparty(*models.Counterparty) error {
	return nil
}

func (tx *Tx) RetrieveCounterparty(counterpartyID ulid.ULID) (*models.Counterparty, error) {
	return nil, nil
}

func (tx *Tx) LookupCounterparty(field, value string) (*models.Counterparty, error) {
	return nil, nil
}

func (tx *Tx) UpdateCounterparty(*models.Counterparty) error {
	return nil
}

func (tx *Tx) DeleteCounterparty(counterpartyID ulid.ULID) error {
	return nil
}

func (tx *Tx) ListContacts(counterparty any, page *models.PageInfo) (*models.ContactsPage, error) {
	return nil, nil
}

func (tx *Tx) CreateContact(*models.Contact) error {
	return nil
}

func (tx *Tx) RetrieveContact(contactID, counterpartyID any) (*models.Contact, error) {
	return nil, nil
}

func (tx *Tx) UpdateContact(*models.Contact) error {
	return nil
}

func (tx *Tx) DeleteContact(contactID, counterpartyID any) error {
	return nil
}

func (tx *Tx) ListSunrise(*models.PageInfo) (*models.SunrisePage, error) {
	return nil, nil
}

func (tx *Tx) CreateSunrise(*models.Sunrise) error {
	return nil
}

func (tx *Tx) RetrieveSunrise(ulid.ULID) (*models.Sunrise, error) {
	return nil, nil
}

func (tx *Tx) UpdateSunrise(*models.Sunrise) error {
	return nil
}

func (tx *Tx) UpdateSunriseStatus(uuid.UUID, enum.Status) error {
	return nil
}

func (tx *Tx) DeleteSunrise(ulid.ULID) error {
	return nil
}

func (tx *Tx) GetOrCreateSunriseCounterparty(email, name string) (*models.Counterparty, error) {
	return nil, nil
}

func (tx *Tx) ListUsers(page *models.UserPageInfo) (*models.UserPage, error) {
	return nil, nil
}

func (tx *Tx) CreateUser(*models.User) error {
	return nil
}

func (tx *Tx) RetrieveUser(emailOrUserID any) (*models.User, error) {
	return nil, nil
}

func (tx *Tx) UpdateUser(*models.User) error {
	return nil
}

func (tx *Tx) SetUserPassword(userID ulid.ULID, password string) error {
	return nil
}

func (tx *Tx) SetUserLastLogin(userID ulid.ULID, lastLogin time.Time) error {
	return nil
}

func (tx *Tx) DeleteUser(userID ulid.ULID) error {
	return nil
}

func (tx *Tx) LookupRole(role string) (*models.Role, error) {
	return nil, nil
}

func (tx *Tx) ListAPIKeys(*models.PageInfo) (*models.APIKeyPage, error) {
	return nil, nil
}

func (tx *Tx) CreateAPIKey(*models.APIKey) error {
	return nil
}

func (tx *Tx) RetrieveAPIKey(clientIDOrKeyID any) (*models.APIKey, error) {
	return nil, nil
}

func (tx *Tx) UpdateAPIKey(*models.APIKey) error {
	return nil
}

func (tx *Tx) DeleteAPIKey(keyID ulid.ULID) error {
	return nil
}

func (tx *Tx) ListResetPasswordLinks(*models.PageInfo) (*models.ResetPasswordLinkPage, error) {
	return nil, nil
}

func (tx *Tx) CreateResetPasswordLink(*models.ResetPasswordLink) error {
	return nil
}

func (tx *Tx) RetrieveResetPasswordLink(ulid.ULID) (*models.ResetPasswordLink, error) {
	return nil, nil
}

func (tx *Tx) UpdateResetPasswordLink(*models.ResetPasswordLink) error {
	return nil
}

func (tx *Tx) DeleteResetPasswordLink(ulid.ULID) error {
	return nil
}

func (tx *Tx) ListDaybreak(ctx context.Context) (map[string]*models.CounterpartySourceInfo, error) {
	return nil, nil
}

func (tx *Tx) CreateDaybreak(counterparty *models.Counterparty) error {
	return nil
}

func (tx *Tx) UpdateDaybreak(counterparty *models.Counterparty) error {
	return nil
}

func (tx *Tx) DeleteDaybreak(counterpartyID ulid.ULID, ignoreTxns bool) error {
	return nil
}
