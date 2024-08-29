package api

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"
	"google.golang.org/protobuf/proto"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
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
	CommonName          string    `json:"common_name,omitempty"`
	Endpoint            string    `json:"endpoint"`
	TravelAddress       string    `json:"travel_address,omitempty"`
	Name                string    `json:"name"`
	Website             string    `json:"website,omitempty"`
	Country             string    `json:"country"`
	BusinessCategory    string    `json:"business_category,omitempty"`
	VASPCategories      []string  `json:"vasp_categories,omitempty"`
	VerifiedOn          time.Time `json:"verified_on,omitempty"`
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
		Country:             model.Country.String,
		BusinessCategory:    model.BusinessCategory.String,
		VASPCategories:      model.VASPCategories,
		VerifiedOn:          model.VerifiedOn.Time,
		Created:             model.Created,
		Modified:            model.Modified,
	}

	// Render the IVMS101 data as as base64 encoded JSON string
	// TODO: select rendering using protocol buffers or JSON as a config option.
	if model.IVMSRecord != nil {
		if data, err := json.Marshal(model.IVMSRecord); err != nil {
			// Log the error but do not stop processing
			log.Error().Err(err).Str("counterparty_id", model.ID.String()).Msg("could not marshal IVMS101 record to JSON")
		} else {
			out.IVMSRecord = base64.URLEncoding.EncodeToString(data)
		}
	}

	// Compute the travel address from the endpoint (ignore errors)
	out.TravelAddress, _ = EndpointTravelAddress(model.Endpoint, model.Protocol)

	return out, nil
}

func NewCounterpartyList(page *models.CounterpartyPage) (out *CounterpartyList, err error) {
	out = &CounterpartyList{
		Page: &PageQuery{
			PageSize: int(page.Page.PageSize),
		},
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

func (c *Counterparty) IVMS101() (p *ivms101.LegalPerson, err error) {
	// Don't handle empty strings.
	if c.IVMSRecord == "" {
		return nil, ErrParsingIVMS101Person
	}

	// Try decoding URL base64 first, then STD before resorting to a string
	var data []byte
	if data, err = base64.URLEncoding.DecodeString(c.IVMSRecord); err != nil {
		if data, err = base64.StdEncoding.DecodeString(c.IVMSRecord); err != nil {
			data = []byte(c.IVMSRecord)
		}
	}

	// Try unmarshaling JSON first, then protocol buffers
	p = &ivms101.LegalPerson{}
	if err = json.Unmarshal(data, p); err != nil {
		if err = proto.Unmarshal(data, p); err != nil {
			return nil, ErrParsingIVMS101Person
		}
	}

	return p, nil
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

	if _, perr := c.IVMS101(); perr != nil {
		err = ValidationError(err, IncorrectField("ivms101", perr.Error()))
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
		Country:             sql.NullString{String: c.Country, Valid: c.Country != ""},
		BusinessCategory:    sql.NullString{String: c.BusinessCategory, Valid: c.BusinessCategory != ""},
		VASPCategories:      models.VASPCategories(c.VASPCategories),
		VerifiedOn:          sql.NullTime{Time: c.VerifiedOn, Valid: !c.VerifiedOn.IsZero()},
		IVMSRecord:          nil,
	}

	if c.IVMSRecord != "" {
		if model.IVMSRecord, err = c.IVMS101(); err != nil {
			return nil, err
		}
	}

	return model, nil
}

func EndpointTravelAddress(endpoint, protocol string) (string, error) {
	params := make(url.Values)
	params.Set("t", "i")
	if protocol != "" {
		params.Set("mode", protocol)
	}

	uri := &url.URL{Host: endpoint, RawQuery: params.Encode()}
	return traddr.Encode(strings.TrimPrefix(uri.String(), "//"))
}
