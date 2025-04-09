package api

import (
	"database/sql"
	"errors"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
	"go.rtnl.ai/ulid"
)

//===========================================================================
// Counterparty Resource
//===========================================================================

type CounterpartyQuery struct {
	PageQuery
	Source string `json:"source,omitempty" url:"source,omitempty" form:"source"`
}

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
	VerifiedOn          *time.Time     `json:"verified_on,omitempty"`
	IVMSRecord          string         `json:"ivms101,omitempty"`
	Contacts            []*Contact     `json:"contacts,omitempty"`
	Created             time.Time      `json:"created,omitempty"`
	Modified            *time.Time     `json:"modified,omitempty"`
	encoding            *EncodingQuery `json:"-"`
}

type Contact struct {
	ID       ulid.ULID `json:"id,omitempty"`
	Name     string    `json:"name"`
	Email    string    `json:"email"`
	Role     string    `json:"role"`
	Created  time.Time `json:"created,omitempty"`
	Modified time.Time `json:"modified,omitempty"`
}

type CounterpartyList struct {
	Page           *CounterpartyQuery `json:"page"`
	Counterparties []*Counterparty    `json:"counterparties"`
}

type ContactList struct {
	Page     *PageQuery `json:"page"`
	Contacts []*Contact `json:"contacts"`
}

func NewCounterparty(model *models.Counterparty, encoding *EncodingQuery) (out *Counterparty, err error) {
	if encoding == nil {
		encoding = &EncodingQuery{}
	}

	out = &Counterparty{
		ID:                  model.ID,
		Source:              model.Source.String(),
		DirectoryID:         model.DirectoryID.String,
		RegisteredDirectory: model.RegisteredDirectory.String,
		Protocol:            model.Protocol.String(),
		CommonName:          model.CommonName,
		Endpoint:            model.Endpoint,
		Name:                model.Name,
		Website:             model.Website.String,
		Country:             model.Country.String,
		BusinessCategory:    model.BusinessCategory.String,
		VASPCategories:      model.VASPCategories,
		Created:             model.Created,
		encoding:            encoding,
	}

	if model.VerifiedOn.Valid {
		out.VerifiedOn = &model.VerifiedOn.Time
	}

	if !model.Modified.IsZero() {
		out.Modified = &model.Modified
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

	// Collect the contact associations
	var contacts []*models.Contact
	if contacts, err = model.Contacts(); err != nil {
		if !errors.Is(err, dberr.ErrMissingAssociation) {
			return nil, err
		}
	}

	// Add the contacts to the response
	out.Contacts = make([]*Contact, 0, len(contacts))
	for _, contact := range contacts {
		c, _ := NewContact(contact)
		out.Contacts = append(out.Contacts, c)
	}

	// Compute the travel address from the endpoint (ignore errors)
	out.TravelAddress, _ = EndpointTravelAddress(model.Endpoint, model.Protocol)
	return out, nil
}

func NewCounterpartyList(page *models.CounterpartyPage) (out *CounterpartyList, err error) {
	out = &CounterpartyList{
		Page: &CounterpartyQuery{
			PageQuery: PageQuery{
				PageSize: int(page.Page.PageSize),
			},
			Source: page.Page.Source,
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
	if c.Source != "" && !enum.ValidSource(c.Source) {
		err = ValidationError(err, ReadOnlyField("source"))
	}

	if c.DirectoryID != "" {
		err = ValidationError(err, ReadOnlyField("directory_id"))
	}

	if c.RegisteredDirectory != "" {
		err = ValidationError(err, ReadOnlyField("registered_directory"))
	}

	if protocol, perr := enum.ParseProtocol(c.Protocol); perr != nil {
		err = ValidationError(err, IncorrectField("protocol", "invalid protocol, use trisa, trp, or sunrise"))
	} else if protocol == enum.ProtocolUnknown {
		err = ValidationError(err, MissingField("protocol"))
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

	if c.IVMSRecord != "" {
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
	}

	return err
}

func (c *Counterparty) Model() (model *models.Counterparty, err error) {
	model = &models.Counterparty{
		Model: models.Model{
			ID:      c.ID,
			Created: c.Created,
		},
		DirectoryID:         sql.NullString{String: c.DirectoryID, Valid: c.DirectoryID != ""},
		RegisteredDirectory: sql.NullString{String: c.RegisteredDirectory, Valid: c.RegisteredDirectory != ""},
		CommonName:          c.CommonName,
		Endpoint:            c.Endpoint,
		Name:                c.Name,
		Website:             sql.NullString{String: c.Website, Valid: c.Website != ""},
		Country:             sql.NullString{String: c.Country, Valid: c.Country != ""},
		BusinessCategory:    sql.NullString{String: c.BusinessCategory, Valid: c.BusinessCategory != ""},
		VASPCategories:      models.VASPCategories(c.VASPCategories),
		IVMSRecord:          nil,
	}

	if model.Source, err = enum.ParseSource(c.Source); err != nil {
		return nil, err
	}

	if model.Protocol, err = enum.ParseProtocol(c.Protocol); err != nil {
		return nil, err
	}

	if c.Modified != nil {
		model.Modified = *c.Modified
	}

	if c.VerifiedOn != nil && !c.VerifiedOn.IsZero() {
		model.VerifiedOn = sql.NullTime{Time: *c.VerifiedOn, Valid: true}
	}

	if c.IVMSRecord != "" {
		if model.IVMSRecord, err = c.IVMS101(); err != nil {
			return nil, err
		}
	}

	if len(c.Contacts) > 0 {
		contacts := make([]*models.Contact, 0, len(c.Contacts))
		for _, contact := range c.Contacts {
			cm, _ := contact.Model(model)
			contacts = append(contacts, cm)
		}

		model.SetContacts(contacts)
	}

	return model, nil
}

func (c *Counterparty) SetEncoding(encoding *EncodingQuery) {
	if encoding == nil {
		encoding = &EncodingQuery{}
	}
	c.encoding = encoding
}

func NewContact(model *models.Contact) (*Contact, error) {
	return &Contact{
		ID:       model.ID,
		Name:     model.Name,
		Email:    model.Email,
		Role:     model.Role,
		Created:  model.Created,
		Modified: model.Modified,
	}, nil
}

func NewContactList(page *models.ContactsPage) (out *ContactList, err error) {
	out = &ContactList{
		Page:     &PageQuery{},
		Contacts: make([]*Contact, 0, len(page.Contacts)),
	}

	for _, model := range page.Contacts {
		var contact *Contact
		if contact, err = NewContact(model); err != nil {
			return nil, err
		}
		out.Contacts = append(out.Contacts, contact)
	}

	return out, nil
}

func (c *Contact) Model(counterparty *models.Counterparty) (*models.Contact, error) {
	contact := &models.Contact{
		Model: models.Model{
			ID:       c.ID,
			Created:  c.Created,
			Modified: c.Modified,
		},
		Name:  c.Name,
		Email: c.Email,
		Role:  c.Role,
	}

	if counterparty != nil {
		contact.CounterpartyID = counterparty.ID
		contact.SetCounterparty(counterparty)
	}
	return contact, nil
}

var emailre = regexp.MustCompile("^[a-zA-Z0-9.!#$%&'*+/=?^_`{|}~-]+@[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?(?:\\.[a-zA-Z0-9](?:[a-zA-Z0-9-]{0,61}[a-zA-Z0-9])?)*$")

func (c *Contact) Validate(create bool) (err error) {
	if create {
		if !c.ID.IsZero() {
			err = ValidationError(err, ReadOnlyField("id"))
		}
	}

	c.Name = strings.TrimSpace(c.Name)
	c.Role = strings.TrimSpace(c.Role)

	c.Email = strings.ToLower(strings.TrimSpace(c.Email))
	if c.Email == "" {
		err = ValidationError(err, MissingField("email"))
	} else if !emailre.MatchString(c.Email) {
		err = ValidationError(err, IncorrectField("email", "not an email address"))
	}

	return err
}

//===========================================================================
// Counterparty Query Methods
//===========================================================================

func (c *CounterpartyQuery) Validate() (err error) {
	if c.Source != "" {
		if ok, _ := enum.CheckSource(c.Source, enum.SourceUnknown, enum.SourceDirectorySync, enum.SourceUserEntry); !ok {
			err = ValidationError(err, IncorrectField("source", "must be one of gds or user"))
		}
	}
	return err
}

func (c *CounterpartyQuery) Query() (query *models.CounterpartyPageInfo) {
	query = &models.CounterpartyPageInfo{
		PageInfo: models.PageInfo{
			PageSize: uint32(c.PageSize),
		},
		Source: c.Source,
	}
	return query
}

//===========================================================================
// Helper Functions
//===========================================================================

func EndpointTravelAddress(endpoint string, protocol enum.Protocol) (string, error) {
	// Cannot generate a travel address for a sunrise Counterparty
	if protocol == enum.ProtocolSunrise {
		return "", nil
	}

	params := make(url.Values)
	params.Set("t", "i")
	if protocol != enum.ProtocolUnknown {
		params.Set("mode", protocol.String())
	}

	uri := &url.URL{Host: endpoint, RawQuery: params.Encode()}
	return traddr.Encode(strings.TrimPrefix(uri.String(), "//"))
}
