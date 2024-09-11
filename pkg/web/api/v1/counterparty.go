package api

import (
	"database/sql"
	"net/url"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/store/models"

	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

//===========================================================================
// Counterparty Resource
//===========================================================================

type Counterparty struct {
	ID                  ulid.ULID      `json:"id,omitempty"`
	Source              string         `json:"source,omitempty"`
	DirectoryID         string         `json:"directory_id,omitempty"`
	RegisteredDirectory string         `json:"registered_directory,omitempty"`
	Protocol            string         `json:"protocol"`
	CommonName          string         `json:"common_name,omitempty"`
	Endpoint            string         `json:"endpoint"`
	TravelAddress       string         `json:"travel_address,omitempty"`
	Name                string         `json:"name"`
	Website             string         `json:"website,omitempty"`
	Country             string         `json:"country"`
	BusinessCategory    string         `json:"business_category,omitempty"`
	VASPCategories      []string       `json:"vasp_categories,omitempty"`
	VerifiedOn          time.Time      `json:"verified_on,omitempty"`
	IVMSRecord          string         `json:"ivms101,omitempty"`
	Created             time.Time      `json:"created,omitempty"`
	Modified            time.Time      `json:"modified,omitempty"`
	encoding            *EncodingQuery `json:"-"`
}

type CounterpartyList struct {
	Page           *PageQuery      `json:"page"`
	Counterparties []*Counterparty `json:"counterparties"`
}

func NewCounterparty(model *models.Counterparty, encoding *EncodingQuery) (out *Counterparty, err error) {
	if encoding == nil {
		encoding = &EncodingQuery{}
	}

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
		encoding:            encoding,
	}

	// Render the IVMS101 data as as base64 encoded JSON string
	if model.IVMSRecord != nil {
		if out.IVMSRecord, err = out.encoding.Marshal(model.IVMSRecord); err != nil {
			// Log the error but do not stop processing
			log.Error().Err(err).
				Str("account_id", model.ID.String()).
				Str("encoding", encoding.Encoding).
				Str("format", encoding.Format).
				Bool("is_base64_std", encoding.b64std).
				Msg("could not marshal IVMS101 record")
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
		if counterparty, err = NewCounterparty(model, nil); err != nil {
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

	if c.encoding == nil {
		c.encoding = &EncodingQuery{}
	}

	p = &ivms101.LegalPerson{}
	if err = c.encoding.Unmarshal(c.IVMSRecord, p); err != nil {
		return nil, err
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
		switch e := perr.(type) {
		case ivms101.ValidationErrors:
			for _, ve := range e {
				err = ValidationError(err, InvalidIVMS101(ve))
			}
		case *ivms101.FieldError:
			err = ValidationError(err, InvalidIVMS101(e))
		case ValidationErrors:
			err = ValidationError(err, e...)
		case *FieldError:
			err = ValidationError(err, e)
		default:
			err = ValidationError(err, IncorrectField("ivms101", perr.Error()))
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

func (c *Counterparty) SetEncoding(encoding *EncodingQuery) {
	if encoding == nil {
		encoding = &EncodingQuery{}
	}
	c.encoding = encoding
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
