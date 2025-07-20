package web

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"

	"github.com/skip2/go-qrcode"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"github.com/trisacrypto/envoy/pkg/web/scene"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"go.rtnl.ai/ulid"
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
		HTMLName: "partials/accounts/list.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) CreateAccount(c *gin.Context) {
	var (
		err     error
		in      *api.Account
		query   *api.EncodingQuery
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
	if err = in.Validate(true); err != nil {
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Convert the API account request into a database model
	if account, err = in.Model(); err != nil {
		c.Error(fmt.Errorf("could not deserialize request into model: %w", err))
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create the model in the database (which will update the pointer)
	// NOTE: creating the account will also create an associated travel address
	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.CreateAccount(c.Request.Context(), account, &models.ComplianceAuditLog{}); err != nil {
		if errors.Is(err, dberr.ErrAlreadyExists) {
			c.JSON(http.StatusConflict, api.Error("account or crypto address already exists"))
			return
		}

		c.Error(fmt.Errorf("could not create account: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// If this is an HTMX request, redirect to the account detail page
	if htmx.IsHTMXRequest(c) {
		htmx.Redirect(c, http.StatusSeeOther, "/accounts/"+account.ID.String()+"/edit")
		return
	}

	// Otherwise, convert the model back to an API response
	if out, err = api.NewAccount(account, query); err != nil {
		c.Error(fmt.Errorf("serialization failed: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.JSON(http.StatusCreated, out)
}

func (s *Server) LookupAccount(c *gin.Context) {
	var (
		err     error
		query   *api.AccountLookupQuery
		account *models.Account
		out     *api.Account
	)

	query = &api.AccountLookupQuery{}
	if err = c.BindQuery(query); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse account lookup query"))
		return
	}

	if err = query.Validate(); err != nil {
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Fetch the model from the database
	if account, err = s.store.LookupAccount(c.Request.Context(), query.CryptoAddress); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model into an API response
	if out, err = api.NewAccount(account, &query.EncodingQuery); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Content negotiation
	ctx := scene.New(c).WithAPIData(out)
	ctx["Prefix"] = query.Prefix

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/accounts/lookup.html",
		HTMLData: ctx,
	})
}

func (s *Server) AccountDetail(c *gin.Context) {
	var (
		err     error
		query   *api.EncodingQuery
		account *models.Account
		out     *api.Account
	)

	// Parse the query parameters from the input request
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

	// Retrieve the account from the database
	if account, err = s.RetrieveAccount(c); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not retrieve account"))
		return
	}

	// Convert the model into an API response
	if out, err = api.NewAccount(account, query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Currently there are no HTMX endpoints that use the account detail, so this is
	// a JSON only endpoint that does not need a partial template for rendering.
	c.JSON(http.StatusOK, out)
}

// Helper function to retrieve an account detail for the accounts UI pages and the API.
func (s *Server) RetrieveAccount(c *gin.Context) (*models.Account, error) {
	var (
		err       error
		accountID ulid.ULID
		account   *models.Account
	)

	// Parse the accountID passed in from the URL
	if accountID, err = ulid.Parse(c.Param("id")); err != nil {
		return nil, ErrNotFound
	}

	// Fetch the model from the database
	if account, err = s.store.RetrieveAccount(c.Request.Context(), accountID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	return account, nil
}

func (s *Server) UpdateAccount(c *gin.Context) {
	var (
		err       error
		accountID ulid.ULID
		query     *api.EncodingQuery
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

	// Sanity check the account IDs of the update request
	if err = CheckIDMatch(in.ID, accountID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	in.SetEncoding(query)
	if err = in.Validate(false); err != nil {
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Convert the API account request into a database model
	if account, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Update the model in the database (which will update the pointer).
	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.UpdateAccount(c.Request.Context(), account, &models.ComplianceAuditLog{}); err != nil {
		switch {
		case errors.Is(err, dberr.ErrNotFound):
			c.JSON(http.StatusNotFound, api.Error("account not found"))
		case errors.Is(err, dberr.ErrAlreadyExists):
			c.JSON(http.StatusConflict, api.Error("account or crypto address already exists"))
		default:
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error(err))
		}
		return
	}

	// If this is an HTMX request, trigger the accounts updated event and return a 204.
	if htmx.IsHTMXRequest(c) {
		htmx.Trigger(c, htmx.AccountsUpdated)
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewAccount(account, query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Render the response back as JSON
	c.JSON(http.StatusOK, out)
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
	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.DeleteAccount(c.Request.Context(), accountID, &models.ComplianceAuditLog{}); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if htmx.IsHTMXRequest(c) {
		htmx.Trigger(c, htmx.AccountsUpdated)
		return
	}

	c.JSON(http.StatusOK, api.Reply{Success: true})
}

func (s *Server) AccountTransfers(c *gin.Context) {
	var (
		err     error
		in      *api.TransactionListQuery
		account *models.Account
		page    *models.TransactionPage
		out     *api.TransactionsList
	)

	// Parse the URL parameters from the input request
	in = &api.TransactionListQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	// Validate the incoming parameters from the query
	if err = in.Validate(); err != nil {
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Retrieve the account from the database
	if account, err = s.RetrieveAccount(c); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not retrieve account"))
		return
	}

	// Fetch the list of transactions from the database
	if page, err = s.store.ListAccountTransactions(c.Request.Context(), account.ID, in.Query()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process account transaction list request"))
		return
	}

	// Convert the transactions page into a transaction list object
	if out, err = api.NewTransactionList(page); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process transaction list request"))
		return
	}

	// Content negotiation
	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		JSONData: out,
		HTMLData: scene.New(c).WithAPIData(out),
		HTMLName: "partials/accounts/transfers.html",
	})
}

func (s *Server) AccountQRCode(c *gin.Context) {
	var (
		err     error
		account *models.Account
		qrc     *qrcode.QRCode
	)

	// Retrieve the account from the database
	if account, err = s.RetrieveAccount(c); err != nil {
		if errors.Is(err, ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not retrieve account"))
		return
	}

	if !account.TravelAddress.Valid || account.TravelAddress.String == "" {
		c.JSON(http.StatusNotFound, api.Error("account does not have a travel address"))
		return
	}

	if qrc, err = qrcode.New(account.TravelAddress.String, qrcode.Medium); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not generate QR code"))
		return
	}

	buf := new(bytes.Buffer)
	if err = qrc.Write(512, buf); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not write QR code"))
		return
	}

	filename := account.ID.String() + ".png"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "image/png", buf.Bytes())
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
		HTMLData: scene.New(c).WithAPIData(out).WithParent(accountID),
		HTMLName: "partials/accounts/cryptoAddresses.html",
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

	// Associate the model with the account
	model.AccountID = accountID

	// Create the model in the database
	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.CreateCryptoAddress(c.Request.Context(), model, &models.ComplianceAuditLog{}); err != nil {
		if errors.Is(err, dberr.ErrAlreadyExists) {
			c.JSON(http.StatusConflict, api.Error("crypto address already exists"))
			return
		}

		c.Error(fmt.Errorf("could not create crypto address: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// If this is an HTMX request, trigger the crypto addresses updated event
	if htmx.IsHTMXRequest(c) {
		htmx.Trigger(c, htmx.CryptoAddressesUpdated)
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewCryptoAddress(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.JSON(http.StatusCreated, out)
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
		JSONData: out,
		HTMLData: scene.New(c).WithAPIData(out).WithParent(accountID),
		HTMLName: "partials/accounts/cryptoAddressDetail.html",
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
	if err = CheckIDMatch(in.ID, cryptoAddressID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	if err = in.Validate(false); err != nil {
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Convert the crypto address request into a database model
	if model, err = in.Model(nil); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Associate the account ID with the model
	model.AccountID = accountID

	// Update the model in the database (which will update the pointer).
	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.UpdateCryptoAddress(c.Request.Context(), model, &models.ComplianceAuditLog{}); err != nil {
		switch {
		case errors.Is(err, dberr.ErrNotFound):
			c.JSON(http.StatusNotFound, api.Error("crypto address or account not found"))
		case errors.Is(err, dberr.ErrAlreadyExists):
			c.JSON(http.StatusConflict, api.Error("crypto address already exists"))
		default:
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error(err))
		}

		return
	}

	// If this is an HTMX request, trigger the crypto addresses updated event
	if htmx.IsHTMXRequest(c) {
		htmx.Trigger(c, htmx.CryptoAddressesUpdated)
		return
	}

	// Convert the model back to an api response
	if out, err = api.NewCryptoAddress(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.JSON(http.StatusOK, out)
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
	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.DeleteCryptoAddress(c.Request.Context(), accountID, cryptoAddressID, &models.ComplianceAuditLog{}); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, "crypto_address or account not found")
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// If this is an HTMX request, trigger the crypto addresses updated event
	if htmx.IsHTMXRequest(c) {
		htmx.Trigger(c, htmx.CryptoAddressesUpdated)
		return
	}

	c.JSON(http.StatusOK, api.Reply{Success: true})
}

func (s *Server) CryptoAddressQRCode(c *gin.Context) {
	var (
		err             error
		accountID       ulid.ULID
		cryptoAddressID ulid.ULID
		model           *models.CryptoAddress
		qrc             *qrcode.QRCode
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

	if !model.TravelAddress.Valid || model.TravelAddress.String == "" {
		c.JSON(http.StatusNotFound, api.Error("account does not have a travel address"))
		return
	}

	if qrc, err = qrcode.New(model.TravelAddress.String, qrcode.Medium); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not generate QR code"))
		return
	}

	buf := new(bytes.Buffer)
	if err = qrc.Write(512, buf); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not write QR code"))
		return
	}

	filename := model.ID.String() + ".png"
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "image/png", buf.Bytes())
}
