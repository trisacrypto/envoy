package web

import (
	"encoding/json"
	"errors"
	"net/http"

	dberr "self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"
	api "self-hosted-node/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/oklog/ulid/v2"
	"github.com/rs/zerolog/log"
)

func (s *Server) ListAccounts(c *gin.Context) {

}

func (s *Server) CreateAccount(c *gin.Context) {}

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

func (s *Server) UpdateAccount(c *gin.Context) {}

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

	return account, nil
}
