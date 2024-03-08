package web

import (
	"errors"
	"net/http"

	dberr "self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/ulids"
	api "self-hosted-node/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/oklog/ulid/v2"
)

func (s *Server) ListAccounts(c *gin.Context) {
	var (
		err   error
		in    *api.PageQuery
		query *models.PageInfo
		page  *models.AccountsPage
		out   *api.AccountsList
	)

	// Parse the URL parameters from the input request
	in = &api.PageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	// TODO: implement better pagination mechanism (with pagination tokens)

	// Fetch the list of accounts from the database
	if page, err = s.store.ListAccounts(c.Request.Context(), query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process account list request"))
		return
	}

	// Convert the accounts page into an accounts list object
	if out, err = api.NewAccountList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process account list request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/account_list.html",
	})
}

func (s *Server) CreateAccount(c *gin.Context) {
	var (
		err     error
		in      *api.Account
		account *models.Account
		out     *api.Account
	)

	// Parse the model from the POST request
	in = &api.Account{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse account data"))
		return
	}

	// TODO: validate the account input

	// Convert the API account request into a database model
	if account, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create the model in the database (which will update the pointer)
	if err = s.store.CreateAccount(c.Request.Context(), account); err != nil {
		// TODO: are there other error types that we need to handle to return a 400?
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewAccount(account); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/account_create.html",
	})
}

func (s *Server) AccountDetail(c *gin.Context) {
	var (
		err       error
		accountID ulid.ULID
		account   *models.Account
		out       *api.Account
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("account not found"))
		return
	}

	// Fetch the model from the database
	if account, err = s.store.RetrieveAccount(c.Request.Context(), accountID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model into an API response
	if out, err = api.NewAccount(account); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/account_detail.html",
	})
}

func (s *Server) UpdateAccount(c *gin.Context) {
	var (
		err       error
		accountID ulid.ULID
		in        *api.Account
		out       *api.Account
		account   *models.Account
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("account not found"))
		return
	}

	// Parse the account data PUT to the endpoint
	in = &api.Account{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse account data"))
		return
	}

	// Sanity check the account IDs of the update request
	if err = ulids.CheckIDMatch(in.ID, accountID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// TODO: validate the account input

	// Convert the API account request into a database model
	if account, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Update the model in the database (which will update the pointer).
	if err = s.store.UpdateAccount(c.Request.Context(), account); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		// TODO: are there other error types that we need to handle to return a 400?
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewAccount(account); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/account_update.html",
	})
}

func (s *Server) DeleteAccount(c *gin.Context) {
	var (
		err       error
		accountID ulid.ULID
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("account not found"))
		return
	}

	// Delete the account from the database
	if err = s.store.DeleteAccount(c.Request.Context(), accountID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLData: gin.H{"AccountID": accountID},
		JSONData: api.Reply{Success: true},
		HTMLName: "partials/account_delete.html",
	})
}

func (s *Server) ListCryptoAddresses(c *gin.Context) {
	var (
		err       error
		in        *api.PageQuery
		accountID ulid.ULID
		query     *models.PageInfo
		page      *models.CryptoAddressPage
		out       *api.CryptoAddressList
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("account not found"))
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

	// Fetch the list of crypto addresses from the database
	if page, err = s.store.ListCryptoAddresses(c.Request.Context(), accountID, query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process crypto address list request"))
		return
	}

	// Convert the crypto addresses page into a crypto addresses list object
	if out, err = api.NewCryptoAddressList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process crypto address list request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/crypto_address_list.html",
	})
}

func (s *Server) CreateCryptoAddress(c *gin.Context) {
	var (
		err       error
		in        *api.CryptoAddress
		accountID ulid.ULID
		model     *models.CryptoAddress
		out       *api.CryptoAddress
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("account not found"))
		return
	}

	// Parse the input from the POST request
	in = &api.CryptoAddress{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse crypto address data"))
		return
	}

	// TODO: validate the input

	// Convert the request into a database model
	if model, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Associate the model with the account
	model.AccountID = accountID

	// Create the model in the database
	if err = s.store.CreateCryptoAddress(c.Request.Context(), model); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		// TODO: handle constraint violations
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewCryptoAddress(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/crypto_address_create.html",
	})
}

func (s *Server) CryptoAddressDetail(c *gin.Context) {
	var (
		err             error
		accountID       ulid.ULID
		cryptoAddressID ulid.ULID
		model           *models.CryptoAddress
		out             *api.CryptoAddress
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("account not found"))
		return
	}

	// Parse the cryptoAddressID passed in from the URL
	if cryptoAddressID, err = ulid.Parse(c.Param("cryptoAddressID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("crypto address not found"))
		return
	}

	// Fetch the model from the database
	if model, err = s.store.RetrieveCryptoAddress(c.Request.Context(), accountID, cryptoAddressID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("crypto address or account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert model into an API response
	if out, err = api.NewCryptoAddress(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/crypto_address_detail.html",
	})
}

func (s *Server) UpdateCryptoAddress(c *gin.Context) {
	var (
		err             error
		accountID       ulid.ULID
		cryptoAddressID ulid.ULID
		in              *api.CryptoAddress
		model           *models.CryptoAddress
		out             *api.CryptoAddress
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("account not found"))
		return
	}

	// Parse the cryptoAddressID passed in from the URL
	if cryptoAddressID, err = ulid.Parse(c.Param("cryptoAddressID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("crypto address not found"))
		return
	}

	// Parse the crypto address data from the PUT request
	in = &api.CryptoAddress{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse crypto address data"))
		return
	}

	// Sanity check the IDs of the update request
	if err = ulids.CheckIDMatch(in.ID, cryptoAddressID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// TODO: validate the crypto address input

	// Convert the crypto address request into a database model
	if model, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Associate the account ID with the model
	model.AccountID = accountID

	// Update the model in the database (which will update the pointer).
	if err = s.store.UpdateCryptoAddress(c.Request.Context(), model); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("crypto address or account not found"))
			return
		}

		// TODO: are there other error types that we need to handle to return a 400?
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an api response
	if out, err = api.NewCryptoAddress(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/crypto_address_update.html",
	})
}

func (s *Server) DeleteCryptoAddress(c *gin.Context) {
	var (
		err             error
		accountID       ulid.ULID
		cryptoAddressID ulid.ULID
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("account not found"))
		return
	}

	// Parse the cryptoAddressID passed in from the URL
	if cryptoAddressID, err = ulid.Parse(c.Param("cryptoAddressID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("crypto address not found"))
		return
	}

	// Delete the crypto address from the database
	if err = s.store.DeleteCryptoAddress(c.Request.Context(), accountID, cryptoAddressID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, "crypto_address or account not found")
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLData: gin.H{"AccountID": accountID, "CryptoAddressID": cryptoAddressID},
		JSONData: api.Reply{Success: true},
		HTMLName: "partials/crypto_address_delete.html",
	})
}
