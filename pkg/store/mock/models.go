package mock

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"go.rtnl.ai/ulid"
	"go.rtnl.ai/x/vero"
)

// ============================================================================
// This file contains functions that allow a test-creator to get a "sample"
// of each of the models in `store/models`. These "sample" models are filled
// with dummy data of the correct types and can be used for most test
// operations.
// ============================================================================

// Returns a sample Account. Can add the IVMS101 and CryptoAddresses and include
// or exclude `NullType` values. `zeroID` returns the model with a zeroed ULID.
func GetSampleAccount(includeNulls bool, addIvms101 bool, addCrypto bool) (account *models.Account) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	account = &models.Account{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow.Add(1 * time.Hour)},
		CustomerID:    sql.NullString{},
		FirstName:     sql.NullString{},
		LastName:      sql.NullString{},
		TravelAddress: sql.NullString{},
		IVMSRecord:    nil,
	}

	if includeNulls {
		account.CustomerID = sql.NullString{String: "CustomerID", Valid: true}
		account.FirstName = sql.NullString{String: "FirstName", Valid: true}
		account.LastName = sql.NullString{String: "LastName", Valid: true}
		account.TravelAddress = sql.NullString{String: "TravelAddress", Valid: true}
	}

	if addIvms101 {
		account.IVMSRecord = &ivms101.Person{
			Person: &ivms101.Person_NaturalPerson{
				NaturalPerson: &ivms101.NaturalPerson{
					Name: &ivms101.NaturalPersonName{
						NameIdentifiers: []*ivms101.NaturalPersonNameId{
							{
								PrimaryIdentifier:   "FirstName",
								SecondaryIdentifier: "LastName",
								NameIdentifierType:  ivms101.NaturalPersonNameTypeCode_NATURAL_PERSON_NAME_TYPE_CODE_LEGL,
							},
						},
					},
				},
			},
		}
	}

	if addCrypto {
		addresses := []*models.CryptoAddress{
			{
				AccountID:     id,
				CryptoAddress: "CryptoAddress1",
				Network:       "BTC",
			},
			{
				AccountID:     id,
				CryptoAddress: "CryptoAddress2",
				Network:       "BTC",
			},
		}
		account.SetCryptoAddresses(addresses)
	}

	return account
}

// Returns a sample CryptoAddress for the account ID provided. The crypto and
// travel addresses will be a random ULID strings and the network is "BTC".
func GetSampleCryptoAddress(accountId ulid.ULID) (account *models.CryptoAddress) {
	return &models.CryptoAddress{
		AccountID:     accountId,
		CryptoAddress: ulid.MakeSecure().String(),
		Network:       "BTC",
		TravelAddress: sql.NullString{String: ulid.MakeSecure().String(), Valid: true},
	}
}

// Returns a sample User. Can include or exclude any `NullType` types. `zeroID`
// returns the model with a zeroed ULID.
func GetSampleUser(includeNulls bool) (model *models.User) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.User{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		Email:     "email@example.com",
		Password:  "Password",
		RoleID:    1,
		LastLogin: sql.NullTime{},
	}

	if includeNulls {
		model.LastLogin = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}

// Returns a sample APIKey. Can include or exclude any `NullType` types. The
// client id and secret will be random ULID strings.
func GetSampleAPIKey(includeNulls bool) (model *models.APIKey) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.APIKey{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		Description: sql.NullString{},
		ClientID:    ulid.MakeSecure().String(),
		Secret:      ulid.MakeSecure().String(),
		LastSeen:    sql.NullTime{},
	}

	if includeNulls {
		model.Description = sql.NullString{String: "Description", Valid: true}
		model.LastSeen = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}

// Returns a sample Role. Can include sample Permissions with it.
func GetSampleRole(id int64, includePermissions bool) (model *models.Role) {
	timeNow := time.Now()

	model = &models.Role{
		ID:          id,
		Created:     timeNow,
		Modified:    timeNow,
		Title:       "Title",
		Description: "Description",
		IsDefault:   true,
	}

	if includePermissions {
		model.SetPermissions([]*models.Permission{GetSamplePermission(1), GetSamplePermission(2)})
	}

	return model
}

// Returns a sample Permission.
func GetSamplePermission(id int64) (model *models.Permission) {
	timeNow := time.Now()

	model = &models.Permission{
		ID:          id,
		Created:     timeNow,
		Modified:    timeNow,
		Title:       "Title",
		Description: "Description",
	}

	return model
}

// Returns a sample ResetPasswordLink. Can include or exclude any `NullType` types.
func GetSampleResetPasswordLink(includeNulls bool) (model *models.ResetPasswordLink) {
	id := ulid.MakeSecure()
	userid := ulid.MakeSecure()
	timeNow := time.Now()
	expiration := timeNow.Add(1 * time.Hour)

	token, err := vero.New(id.Bytes(), expiration)
	if err != nil {
		panic(err)
	}
	_, sig, err := token.Sign()
	if err != nil {
		panic(err)
	}

	model = &models.ResetPasswordLink{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		UserID:     userid,
		Email:      "email@example.com",
		Expiration: expiration,
		Signature:  sig,
		SentOn:     sql.NullTime{},
	}

	if includeNulls {
		model.SentOn = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}

// Returns a sample Counterparty. This counterparty will be unique, using it's
// ID for the CommonName and Endpoint ("ULID_STRING.sample.example.com").
func GetSampleCounterparty(includeNulls bool, includeContacts bool) (model *models.Counterparty) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.Counterparty{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		Source:              enum.SourceUserEntry,
		DirectoryID:         sql.NullString{},
		RegisteredDirectory: sql.NullString{},
		Protocol:            enum.ProtocolTRISA,
		CommonName:          fmt.Sprintf("%s.sample.example.com", id.String()),
		Endpoint:            fmt.Sprintf("https://%s.sample.example.com:808/api/v1", id.String()),
		Name:                "Sample Counterparty",
		Website:             sql.NullString{},
		Country:             sql.NullString{},
		BusinessCategory:    sql.NullString{},
		VASPCategories:      models.VASPCategories{},
		VerifiedOn:          sql.NullTime{},
		IVMSRecord:          nil,
		LEI:                 sql.NullString{},
	}

	if includeNulls {
		model.DirectoryID = sql.NullString{String: uuid.NewString(), Valid: true}
		model.RegisteredDirectory = sql.NullString{String: "RegisteredDirectory", Valid: true}
		model.Website = sql.NullString{String: "https://sample.example.com", Valid: true}
		model.Country = sql.NullString{String: "US", Valid: true}
		model.BusinessCategory = sql.NullString{String: "BusinessCategory", Valid: true}
		model.VASPCategories = models.VASPCategories{"Category One", "Category Two"}
		model.VerifiedOn = sql.NullTime{Time: timeNow, Valid: true}
		model.LEI = sql.NullString{String: id.String(), Valid: true}
	}

	if includeContacts {
		model.SetContacts([]*models.Contact{
			GetSampleContact("CA6DF33A-26D9-4E0E-B5F7-66D1964BE696@example.com"),
			GetSampleContact(""),
			GetSampleContact(""),
		})
	}

	return model
}

// Returns a sample Contact.
func GetSampleContact(email string) (model *models.Contact) {
	id := ulid.MakeSecure()
	timeNow := time.Now()
	if email == "" {
		email = fmt.Sprintf("%s@example.com", id.String())
	}

	model = &models.Contact{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		Name:           "Name",
		Email:          email,
		Role:           "Role",
		CounterpartyID: ulid.MakeSecure(),
	}

	return model
}

// Returns a sample Sunrise. It won't have a signature unless includeNulls is true,
// and the email will be unique on the message ID.
func GetSampleSunrise(includeNulls bool) (model *models.Sunrise) {
	id := ulid.MakeSecure()
	envId := uuid.New()
	timeNow := time.Now()

	model = &models.Sunrise{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		EnvelopeID: envId,
		Email:      id.String() + "@example.com",
		Expiration: timeNow.Add(1 * time.Hour),
		Signature:  nil,
		Status:     enum.StatusPending,
		SentOn:     sql.NullTime{},
		VerifiedOn: sql.NullTime{},
	}

	if includeNulls {
		veroToken, err := vero.New(id.Bytes(), timeNow.Add(1*time.Hour))
		if err != nil {
			panic(err)
		}
		_, sig, err := veroToken.Sign()
		if err != nil {
			panic(err)
		}
		model.Signature = sig
		model.SentOn = sql.NullTime{Time: timeNow, Valid: true}
		model.VerifiedOn = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}

// Returns a sample Transaction. The counterparty, beneficiary, originator, and
// crypto addresses are unique using the ID of the returned model.
func GetSampleTransaction(includeNulls bool, includeEnvelopes bool, archived bool) (model *models.Transaction) {
	id := uuid.New()
	timeNow := time.Now()

	model = &models.Transaction{
		ID:                 id,
		Source:             enum.SourceLocal,
		Status:             enum.StatusAccepted,
		Counterparty:       "Counterparty_" + id.String(),
		CounterpartyID:     ulid.NullULID{},
		Originator:         sql.NullString{},
		OriginatorAddress:  sql.NullString{},
		Beneficiary:        sql.NullString{},
		BeneficiaryAddress: sql.NullString{},
		VirtualAsset:       "BTC",
		Amount:             0.123456,
		Archived:           false,
		ArchivedOn:         sql.NullTime{},
		LastUpdate:         sql.NullTime{},
		Created:            timeNow,
		Modified:           timeNow,
	}

	if includeNulls {
		model.CounterpartyID = ulid.NullULID{ULID: ulid.MakeSecure(), Valid: true}
		model.Originator = sql.NullString{String: "Originator_" + id.String(), Valid: true}
		model.OriginatorAddress = sql.NullString{String: "O_" + id.String(), Valid: true}
		model.Beneficiary = sql.NullString{String: "Beneficiary_" + id.String(), Valid: true}
		model.BeneficiaryAddress = sql.NullString{String: "B_" + id.String(), Valid: true}
		model.LastUpdate = sql.NullTime{Time: timeNow, Valid: true}
	}

	if includeEnvelopes {
		env1 := GetSampleSecureEnvelope(true, false)
		env1.SetTransaction(model)
		env2 := GetSampleSecureEnvelope(true, false)
		env2.SetTransaction(model)
		model.SetSecureEnvelopes([]*models.SecureEnvelope{env1, env2})
	}

	if archived {
		model.Archived = true
		model.ArchivedOn = sql.NullTime{Time: timeNow, Valid: true}
	}

	return model
}

// Returns a sample SecureEnvelope.
func GetSampleSecureEnvelope(includeNulls bool, includeTransaction bool) (model *models.SecureEnvelope) {
	id := ulid.MakeSecure()
	timeNow := time.Now()

	model = &models.SecureEnvelope{
		Model: models.Model{
			ID:       id,
			Created:  timeNow,
			Modified: timeNow,
		},
		EnvelopeID:    uuid.New(),
		Direction:     enum.DirectionOutgoing,
		Remote:        sql.NullString{},
		ReplyTo:       ulid.NullULID{},
		IsError:       false,
		EncryptionKey: nil,
		HMACSecret:    nil,
		ValidHMAC:     sql.NullBool{},
		Timestamp:     timeNow,
		PublicKey:     sql.NullString{},
		TransferState: 1,
		Envelope:      nil,
	}

	if includeNulls {
		model.Remote = sql.NullString{String: "Remote", Valid: true}
		model.ReplyTo = ulid.NullULID{ULID: ulid.MakeSecure(), Valid: true}
		model.ValidHMAC = sql.NullBool{Bool: false, Valid: true}
		model.PublicKey = sql.NullString{String: "PK_" + id.String(), Valid: true}
	}

	if includeTransaction {
		txn := GetSampleTransaction(true, false, false)
		txn.SetSecureEnvelopes([]*models.SecureEnvelope{model})
		model.SetTransaction(txn)
	}

	return model
}

// Returns a sample ComplianceAuditLog. ChangeNotes is optional, as is signing it before returning.
func GetComplianceAuditLog(includeChangeNotes bool, signIt bool) (model *models.ComplianceAuditLog) {
	id := ulid.MakeSecure()
	timeNow := time.Now()
	resourceId, err := uuid.New().MarshalBinary()
	if err != nil {
		panic(err)
	}

	model = &models.ComplianceAuditLog{
		ID:               id,
		ActorID:          ulid.MakeSecure().Bytes(),
		ActorType:        enum.ActorUser,
		ResourceID:       resourceId,
		ResourceType:     enum.ResourceTransaction,
		ResourceModified: timeNow,
		Action:           enum.ActionCreate,
		ChangeNotes:      sql.NullString{},
	}

	if includeChangeNotes {
		model.ChangeNotes = sql.NullString{Valid: true, String: "ChangeNotes_" + ulid.MakeSecure().String()}
	}

	if signIt {
		// FIXME: fix the Sign call below, which is now in the Store
		// if err := model.Sign(); err != nil {
		// 	panic(err)
		// }
	}

	return model
}
