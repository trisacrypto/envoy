package api

import (
	"database/sql"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/envoy/pkg/store/models"
)

type APIKey struct {
	ID          ulid.ULID  `json:"id,omitempty"`
	Description string     `json:"description"`
	ClientID    string     `json:"client_id"`
	Secret      string     `json:"client_secret,omitempty"`
	LastSeen    *time.Time `json:"last_seen"`
	Permissions []string   `json:"permissions,omitempty"`
	Created     time.Time  `json:"created,omitempty"`
	Modified    time.Time  `json:"modified,omitempty"`
}

type APIKeyList struct {
	Page    *PageQuery `json:"page"`
	APIKeys []*APIKey  `json:"api_keys"`
}

func NewAPIKey(model *models.APIKey) (out *APIKey, err error) {
	out = &APIKey{
		ID:          model.ID,
		Description: model.Description.String,
		ClientID:    model.ClientID,
		Secret:      model.Secret, // Should be zero-valued except on create and therefore omitted from the payload in most cases
		Permissions: model.Permissions(),
		Created:     model.Created,
		Modified:    model.Modified,
	}

	if model.LastSeen.Valid {
		out.LastSeen = &model.LastSeen.Time
	}

	return out, nil
}

func NewAPIKeyList(page *models.APIKeyPage) (out *APIKeyList, err error) {
	out = &APIKeyList{
		Page: &PageQuery{
			PageSize: int(page.Page.PageSize),
		},
		APIKeys: make([]*APIKey, 0, len(page.APIKeys)),
	}

	for _, model := range page.APIKeys {
		var key *APIKey
		if key, err = NewAPIKey(model); err != nil {
			return nil, err
		}
		out.APIKeys = append(out.APIKeys, key)
	}

	return out, nil
}

func (k *APIKey) Validate(create bool) (err error) {
	if k.ClientID != "" {
		err = ValidationError(err, ReadOnlyField("client_id"))
	}

	if k.Secret != "" {
		err = ValidationError(err, ReadOnlyField("client_secret"))
	}

	if k.LastSeen != nil {
		err = ValidationError(err, ReadOnlyField("last_seen"))
	}

	// Permissions should be zero on update, but non-zero on create
	if create {
		if len(k.Permissions) == 0 {
			err = ValidationError(err, MissingField("permissions"))
		}
	} else {
		if len(k.Permissions) > 0 {
			err = ValidationError(err, ReadOnlyField("permissions"))
		}
	}

	return err
}

func (k *APIKey) Model() (model *models.APIKey, err error) {
	model = &models.APIKey{
		Model: models.Model{
			ID:       k.ID,
			Created:  k.Created,
			Modified: k.Modified,
		},
		Description: sql.NullString{String: k.Description, Valid: k.Description != ""},
		ClientID:    k.ClientID,
	}

	if k.LastSeen != nil {
		model.LastSeen = sql.NullTime{
			Time:  *k.LastSeen,
			Valid: true,
		}
	}

	if len(k.Permissions) > 0 {
		model.SetPermissions(k.Permissions)
	}

	return model, nil
}
