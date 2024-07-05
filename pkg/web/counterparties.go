package web

import (
	"errors"
	"net/http"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/oklog/ulid/v2"
)

// ListCounterparties - Paginated list of all stored counterparties
//
//	@Summary		List counterparties
//	@Description	Paginated list of all stored counterparties
//	@ID				listCounterparties
//	@Tags			Counterparty
//	@Security		BearerAuth
//	@Produce		json
//	@Param			page	query		api.PageQuery			true	"Page query parameters"
//	@Success		200		{object}	api.CounterpartyList	"Successful operation"
//	@Failure		400		{object}	api.Reply				"Invalid input"
//	@Failure		401		{object}	api.Reply				"Unauthorized"
//	@Failure		500		{object}	api.Reply				"Internal server error"
//	@Router			/v1/counterparties [get]
func (s *Server) ListCounterparties(c *gin.Context) {
	var (
		err   error
		in    *api.PageQuery
		query *models.PageInfo
		page  *models.CounterpartyPage
		out   *api.CounterpartyList
	)

	// Parse the URL parameters from the input request
	in = &api.PageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	// TODO: implement better pagination mechanism (with pagination tokens)

	if page, err = s.store.ListCounterparties(c.Request.Context(), query); err != nil {
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
		HTMLName: "counterparty_list.html",
	})
}

// CreateCounterparty - Create a new counterparty
//
//	@Summary		Create counterparty
//	@Description	Create a new counterparty
//	@ID				createCounterparty
//	@Tags			Counterparty
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			counterparty	body		api.Counterparty	true	"Create a new counterparty"
//	@Success		201				{object}	api.Counterparty
//	@Failure		400				{object}	api.Reply	"Invalid input"
//	@Failure		401				{object}	api.Reply	"Unauthorized"
//	@Failure		422				{object}	api.Reply	"Validation exception or missing field"
//	@Failure		500				{object}	api.Reply	"Internal server error"
//	@Router			/v1/counterparties [post]
func (s *Server) CreateCounterparty(c *gin.Context) {
	var (
		err          error
		in           *api.Counterparty
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

	// Validate the counterparty input
	if !ulids.IsZero(in.ID) {
		c.JSON(http.StatusBadRequest, api.Error("cannot specify an id when creating a counterparty"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Set the source after validation
	in.Source = models.SourceUserEntry

	// Covert the API serializer into a database model
	if counterparty, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create the model in the database (which will update the pointer)
	if err = s.store.CreateCounterparty(c.Request.Context(), counterparty); err != nil {
		// TODO: are there other error types we need to handle to return a 400?
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewCounterparty(counterparty); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "counterparty_create.html",
	})
}

// CounterpartyDetail - Returns a single counterparty if found
//
//	@Summary		Find counterparty by ID
//	@Description	Returns a single counterparty if found
//	@ID				counterpartyDetail
//	@Tags			Counterparty
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"ID of counterparty to return"
//	@Success		200	{object}	api.Counterparty
//	@Failure		401	{object}	api.Reply	"Unauthorized"
//	@Failure		404	{object}	api.Reply	"Counterparty not found"
//	@Failure		500	{object}	api.Reply	"Internal server error"
//	@Router			/v1/counterparties/{counterpartyID} [get]
func (s *Server) CounterpartyDetail(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		counterparty   *models.Counterparty
		out            *api.Counterparty
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
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

	if out, err = api.NewCounterparty(counterparty); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "counterparty_detail.html",
	})
}

func (s *Server) UpdateCounterpartyPreview(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		counterparty   *models.Counterparty
		out            *api.Counterparty
	)

	// Parse the counterpartyID passed in from the URL
	if counterpartyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("counterparty not found"))
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

	if out, err = api.NewCounterparty(counterparty); err != nil {
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

// UpdateCounterparty - Update a counterparty record (does not patch, all fields are required)
//
//	@Summary		Updates a counterparty record
//	@Description	Update a counterparty record (does not patch, all fields are required)
//	@ID				updateCounterparty
//	@Tags			Counterparty
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string				true	"ID of counterparty to update"
//	@Param			counterparty	body		api.Counterparty	true	"Updated counterparty record"
//	@Success		200				{object}	api.Counterparty
//	@Failure		400				{object}	api.Reply	"Invalid input"
//	@Failure		401				{object}	api.Reply	"Unauthorized"
//	@Failure		403				{object}	api.Reply	"Forbidden"
//	@Failure		404				{object}	api.Reply	"Counterparty not found"
//	@Failure		422				{object}	api.Reply	"Validation exception or missing field"
//	@Failure		500				{object}	api.Reply	"Internal server error"
//	@Router			/v1/counterparties/{counterpartyID} [put]
func (s *Server) UpdateCounterparty(c *gin.Context) {
	var (
		err            error
		counterpartyID ulid.ULID
		counterparty   *models.Counterparty
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

	// Sanity check
	if err = ulids.CheckIDMatch(in.ID, counterpartyID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Validation
	if in.Source != "" && in.Source != models.SourceUserEntry {
		c.JSON(http.StatusForbidden, api.Error("this record cannot be edited"))
		return
	}

	// Blank source for validation purposes
	in.Source = ""
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Replace source to update database
	in.Source = models.SourceUserEntry

	if counterparty, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
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
	if out, err = api.NewCounterparty(counterparty); err != nil {
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

// DeleteCounterparty - Deletes a counterparty
//
//	@Summary		Deletes a counterparty
//	@Description	Deletes a counterparty
//	@ID				deleteCounterparty
//	@Tags			Counterparty
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"ID of counterparty to delete"
//	@Success		200	{object}	api.Reply
//	@Failure		401	{object}	api.Reply	"Unauthorized"
//	@Failure		403	{object}	api.Reply	"Forbidden"
//	@Failure		404	{object}	api.Reply	"Counterparty not found"
//	@Failure		500	{object}	api.Reply	"Internal server error"
//	@Router			/v1/counterparties/{counterpartyID} [delete]
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
	if counterparty.Source != "" && counterparty.Source != models.SourceUserEntry {
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
		Data:     gin.H{"CounterpartyID": counterpartyID, "Source": counterparty.Source},
		HTMLName: "counterparty_delete.html",
	})
}
