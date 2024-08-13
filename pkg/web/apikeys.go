package web

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/oklog/ulid/v2"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth/passwords"
	"github.com/trisacrypto/envoy/pkg/web/scene"
)

func (s *Server) ListAPIKeys(c *gin.Context) {
	var (
		err   error
		in    *api.PageQuery
		query *models.PageInfo
		page  *models.APIKeyPage
		out   *api.APIKeyList
	)

	// Parse the URL parameters from the input request
	in = &api.PageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	// TODO: implement better pagination mechanism

	// Fetch the list of api keys from the database
	if page, err = s.store.ListAPIKeys(c.Request.Context(), query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikeys list request"))
		return
	}

	// Convert the users page into a users list object
	if out, err = api.NewAPIKeyList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikeys list request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "apikey_list.html",
	})
}

func (s *Server) CreateAPIKey(c *gin.Context) {
	var (
		err    error
		in     *api.APIKey
		apikey *models.APIKey
		secret string
		out    *api.APIKey
	)

	// Parse the model from the POST request
	in = &api.APIKey{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse apikey data"))
		return
	}

	// Validate an API key to be created
	// NOTE: this also validates the permissions using the permissions package in auth
	if err = in.Validate(true); err != nil {
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Convert the API serializer into a database model
	if apikey, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create a client ID for the api key
	apikey.ClientID = passwords.KeyID()

	// Create a secret and the derived key of that secret for the api key
	secret = passwords.Secret()
	if apikey.Secret, err = passwords.CreateDerivedKey(secret); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}

	if err = s.store.CreateAPIKey(c.Request.Context(), apikey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewAPIKey(apikey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process create apikey request"))
		return
	}

	// Ensure the created apikey secret is returned back to the user
	out.Secret = secret

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "apikey_create.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) APIKeyDetail(c *gin.Context) {
	var (
		err    error
		keyID  ulid.ULID
		apikey *models.APIKey
		out    *api.APIKey
	)

	// Parse the keyID from the URL
	if keyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("apikey not found"))
		return
	}

	// Fetch the model from the database
	if apikey, err = s.store.RetrieveAPIKey(c.Request.Context(), keyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("apikey not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process apikey detail request"))
		return
	}

	if out, err = api.NewAPIKey(apikey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to process apikey detail request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "apikey_detail.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) UpdateAPIKey(c *gin.Context) {
	var (
		err    error
		keyID  ulid.ULID
		apikey *models.APIKey
		in     *api.APIKey
		out    *api.APIKey
	)

	// Parse the keyID from the URL
	if keyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("apikey not found"))
		return
	}

	// Parse the apikey data for the update request
	in = &api.APIKey{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse apikey data"))
		return
	}

	// Sanity check
	if err = ulids.CheckIDMatch(in.ID, keyID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Validation in update mode (e.g. create=false)
	if err = in.Validate(false); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create the model to be updated
	if apikey, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Update the APIKey in the database
	if err = s.store.UpdateAPIKey(c.Request.Context(), apikey); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("apikey not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikey update request"))
		return
	}

	// Convert model back to an API response
	if out, err = api.NewAPIKey(apikey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikey update request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "apikey_update.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) DeleteAPIKey(c *gin.Context) {
	var (
		err   error
		keyID ulid.ULID
	)

	// Parse the keyID from the URL
	if keyID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("apikey not found"))
		return
	}

	// Delete the API key from the database
	// TODO: for audit purposes we may simply want to move the API key to a revoked table.
	if err = s.store.DeleteAPIKey(c.Request.Context(), keyID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("apikey not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process apikey revoke request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     api.Reply{Success: true},
		HTMLName: "apikey_detail.html",
		HTMLData: scene.Scene{"KeyID": keyID},
	})
}
