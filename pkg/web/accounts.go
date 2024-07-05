package web

import (
	"errors"
	"fmt"
	"net/http"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/oklog/ulid/v2"
)

// ListAccounts - Paginated list of all stored customer accounts
//
//	@Summary		List customer accounts
//	@Description	Paginated list of all stored customer accounts
//	@ID				listAccounts
//	@Tags			Account
//	@Security		BearerAuth
//	@Produce		json
//	@Param			page	query		api.PageQuery	true	"Page query parameters"
//	@Success		200		{object}	api.AccountsList
//	@Failure		400		{object}	api.Reply	"Invalid input"
//	@Failure		401		{object}	api.Reply	"Unauthorized"
//	@Failure		500		{object}	api.Reply	"Internal server error"
//	@Router			/v1/accounts [get]
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
		HTMLName: "account_list.html",
	})
}

// CreateAccount - Create a new customer account
//
//	@Summary		Create customer account
//	@Description	Create a new customer account
//	@ID				createAccount
//	@Tags			Account
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			account	body		api.Account	true	"Create a new customer account"
//	@Success		201		{object}	api.Account
//	@Failure		400		{object}	api.Reply	"Invalid input"
//	@Failure		401		{object}	api.Reply	"Unauthorized"
//	@Failure		422		{object}	api.Reply	"Validation exception or missing field"
//	@Failure		500		{object}	api.Reply	"Internal server error"
//	@Router			/v1/accounts [post]
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
		c.Error(fmt.Errorf("could not deserialize request into model: %w", err))
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create the model in the database (which will update the pointer)
	// NOTE: creating the account will also create an associated travel address
	if err = s.store.CreateAccount(c.Request.Context(), account); err != nil {
		// TODO: are there other error types that we need to handle to return a 400?
		c.Error(fmt.Errorf("could not create account: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewAccount(account); err != nil {
		c.Error(fmt.Errorf("serialization failed: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "account_create.html",
	})
}

// AccountDetail - Returns a single account if found
//
//	@Summary		Find account by ID
//	@Description	Returns a single account if found
//	@ID				accountDetail
//	@Tags			Account
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"ID of account to return"
//	@Success		200	{object}	api.Account
//	@Failure		401	{object}	api.Reply	"Unauthorized"
//	@Failure		404	{object}	api.Reply	"Account not found"
//	@Failure		500	{object}	api.Reply	"Internal server error"
//	@Router			/v1/account/{accountID} [get]
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
		HTMLName: "account_detail.html",
	})
}

// UpdateAccountPreview - Returns a preview of the updated account
//
//	@Summary		Preview account update
//	@Description	Returns a preview of the updated account
//	@ID				updateAccountPreview
//	@Tags			Account
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"ID of account to preview update for"
//	@Success		200	{object}	api.Account
//	@Failure		401	{object}	api.Reply	"Unauthorized"
//	@Failure		404	{object}	api.Reply	"Account not found"
//	@Failure		500	{object}	api.Reply	"Internal server error"
//	@Router			/v1/account/{accountID}/preview [get]
func (s *Server) UpdateAccountPreview(c *gin.Context) {
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
		HTMLName: "account_preview.html",
	})
}

// UpdateAccount - Update an account record (does not patch, all fields are required)
//
//	@Summary		Updates an account record
//	@Description	Update an account record (does not patch, all fields are required)
//	@ID				updateAccount
//	@Tags			Account
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id		path		string		true	"ID of account to update"
//	@Param			account	body		api.Account	true	"Updated account record"
//	@Success		200		{object}	api.Account
//	@Failure		400		{object}	api.Reply	"Invalid input"
//	@Failure		401		{object}	api.Reply	"Unauthorized"
//	@Failure		404		{object}	api.Reply	"Account not found"
//	@Failure		422		{object}	api.Reply	"Validation exception or missing field"
//	@Failure		500		{object}	api.Reply	"Internal server error"
//	@Router			/v1/account/{accountID} [put]
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
		c.JSON(http.StatusInternalServerError, api.Error(err))
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
		HTMLName: "account_update.html",
	})
}

// DeleteAccount - Deletes an account and associated crypto addresses
//
//	@Summary		Deletes an account
//	@Description	Deletes an account and associated crypto addresses
//	@ID				deleteAccount
//	@Tags			Account
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id	path		string	true	"ID of account to delete"
//	@Success		200	{object}	api.Reply
//	@Failure		401	{object}	api.Reply	"Unauthorized"
//	@Failure		404	{object}	api.Reply	"Account not found"
//	@Failure		500	{object}	api.Reply	"Internal server error"
//	@Router			/v1/account/{accountID} [delete]
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
		HTMLName: "account_delete.html",
	})
}

// ListCryptoAddresses - Returns a paginated list of all crypto addresses associated with the account
//
//	@Summary		List crypto addresses for account
//	@Description	Returns a paginated list of all crypto addresses associated with the account
//	@ID				listCryptoAddresses
//	@Tags			CryptoAddress
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id		path		string			true	"ID of account to return crypto addresses for"
//	@Param			page	query		api.PageQuery	true	"Page query parameters"
//	@Success		200		{object}	api.CryptoAddressList
//	@Failure		401		{object}	api.Reply	"Unauthorized"
//	@Failure		404		{object}	api.Reply	"Account not found"
//	@Failure		500		{object}	api.Reply	"Internal server error"
//	@Router			/v1/accounts/{accountID}/crypto-addresses [get]
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
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

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
		HTMLName: "crypto_address_list.html",
	})
}

// CreateCryptoAddress - Create a crypto address associated with the specified account
//
//	@Summary		Create crypto address
//	@Description	Create a crypto address associated with the specified account
//	@ID				createCryptoAddress
//	@Tags			CryptoAddress
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string				true	"ID of account to create crypto address for"
//	@Param			cryptoAddress	body		api.CryptoAddress	true	"Crypto address to create"
//	@Success		201				{object}	api.CryptoAddress
//	@Failure		400				{object}	api.Reply	"Invalid input"
//	@Failure		401				{object}	api.Reply	"Unauthorized"
//	@Failure		404				{object}	api.Reply	"Account not found"
//	@Failure		422				{object}	api.Reply	"Validation exception or missing field"
//	@Failure		500				{object}	api.Reply	"Internal server error"
//	@Router			/v1/accounts/{accountID}/crypto-addresses [post]
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
	if model, err = in.Model(nil); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Associate the model with the account
	model.AccountID = accountID

	// Create the model in the database
	// NOTE: creating the account will also create an associated travel address
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
		HTMLName: "crypto_address_create.html",
	})
}

// CryptoAddressDetail - Returns detailed information about the specified crypto address
//
//	@Summary		Lookup a specific crypto address
//	@Description	Returns detailed information about the specified crypto address
//	@ID				cryptoAddressDetail
//	@Tags			CryptoAddress
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id				path		string	true	"ID of account of crypto address to be returned"
//	@Param			cryptoAddressID	path		string	true	"ID of crypto address to return"
//	@Success		200				{object}	api.CryptoAddress
//	@Failure		401				{object}	api.Reply	"Unauthorized"
//	@Failure		404				{object}	api.Reply	"Account or crypto address not found"
//	@Failure		500				{object}	api.Reply	"Internal server error"
//	@Router			/v1/accounts/{accountID}/crypto-addresses/{cryptoAddressID} [get]
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
		HTMLName: "crypto_address_detail.html",
	})
}

// UpdateCryptoAddress - Update a crypto address record (does not patch, all fields are required)
//
//	@Summary		Update a crypto address
//	@Description	Update a crypto address record (does not patch, all fields are required)
//	@ID				updateCryptoAddress
//	@Tags			CryptoAddress
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			id				path		string				true	"ID of account of crypto address to be updated"
//	@Param			cryptoAddressID	path		string				true	"ID of crypto address to update"
//	@Param			cryptoAddress	body		api.CryptoAddress	true	"Updated crypto address record"
//	@Success		200				{object}	api.CryptoAddress
//	@Failure		400				{object}	api.Reply	"Invalid input"
//	@Failure		401				{object}	api.Reply	"Unauthorized"
//	@Failure		404				{object}	api.Reply	"Account or crypto address not found"
//	@Failure		422				{object}	api.Reply	"Validation exception or missing field"
//	@Failure		500				{object}	api.Reply	"Internal server error"
//	@Router			/v1/accounts/{accountID}/crypto-addresses/{cryptoAddressID} [put]
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
	if model, err = in.Model(nil); err != nil {
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
		HTMLName: "crypto_address_update.html",
	})
}

// DeleteCryptoAddress - Delete a crypto address record associated with account
//
//	@Summary		Delete a crypto address
//	@Description	Delete a crypto address record associated with account
//	@ID				deleteCryptoAddress
//	@Tags			CryptoAddress
//	@Security		BearerAuth
//	@Produce		json
//	@Param			id				path		string	true	"ID of account of crypto address to be deleted"
//	@Param			cryptoAddressID	path		string	true	"ID of crypto address to delete"
//	@Success		200				{object}	api.Reply
//	@Failure		401				{object}	api.Reply	"Unauthorized"
//	@Failure		404				{object}	api.Reply	"Account or crypto address not found"
//	@Failure		500				{object}	api.Reply	"Internal server error"
//	@Router			/v1/accounts/{accountID}/crypto-addresses/{cryptoAddressID} [delete]
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
		HTMLName: "crypto_address_delete.html",
	})
}
