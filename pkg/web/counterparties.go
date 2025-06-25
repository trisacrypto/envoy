package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/scene"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/ulid"
)

const defaultSearchLimit = 10

func (s *Server) SearchCounterparties(c *gin.Context) {
	var (
		err  error
		in   *api.SearchQuery
		page *models.CounterpartyPage
		out  *api.CounterpartyList
	)

	// Parse the URL parameters from the input request
	in = &api.SearchQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse counterparties search request"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Set the default value on the limit
	if in.Limit == 0 {
		in.Limit = defaultSearchLimit
	}

	if page, err = s.store.SearchCounterparties(c.Request.Context(), in.Model()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process counterparty list request"))
		return
	}

	// Convert the counterparties page into an api response
	if out, err = api.NewCounterpartyList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process counterparty list request"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) ListCounterparties(c *gin.Context) {
	var (
		err  error
		in   *api.CounterpartyQuery
		page *models.CounterpartyPage
		out  *api.CounterpartyList
	)

	// Parse the URL parameters from the input request
	in = &api.CounterpartyQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	if err = in.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// TODO: implement better pagination mechanism (with pagination tokens)

	if page, err = s.store.ListCounterparties(c.Request.Context(), in.Query()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process counterparty list request"))
		return
	}

	// Convert the counterparties page into an api response
	if out, err = api.NewCounterpartyList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process counterparty list request"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/counterparties/list.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) CreateCounterparty(c *gin.Context) {
	var (
		err          error
		in           *api.Counterparty
		query        *api.EncodingQuery
		counterparty *models.Counterparty
		out          *api.Counterparty
	)

	// Parse the model from the POST reqeust
	in = &api.Counterparty{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse counterparty data"))
		return
	}

	query = &api.EncodingQuery{}
	if err = c.BindQuery(query); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse encoding query"))
		return
	}

	if err = query.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	in.SetEncoding(query)
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Validate the counterparty input
	if !in.ID.IsZero() {
		c.JSON(http.StatusBadRequest, api.Error("cannot specify an id when creating a counterparty"))
		return
	}

	// Covert the API serializer into a database model
	if counterparty, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// The source of the counterparty created by the API is user entry
	counterparty.Source = enum.SourceUserEntry

	// ensure the website has a protocol (default to `https://`)
	if counterparty.Website.Valid {
		if !strings.Contains(counterparty.Website.String, "://") {
			counterparty.Website.String = fmt.Sprintf("https://%s", counterparty.Website.String)
		}
		parsed, err := url.Parse(counterparty.Website.String)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error(err))
			return
		}
		counterparty.Website.String = parsed.String()
	}

	// Create the model in the database (which will update the pointer)
	if err = s.store.CreateCounterparty(c.Request.Context(), counterparty); err != nil {
		// TODO: are there other error types we need to handle to return a 400?
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewCounterparty(counterparty, query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusCreated, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "counterparty_create.html",
	})
}

func (s *Server) CounterpartyDetail(c *gin.Context) {
	var (
		err            error
		query          *api.EncodingQuery
		counterpartyID ulid.ULID
		counterparty   *models.Counterparty
		out            *api.Counterparty
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	query = &api.EncodingQuery{}
	if err = c.BindQuery(query); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse encoding query"))
		return
	}

	if err = query.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Fetch the model from the database
	if counterparty, err = s.store.RetrieveCounterparty(c.Request.Context(), counterpartyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model into an API response
	if out, err = api.NewCounterparty(counterparty, query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/counterparties/detail.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) UpdateCounterpartyPreview(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		query          *api.EncodingQuery
		counterparty   *models.Counterparty
		out            *api.Counterparty
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	query = &api.EncodingQuery{}
	if err = c.BindQuery(query); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse encoding query"))
		return
	}

	if err = query.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Fetch the model from the database
	if counterparty, err = s.store.RetrieveCounterparty(c.Request.Context(), counterpartyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if out, err = api.NewCounterparty(counterparty, query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "counterparty_preview.html",
	})
}

func (s *Server) UpdateCounterparty(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		original       *models.Counterparty
		counterparty   *models.Counterparty
		query          *api.EncodingQuery
		in             *api.Counterparty
		out            *api.Counterparty
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	// Parse the counterparty data to PUT to the endpoint
	in = &api.Counterparty{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse counterparty data"))
		return
	}

	query = &api.EncodingQuery{}
	if err = c.BindQuery(query); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse encoding query"))
		return
	}

	if err = query.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Sanity check
	if err = CheckIDMatch(in.ID, counterpartyID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Ensure that only user sourced entries can be edited.
	if ok, _ := enum.CheckSource(in.Source, enum.SourceUnknown, enum.SourceUserEntry); !ok {
		c.JSON(http.StatusConflict, api.Error("only user entered records can be updated"))
		return
	}

	in.SetEncoding(query)
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	if counterparty, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Ensure the user is not trying to overwrite an entity created by another source
	if original, err = s.store.RetrieveCounterparty(c.Request.Context(), counterpartyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if counterparty.Source != enum.SourceUnknown && original.Source != counterparty.Source {
		if original.Source != enum.SourceUserEntry {
			c.JSON(http.StatusConflict, api.Error("only user created records can be edited"))
		} else {
			c.JSON(http.StatusConflict, api.Error("the source of a counterparty record cannot be changed"))
		}
		return
	}

	// Ensure that the source is always set to the original source for API updates
	// (e.g. overwrite source unknown)
	counterparty.Source = original.Source

	// ensure the website has a protocol (default to `https://`)
	if counterparty.Website.Valid {
		if !strings.Contains(counterparty.Website.String, "://") {
			counterparty.Website.String = fmt.Sprintf("https://%s", counterparty.Website.String)
		}
		parsed, err := url.Parse(counterparty.Website.String)
		if err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error(err))
			return
		}
		counterparty.Website.String = parsed.String()
	}

	if err = s.store.UpdateCounterparty(c.Request.Context(), counterparty); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
			return
		}

		// TODO: handle other types of dberrors and constraints
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert model back to an api response
	if out, err = api.NewCounterparty(counterparty, query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "counterparty_update.html",
	})
}

func (s *Server) DeleteCounterparty(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		counterparty   *models.Counterparty
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	// Retrieve the counterparty to validate the source
	if counterparty, err = s.store.RetrieveCounterparty(c.Request.Context(), counterpartyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Check the source to make sure it is a user record
	if ok, _ := enum.CheckSource(counterparty.Source, enum.SourceUnknown, enum.SourceUserEntry); !ok {
		c.JSON(http.StatusForbidden, api.Error("this record cannot be edited"))
		return
	}

	if err = s.store.DeleteCounterparty(c.Request.Context(), counterpartyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     scene.Scene{"CounterpartyID": counterpartyID, "Source": counterparty.Source},
		HTMLName: "counterparty_delete.html",
	})
}

func (s *Server) ListContacts(c *gin.Context) {
	var (
		err            error
		in             *api.PageQuery
		counterpartyID ulid.ULID
		query          *models.PageInfo
		page           *models.ContactsPage
		out            *api.ContactList
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	// Parse the URL parameters from the input request
	in = &api.PageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	// TODO: implement better pagination mechanism (with pagination tokens)

	// Fetch the list of contacts from the database
	if page, err = s.store.ListContacts(c.Request.Context(), counterpartyID, query); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process contacts list request"))
		return
	}

	// Convert the page into a contact list object
	if out, err = api.NewContactList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process contacts list request"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "contact_list.html",
	})
}

func (s *Server) CreateContact(c *gin.Context) {
	var (
		err            error
		in             *api.Contact
		counterpartyID ulid.ULID
		model          *models.Contact
		out            *api.Contact
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	// Parse the input from the POST request
	in = &api.Contact{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse contact data"))
		return
	}

	if err = in.Validate(true); err != nil {
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Convert the request into a database model
	if model, err = in.Model(nil); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Associate the model with the counterparty
	model.CounterpartyID = counterpartyID

	// Create the model in the database
	if err = s.store.CreateContact(c.Request.Context(), model); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
			return
		}

		// TODO: handle constraint violations
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewContact(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusCreated, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "contact_create.html",
	})
}

func (s *Server) ContactDetail(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		contactID      ulid.ULID
		model          *models.Contact
		out            *api.Contact
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	// Parse the contactID passed in from the URL
	if contactID, err = ulid.Parse(c.Param("contactID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("contact not found"))
		return
	}

	// Fetch the model from the database
	if model, err = s.store.RetrieveContact(c.Request.Context(), contactID, counterpartyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("contact or counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert model into an API response
	if out, err = api.NewContact(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "contact_detail.html",
	})
}

func (s *Server) UpdateContact(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		contactID      ulid.ULID
		in             *api.Contact
		model          *models.Contact
		out            *api.Contact
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	// Parse the contactID passed in from the URL
	if contactID, err = ulid.Parse(c.Param("contactID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("contact not found"))
		return
	}

	// Parse contact data from the PUT request
	in = &api.Contact{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not parse contact data"))
		return
	}

	// Sanity check the IDs of the update request
	if err = CheckIDMatch(in.ID, contactID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if err = in.Validate(false); err != nil {
		c.JSON(http.StatusUnsupportedMediaType, api.Error(err))
		return
	}

	// Convert the contact request into a database model
	if model, err = in.Model(nil); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Associate the counterprty ID with the model
	model.CounterpartyID = counterpartyID

	// Update the model in the database (which will update the pointer).
	if err = s.store.UpdateContact(c.Request.Context(), model); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("contact or counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an api response
	if out, err = api.NewContact(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "contact_update.html",
	})
}

func (s *Server) DeleteContact(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		contactID      ulid.ULID
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
		return
	}

	// Parse the contactID passed in from the URL
	if contactID, err = ulid.Parse(c.Param("contactID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("contact not found"))
		return
	}

	// Delete the contact from the database
	if err = s.store.DeleteContact(c.Request.Context(), contactID, counterpartyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("contact or counterparty not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLData: scene.Scene{"CounterpartyID": counterpartyID, "ContactID": contactID},
		JSONData: api.Reply{Success: true},
		HTMLName: "contact_delete.html",
	})
}
