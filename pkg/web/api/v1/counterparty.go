package api

import (
	"database/sql"
	"encoding/json"
	"net/url"
	"self-hosted-node/pkg/store/models"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

//===========================================================================
// Counterparty Resource
//===========================================================================

type Counterparty struct {
	ID                  ulid.ULID `json:"id,omitempty"`
	Source              string    `json:"source,omitempty"`
	DirectoryID         string    `json:"directory_id,omitempty"`
	RegisteredDirectory string    `json:"registered_directory,omitempty"`
	Protocol            string    `json:"protocol"`
	CommonName          string    `json:"common_name"`
	Endpoint            string    `json:"endpoint"`
	Name                string    `json:"name"`
	Website             string    `json:"website"`
	Country             string    `json:"country"`
	BusinessCategory    string    `json:"business_category"`
	VASPCategories      []string  `json:"vasp_categories"`
	VerifiedOn          time.Time `json:"verified_on"`
	IVMSRecord          string    `json:"ivms101,omitempty"`
	Created             time.Time `json:"created,omitempty"`
	Modified            time.Time `json:"modified,omitempty"`
}

type CounterpartyList struct {
	Page           *PageQuery      `json:"page"`
	Counterparties []*Counterparty `json:"counterparties"`
}

func NewCounterparty(model *models.Counterparty) (out *Counterparty, err error) {
	out = &Counterparty{
		ID:                  model.ID,
		Source:              model.Source,
		DirectoryID:         model.DirectoryID.String,
		RegisteredDirectory: model.RegisteredDirectory.String,
		Protocol:            model.Protocol,
		CommonName:          model.CommonName,
		Endpoint:            model.Endpoint,
		Name:                model.Name,
		Website:             model.Website.String,
		Country:             model.Country,
		BusinessCategory:    model.BusinessCategory.String,
		VASPCategories:      model.VASPCategories,
		VerifiedOn:          model.VerifiedOn.Time,
		Created:             model.Created,
		Modified:            model.Modified,
	}

	if model.IVMSRecord != nil {
		if data, err := json.Marshal(model.IVMSRecord); err != nil {
			// Log the error but do not stop processing
			log.Error().Err(err).Str("counterparty_id", model.ID.String()).Msg("could not marshal IVMS101 record to JSON")
		} else {
			out.IVMSRecord = string(data)
		}
	}

	return out, nil
}

func NewCounterpartyList(page *models.CounterpartyPage) (out *CounterpartyList, err error) {
	out = &CounterpartyList{
		Page:           &PageQuery{},
		Counterparties: make([]*Counterparty, 0, len(page.Counterparties)),
	}

	for _, model := range page.Counterparties {
		var counterparty *Counterparty
		if counterparty, err = NewCounterparty(model); err != nil {
			return nil, err
		}
		out.Counterparties = append(out.Counterparties, counterparty)
	}

	return out, nil
}

func (c *Counterparty) Validate() (err error) {
	if c.Source != "" {
		err = ValidationError(err, ReadOnlyField("source"))
	}

	if c.DirectoryID != "" {
		err = ValidationError(err, ReadOnlyField("directory_id"))
	}

	if c.RegisteredDirectory != "" {
		err = ValidationError(err, ReadOnlyField("registered_directory"))
	}

	c.Protocol = strings.TrimSpace(strings.ToLower(c.Protocol))
	if c.Protocol == "" {
		err = ValidationError(err, MissingField("protocol"))
	} else {
		if c.Protocol != "trisa" && c.Protocol != "trp" {
			err = ValidationError(err, IncorrectField("protocol", "protocol must be either trisa or trp"))
		}
	}

	if c.CommonName == "" {
		// Set common name to the hostname endpoint if not supplied by default
		if c.Endpoint != "" {
			if u, err := url.Parse(c.Endpoint); err == nil {
				c.CommonName = u.Hostname()
			}
		}

		// If no common name still exists (e.g. endpoint is missing or not parseable)
		// then return a missing field error
		if c.CommonName == "" {
			err = ValidationError(err, MissingField("common_name"))
		}
	}

	if c.Endpoint == "" {
		err = ValidationError(err, MissingField("endpoint"))
	}

	if c.Name == "" {
		err = ValidationError(err, MissingField("name"))
	}

	c.Country = strings.TrimSpace(strings.ToUpper(c.Country))
	if c.Country == "" {
		err = ValidationError(err, MissingField("country"))
	} else {
		if len(c.Country) != 2 {
			err = ValidationError(err, IncorrectField("country", "country must be the two character (alpha-2) country code"))
		}
	}

	return err
}

func (c *Counterparty) Model() (model *models.Counterparty, err error) {
	model = &models.Counterparty{
		Model: models.Model{
			ID:       c.ID,
			Created:  c.Created,
			Modified: c.Modified,
		},
		Source:              c.Source,
		DirectoryID:         sql.NullString{String: c.DirectoryID, Valid: c.DirectoryID != ""},
		RegisteredDirectory: sql.NullString{String: c.RegisteredDirectory, Valid: c.RegisteredDirectory != ""},
		Protocol:            c.Protocol,
		CommonName:          c.CommonName,
		Endpoint:            c.Endpoint,
		Name:                c.Name,
		Website:             sql.NullString{String: c.Website, Valid: c.Website != ""},
		Country:             c.Country,
		BusinessCategory:    sql.NullString{String: c.BusinessCategory, Valid: c.BusinessCategory != ""},
		VASPCategories:      models.VASPCategories(c.VASPCategories),
		VerifiedOn:          sql.NullTime{Time: c.VerifiedOn, Valid: !c.VerifiedOn.IsZero()},
		IVMSRecord:          nil,
	}

	if c.IVMSRecord != "" {
		model.IVMSRecord = &ivms101.LegalPerson{}
		if err = json.Unmarshal([]byte(c.IVMSRecord), model.IVMSRecord); err != nil {
			return nil, err
		}
	}

	return model, nil
}
