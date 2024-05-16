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
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

func (s *Server) ListTransactions(c *gin.Context) {
	var (
		err   error
		in    *api.PageQuery
		query *models.PageInfo
		page  *models.TransactionPage
		out   *api.TransactionsList
	)

	// Parse the URL parameters from the input request
	in = &api.PageQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse page query request"))
		return
	}

	// TODO: implement better pagination mechanism (with pagination tokens)

	// Fetch the list of transactions from the database
	if page, err = s.store.ListTransactions(c.Request.Context(), query); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process transaction list request"))
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
		Data:     out,
		HTMLName: "transaction_list.html",
	})
}

func (s *Server) CreateTransaction(c *gin.Context) {
	var (
		err         error
		in          *api.Transaction
		transaction *models.Transaction
		out         *api.Transaction
	)

	in = &api.Transaction{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse transaction data"))
		return
	}

	// If the transaction is created by the API, it is considered local.
	in.Source = models.SourceLocal

	// Mark the transaction as a draft until a secure envelope is sent.
	in.Status = models.StatusDraft

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Convert the transaction request into a database model
	if transaction, err = in.Model(); err != nil {
		c.Error(fmt.Errorf("could not deserialize request into model: %w", err))
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	if err = s.store.CreateTransaction(c.Request.Context(), transaction); err != nil {
		// TODO: handle other error types and constraint violations
		c.Error(fmt.Errorf("could not create transaction: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewTransaction(transaction); err != nil {
		c.Error(fmt.Errorf("serialization failed: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_create.html",
	})
}

func (s *Server) TransactionDetail(c *gin.Context) {
	var (
		err           error
		transactionID uuid.UUID
		transaction   *models.Transaction
		out           *api.Transaction
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	if transaction, err = s.store.RetrieveTransaction(c.Request.Context(), transactionID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if out, err = api.NewTransaction(transaction); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_detail.html",
	})
}

func (s *Server) AcceptTransactionPreview(c *gin.Context) {
	var (
		err           error
		transactionID uuid.UUID
		transaction   *models.Transaction
		out           *api.Transaction
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	if transaction, err = s.store.RetrieveTransaction(c.Request.Context(), transactionID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if out, err = api.NewTransaction(transaction); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_accept.html",
	})
}

func (s *Server) UpdateTransaction(c *gin.Context) {
	var (
		err           error
		transactionID uuid.UUID
		transaction   *models.Transaction
		in            *api.Transaction
		out           *api.Transaction
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the transaction data to be updated
	in = &api.Transaction{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse transaction data"))
		return
	}

	// Sanity check the IDs match
	if err = CheckUUIDMatch(in.ID, transactionID); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Validate the transaction input
	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Convert the input transaction into a database model
	if transaction, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Update the model in the database (which will update the pointer).
	if err = s.store.UpdateTransaction(c.Request.Context(), transaction); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		// TODO: are there other error types that we need to handle to return a 400?
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Convert the model back to an API response
	if out, err = api.NewTransaction(transaction); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_update.html",
	})
}

func (s *Server) DeleteTransaction(c *gin.Context) {
	var (
		err           error
		transactionID uuid.UUID
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	if err = s.store.DeleteTransaction(c.Request.Context(), transactionID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLData: gin.H{"TransactionID": transactionID},
		JSONData: api.Reply{Success: true},
		HTMLName: "transaction_delete.html",
	})
}

func (s *Server) ListSecureEnvelopes(c *gin.Context) {
	var (
		err           error
		in            *api.EnvelopeListQuery
		transactionID uuid.UUID
		page          *models.SecureEnvelopePage
		out           *api.EnvelopesList
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the URL parameters from the request
	in = &api.EnvelopeListQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse envelope list query request"))
		return
	}

	// TODO: implement better pagination mechanism (with pagination tokens)

	// Fetch the list of secure envelopes for the specified transactionID
	if page, err = s.store.ListSecureEnvelopes(c.Request.Context(), transactionID, nil); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process secure envelopes list request"))
		return
	}

	// TODO: implement decryption!
	// TODO: handle archive queries
	if in.Decrypt {
		err = errors.New("envelope decryption not implemented yet")
		c.Error(err)
		c.JSON(http.StatusNotImplemented, api.Error(err))
		return
	} else {
		if out, err = api.NewSecureEnvelopeList(page); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not process secure envelopes list request"))
			return
		}
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "secure_envelope_list.html",
	})
}

func (s *Server) SecureEnvelopeDetail(c *gin.Context) {
	var (
		err           error
		in            *api.EnvelopeQuery
		transactionID uuid.UUID
		envelopeID    ulid.ULID
		model         *models.SecureEnvelope
		out           *api.SecureEnvelope
	)

	in = &api.EnvelopeQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse envelope query"))
		return
	}

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the envelopeID passed in from the URL
	if envelopeID, err = ulid.Parse(c.Param("envelopeID")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("secure envelope not found"))
		return
	}

	// Fetch the model from the database
	if model, err = s.store.RetrieveSecureEnvelope(c.Request.Context(), transactionID, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction or secure envelope not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// TODO: handle decryption
	// TODO: handle archive queries
	if in.Decrypt {
		err = errors.New("envelope decryption not implemented yet")
		c.Error(err)
		c.JSON(http.StatusNotImplemented, api.Error(err))
		return
	} else {
		if out, err = api.NewSecureEnvelope(model); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error(err))
			return
		}
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "secure_envelope_detail.html",
	})
}

func CheckUUIDMatch(id, target uuid.UUID) error {
	if id == uuid.Nil {
		return ulids.ErrMissingID
	}

	if id != target {
		return ulids.ErrIDMismatch
	}

	return nil
}

func (s *Server) TransactionInfo(c *gin.Context) {
	var (
		err           error
		transactionID uuid.UUID
		transaction   *models.Transaction
		out           *api.Transaction
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	if transaction, err = s.store.RetrieveTransaction(c.Request.Context(), transactionID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if out, err = api.NewTransaction(transaction); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_info.html",
	})
}
