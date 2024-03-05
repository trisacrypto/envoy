package web

import (
	"encoding/json"
	"errors"
	"net/http"

	dberr "self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/ulids"
	api "self-hosted-node/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/ivms101"
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

	// TODO: convert the page query into page info
	// TODO: implement better pagination mechanism (with pagination tokens)

	// Fetch the list of accounts from the database
	if page, err = s.store.ListAccounts(query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process account list request"))
		return
	}

	// Convert the accounts page into an accounts list object
	out = &api.AccountsList{
		Page:     &api.PageQuery{},
		Accounts: make([]*api.Account, 0, len(page.Accounts)),
	}

	for _, model := range page.Accounts {
		var account *api.Account
		if account, err = AccountFromModel(model); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not process account list request"))
			return
		}

		out.Accounts = append(out.Accounts, account)
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
	if account, err = ModelFromAccount(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Create the model in the database (which will update the pointer)
	if err = s.store.CreateAccount(account); err != nil {
		// TODO: are there other error types that we need to handle to return a 400?
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = AccountFromModel(account); err != nil {
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
	if account, err = s.store.RetrieveAccount(accountID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("account not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model into an API response
	if out, err = AccountFromModel(account); err != nil {
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
	if account, err = ModelFromAccount(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Update the model in the database (which will update the pointer).
	if err = s.store.UpdateAccount(account); err != nil {
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
	if out, err = AccountFromModel(account); err != nil {
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
	if err = s.store.DeleteAccount(accountID); err != nil {
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

func AccountFromModel(account *models.Account) (out *api.Account, err error) {
	out = &api.Account{
		ID:            account.ID,
		CustomerID:    account.CustomerID,
		FirstName:     account.FirstName,
		LastName:      account.LastName,
		TravelAddress: account.TravelAddress,
		Created:       account.Created,
		Modified:      account.Modified,
	}

	// Render the IVMS101 data as as JSON string
	if account.IVMSRecord != nil {
		if data, err := json.Marshal(account.IVMSRecord); err != nil {
			// Log the error but do not stop processing
			log.Error().Err(err).Str("account_id", account.ID.String()).Msg("could not marshal IVMS101 record to JSON")
		} else {
			out.IVMSRecord = string(data)
		}
	}

	// Collect the crypto address associations
	var addresses []*models.CryptoAddress
	if addresses, err = account.CryptoAddresses(); err != nil {
		return nil, err
	}

	// Add the crypto addresses to the response
	out.CryptoAdddresses = make([]*api.CryptoAddress, 0, len(addresses))
	for _, address := range addresses {
		out.CryptoAdddresses = append(out.CryptoAdddresses, &api.CryptoAddress{
			ID:            address.ID,
			CryptoAddress: address.CryptoAddress,
			Network:       address.Network,
			AssetType:     address.AssetType,
			Tag:           address.Tag,
			Created:       address.Created,
			Modified:      address.Modified,
		})
	}

	return out, nil
}

func ModelFromAccount(in *api.Account) (account *models.Account, err error) {
	account = &models.Account{
		Model: models.Model{
			ID:       in.ID,
			Created:  in.Created,
			Modified: in.Modified,
		},
		CustomerID:    in.CustomerID,
		FirstName:     in.FirstName,
		LastName:      in.LastName,
		TravelAddress: in.TravelAddress,
		IVMSRecord:    nil,
	}

	if in.IVMSRecord != "" {
		account.IVMSRecord = &ivms101.Person{}
		if err = json.Unmarshal([]byte(in.IVMSRecord), account.IVMSRecord); err != nil {
			return nil, err
		}
	}

	if len(in.CryptoAdddresses) > 0 {
		addresses := make([]*models.CryptoAddress, 0, len(in.CryptoAdddresses))
		for _, address := range in.CryptoAdddresses {
			addresses = append(addresses, &models.CryptoAddress{
				Model: models.Model{
					ID:       address.ID,
					Created:  address.Created,
					Modified: address.Modified,
				},
				CryptoAddress: address.CryptoAddress,
				Network:       address.Network,
				AssetType:     address.AssetType,
				Tag:           address.Tag,
			})
		}

		account.SetCryptoAddresses(addresses)
	}

	return account, nil
}
