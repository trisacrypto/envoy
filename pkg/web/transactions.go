package web

import (
	"errors"
	"fmt"
	"net/http"

	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
)

//===========================================================================
// Transactions REST Resource
//===========================================================================

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

//===========================================================================
// Transaction Detail Actions
//===========================================================================

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

func (s *Server) SendEnvelopeForTransaction(c *gin.Context) {
	var (
		err          error
		in           *api.Envelope
		out          *api.Envelope
		envelopeID   uuid.UUID
		transaction  *models.Transaction
		counterparty *models.Counterparty
		payload      *trisa.Payload
		outgoing     *envelope.Envelope
		incoming     *envelope.Envelope
	)

	ctx := c.Request.Context()

	// Parse the envelopeID (also the transactionID) passed in from the URL
	if envelopeID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the envelope that the user wants to send from the JSON payload
	in = &api.Envelope{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse prepared transaction data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Lookup the transaction from the database
	if transaction, err = s.store.RetrieveTransaction(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Get the counterparty from the database
	if counterparty, err = s.store.RetrieveCounterparty(ctx, transaction.CounterpartyID.ULID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found: transaction needs to be updated"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Create the outgoing payload and envelope
	if in.Error != nil {
		// Create a secure envelope with an error
		if outgoing, err = envelope.New(nil, envelope.WithEnvelopeID(envelopeID.String())); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create outgoing envelope for transfer"))
			return
		}

		if outgoing, err = outgoing.Reject(in.Error); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create outgoing envelope for transfer"))
			return
		}

		if err = outgoing.ValidateMessage(); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("invalid trisa error"))
			return
		}
	} else {
		// Create a secure envelope with a Payload
		if payload, err = in.Payload(); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer"))
			return
		}

		if outgoing, err = envelope.New(payload, envelope.WithEnvelopeID(envelopeID.String())); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create outgoing envelope for transfer"))
			return
		}
	}

	// Send the secure envelope and get secure envelope response
	// TODO: determine if this is a TRISA or TRP transfer and send TRP
	if outgoing, incoming, err = s.SendTRISATransfer(ctx, outgoing, counterparty); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// TODO: unify secure envelope database storage code (see SendPreparedTransaction)
	// Save the outgoing envelope to the database
	storeOutgoing := models.FromOutgoingEnvelope(outgoing)

	// Fetch the public key for storing the outgoing envelope
	var storageKey keys.PublicKey
	if storageKey, err = s.trisa.StorageKey(incoming.Proto().PublicKeySignature, counterparty.CommonName); err != nil {
		c.Error(fmt.Errorf("could not fetch storage key: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Update the cryptography on the outgoing message for storage
	if err = storeOutgoing.Reseal(storageKey, outgoing.Crypto()); err != nil {
		c.Error(fmt.Errorf("could not encrypt outgoing message for storage: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	if err = s.store.CreateSecureEnvelope(ctx, storeOutgoing); err != nil {
		c.Error(fmt.Errorf("could not store outgoing secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Save the incoming envelope to the database
	storeIncoming := models.FromIncomingEnvelope(incoming)
	if err = s.store.CreateSecureEnvelope(ctx, storeIncoming); err != nil {
		c.Error(fmt.Errorf("could not store incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Convert the incoming envelope into something readable for the end user
	if out, err = api.NewEnvelope(incoming); err != nil {
		c.Error(fmt.Errorf("could not parse incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "envelope_sent.html",
	})
}

func (s *Server) AcceptTransaction(c *gin.Context) {
	c.AbortWithError(http.StatusNotImplemented, dberr.ErrNotImplemented)
}

func (s *Server) RejectTransaction(c *gin.Context) {
	c.AbortWithError(http.StatusNotImplemented, dberr.ErrNotImplemented)
}

//===========================================================================
// Secure Envelopes REST Resource
//===========================================================================

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

//===========================================================================
// Helpers
//===========================================================================

func CheckUUIDMatch(id, target uuid.UUID) error {
	if id == uuid.Nil {
		return ulids.ErrMissingID
	}

	if id != target {
		return ulids.ErrIDMismatch
	}

	return nil
}
