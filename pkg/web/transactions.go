package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	generic "github.com/trisacrypto/trisa/pkg/trisa/data/generic/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"google.golang.org/protobuf/types/known/anypb"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"go.rtnl.ai/ulid"
)

//===========================================================================
// Transactions REST Resource
//===========================================================================

func (s *Server) ListTransactions(c *gin.Context) {
	var (
		err  error
		in   *api.TransactionListQuery
		page *models.TransactionPage
		out  *api.TransactionsList
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

	// TODO: implement better pagination mechanism (with pagination tokens)

	// Fetch the list of transactions from the database
	if page, err = s.store.ListTransactions(c.Request.Context(), in.Query()); err != nil {
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
		HTMLName: "partials/transactions/list.html",
		HTMLData: scene.New(c).WithAPIData(out),
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
	in.Source = enum.SourceLocal.String()

	// Mark the transaction as a draft until a secure envelope is sent.
	in.Status = enum.StatusDraft.String()

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Convert the transaction request into a database model
	if transaction, err = in.Model(); err != nil {
		c.Error(fmt.Errorf("could not deserialize request into model: %w", err))
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.CreateTransaction(c.Request.Context(), transaction, &models.ComplianceAuditLog{}); err != nil {
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

	c.Negotiate(http.StatusCreated, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_create.html",
	})
}

func (s *Server) TransactionDetail(c *gin.Context) {
	var (
		err error
		out *api.Transaction
	)

	if out, err = s.retrieveTransaction(c); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/transactions/detail.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

// Helper method for pages and API routes that require access to the transaction detail
// but handle the output data differently (e.g. rendering a page or a partial).
// NOTE: no error handling or logging happens in this method, so callers must handle
// all errors and logging before aborting the request.
func (s *Server) retrieveTransaction(c *gin.Context) (out *api.Transaction, err error) {
	var (
		transactionID uuid.UUID
		transaction   *models.Transaction
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		// Return a db error so that the caller can return a 404 error.
		return nil, dberr.ErrNotFound
	}

	if transaction, err = s.store.RetrieveTransaction(c.Request.Context(), transactionID); err != nil {
		if !errors.Is(err, dberr.ErrNotFound) {
			// If this is a database error other than not found, log the error and
			// return an internal error to prevent leaking critical backend details to
			// the user that will confuse them or allow them to exploit the system.
			c.Error(err)
			return nil, dberr.ErrInternal
		}
		return nil, err
	}

	if out, err = api.NewTransaction(transaction); err != nil {
		return nil, err
	}

	return out, nil
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
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Convert the input transaction into a database model
	if transaction, err = in.Model(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Update the model in the database (which will update the pointer).
	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.UpdateTransaction(c.Request.Context(), transaction, &models.ComplianceAuditLog{}); err != nil {
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

	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.DeleteTransaction(c.Request.Context(), transactionID, &models.ComplianceAuditLog{}); err != nil {
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
		HTMLData: scene.Scene{"TransactionID": transactionID},
		JSONData: api.Reply{Success: true},
		HTMLName: "transaction_delete.html",
	})
}

//===========================================================================
// Transaction Detail Actions
//===========================================================================

func (s *Server) SendEnvelopeForTransaction(c *gin.Context) {
	var (
		err        error
		in         *api.Envelope
		out        *api.Envelope
		envelopeID uuid.UUID
		packet     *postman.TRISAPacket
	)

	// Parse the envelopeID (also the transactionID) passed in from the URL
	if envelopeID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the envelope that the user wants to send from the JSON payload
	in = &api.Envelope{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse envelope data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Create the outgoing packet
	if in.Error != nil {
		// Create a secure envelope with an error
		if packet, err = postman.SendTRISAReject(envelopeID, in.Error); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create outgoing packet for transfer"))
			return
		}
	} else {
		// Create a secure envelope with a Payload
		var payload *trisa.Payload
		if payload, err = in.Payload(); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer"))
			return
		}

		if packet, err = postman.SendTRISA(envelopeID, payload, in.ParseTransferState()); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create outgoing packet for transfer"))
			return
		}
	}

	// Ensure the logger is set!
	ctx := c.Request.Context()
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Lookup the transaction from the database
	if packet.Transaction, err = s.store.RetrieveTransaction(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Get the counterparty from the database
	if packet.Counterparty, err = s.store.RetrieveCounterparty(ctx, packet.Transaction.CounterpartyID.ULID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found: transaction needs to be updated"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Create a prepared transaction to update the transaction and secure envelopes
	//FIXME: COMPLETE AUDIT LOG
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID, &models.ComplianceAuditLog{}); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}
	defer packet.DB.Rollback()

	// Update the transaction based on the outgoing message from the API client
	if err = packet.Out.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(ctx, packet); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// Update transaction state based on response from counterparty
	if err = packet.In.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("transfer sent but unable to store secure envelopes locally"))
		return
	}

	// If the content requested is HTML (e.g. the web-front end), then
	// respond with a 204 no content response and the front-end will handle the
	// success message in the toast.
	if c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) == binding.MIMEHTML {
		htmx.Trigger(c, "transactionCompleted")
		return
	}

	// If the content request is JSON (e.g. the API) then render the incoming envelope
	// as the response by decrypting it and sending it back to the user.
	if out, err = api.NewEnvelope(packet.In.Model(), packet.In.Envelope); err != nil {
		c.Error(fmt.Errorf("could not parse incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) LatestEnvelope(c *gin.Context) {
	var (
		err           error
		in            *api.EnvelopeQuery
		transactionID uuid.UUID
		env           *models.SecureEnvelope
		decrypted     *envelope.Envelope
		out           *api.Envelope
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Retrieve the query parameters from the request
	in = &api.EnvelopeQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse query parameters"))
		return
	}

	if err = in.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Should be no error after validation.
	direction, _ := enum.ParseDirection(in.Direction)

	// Retrieve the latest secure envelope with a payload for the transaction from the database
	ctx := c.Request.Context()
	if env, err = s.store.LatestSecureEnvelope(ctx, transactionID, direction); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Decrypt the secure envelope using the private keys in the key store
	if decrypted, err = s.Decrypt(env); err != nil {
		// If we are unable to decrypt the envelope return partial content status.
		c.Error(err)

		var partial *api.SecureEnvelope
		if partial, err = api.NewSecureEnvelope(env); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error(err))
			return
		}

		c.Negotiate(http.StatusPartialContent, gin.Negotiate{
			Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
			Data:     partial,
			HTMLName: "partials/transactions/undecrypted.html",
			HTMLData: scene.New(c).WithAPIData(partial),
		})
		return
	}

	if out, err = api.NewEnvelope(env, decrypted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/transactions/payload.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) LatestPayloadEnvelope(c *gin.Context) {
	var (
		err           error
		in            *api.EnvelopeQuery
		transactionID uuid.UUID
		env           *models.SecureEnvelope
		decrypted     *envelope.Envelope
		out           *api.Envelope
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Retrieve the query parameters from the request
	in = &api.EnvelopeQuery{}
	if err = c.BindQuery(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse query parameters"))
		return
	}

	if err = in.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	// Should be no error after validation.
	direction, _ := enum.ParseDirection(in.Direction)

	// Retrieve the latest secure envelope with a payload for the transaction from the database
	ctx := c.Request.Context()
	if env, err = s.store.LatestPayloadEnvelope(ctx, transactionID, direction); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Decrypt the secure envelope using the private keys in the key store
	if decrypted, err = s.Decrypt(env); err != nil {
		// If we are unable to decrypt the envelope return partial content status.
		c.Error(err)

		var partial *api.SecureEnvelope
		if partial, err = api.NewSecureEnvelope(env); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error(err))
			return
		}

		c.Negotiate(http.StatusPartialContent, gin.Negotiate{
			Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
			Data:     partial,
			HTMLName: "partials/transactions/undecrypted.html",
			HTMLData: scene.New(c).WithAPIData(partial),
		})
		return
	}

	if out, err = api.NewEnvelope(env, decrypted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/transactions/payload.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) AcceptTransactionPreview(c *gin.Context) {
	var (
		err           error
		archived      bool
		status        enum.Status
		transactionID uuid.UUID
		env           *models.SecureEnvelope
		decrypted     *envelope.Envelope
		out           *api.Envelope
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Check that transaction is in a repairable state.
	ctx := c.Request.Context()
	if archived, status, err = s.store.TransactionState(ctx, transactionID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if archived {
		c.JSON(http.StatusBadRequest, api.Error("transaction is archived and cannot be accepted"))
		return
	}

	if status != enum.StatusReview {
		c.JSON(http.StatusBadRequest, api.Error("transaction not in a reviewable state"))
		return
	}

	// Retrieve the latest secure envelope for the transaction from the database
	if env, err = s.store.LatestSecureEnvelope(ctx, transactionID, enum.DirectionAny); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Decrypt the secure envelope using the private keys in the key store
	if decrypted, err = s.Decrypt(env); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("this payload cannot be decrypted"))
		return
	}

	if out, err = api.NewEnvelope(env, decrypted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/transactions/accept.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) AcceptTransaction(c *gin.Context) {
	var (
		err        error
		envelopeID uuid.UUID
		archived   bool
		status     enum.Status
		in         *api.Envelope
		payload    *trisa.Payload
		out        *api.Envelope
		packet     *postman.TRISAPacket
	)

	// Parse the envelopeID (also the transactionID) passed in from the URL
	if envelopeID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the envelope that the user wants to send from the JSON payload
	in = &api.Envelope{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse accept data"))
		return
	}

	// TODO: if the envelope is empty then retrieve the latest payload from the database
	// This will allow easy accepts for the transaction without statefulness.

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	if ts := in.ParseTransferState(); ts != trisa.TransferStateUnspecified && ts != trisa.TransferAccepted {
		c.JSON(http.StatusBadRequest, api.Error(fmt.Errorf("this endpoint does not accept the %q transfer state", ts.String())))
		return
	}

	// Create a secure envelope with a Payload
	if payload, err = in.Payload(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer accept"))
		return
	}

	// Verify that the transaction is in a state that we can perform the action on.
	// Check that transaction is in a repairable state.
	ctx := c.Request.Context()
	if archived, status, err = s.store.TransactionState(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if archived {
		c.JSON(http.StatusBadRequest, api.Error("transaction is archived and cannot be repaired"))
		return
	}

	if status != enum.StatusReview {
		c.JSON(http.StatusBadRequest, api.Error("transaction not in a reviewable state"))
		return
	}

	// Send the payload with the accept transfer state
	if packet, err = postman.SendTRISA(envelopeID, payload, trisa.TransferAccepted); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create outgoing packet for transfer accept"))
		return
	}

	// Ensure the logger is set!
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Lookup the transaction from the database
	if packet.Transaction, err = s.store.RetrieveTransaction(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Get the counterparty from the database
	if packet.Counterparty, err = s.store.RetrieveCounterparty(ctx, packet.Transaction.CounterpartyID.ULID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found: transaction needs to be updated"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Create a prepared transaction to update the transaction and secure envelopes
	//FIXME: COMPLETE AUDIT LOG
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID, &models.ComplianceAuditLog{}); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}
	defer packet.DB.Rollback()

	// Update the transaction based on the outgoing message from the API client
	if err = packet.Out.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(ctx, packet); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// Update transaction state based on response from counterparty
	if err = packet.In.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("transfer sent but unable to store secure envelopes locally"))
		return
	}

	// If the content requested is HTML (e.g. the web-front end), then redirect the user
	// to the transaction detail page and set a cookie to display a toast message
	if htmx.IsHTMXRequest(c) {
		detailURL, _ := url.JoinPath("/transactions", packet.Transaction.ID.String())
		s.AddToastMessage(c, "Transaction Accepted", "The transaction was accepted and successfully sent to the counterparty.", "success")
		htmx.Redirect(c, http.StatusSeeOther, detailURL)
		return
	}

	// If the content request is JSON (e.g. the API) then render the incoming envelope
	// as the response by decrypting it and sending it back to the user.
	if out, err = api.NewEnvelope(packet.In.Model(), packet.In.Envelope); err != nil {
		c.Error(fmt.Errorf("could not parse incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) RejectTransaction(c *gin.Context) {
	var (
		err        error
		archived   bool
		status     enum.Status
		envelopeID uuid.UUID
		in         *api.Rejection
		out        *api.Envelope
		packet     *postman.TRISAPacket
	)

	// Parse the envelopeID (also the transactionID) passed in from the URL
	if envelopeID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the envelope that the user wants to send from the JSON payload
	in = &api.Rejection{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse rejection data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Check that transaction is in a rejectable state.
	ctx := c.Request.Context()
	if archived, status, err = s.store.TransactionState(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if archived {
		c.JSON(http.StatusBadRequest, api.Error("transaction is archived and cannot be repaired"))
		return
	}

	if status != enum.StatusReview {
		c.JSON(http.StatusBadRequest, api.Error("transaction not in a reviewable state"))
		return
	}

	if packet, err = postman.SendTRISAReject(envelopeID, in.Proto()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process reject transaction request"))
		return
	}

	// Ensure the logger is set!
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Lookup the transaction from the database
	if packet.Transaction, err = s.store.RetrieveTransaction(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Get the counterparty from the database
	if packet.Counterparty, err = s.store.RetrieveCounterparty(ctx, packet.Transaction.CounterpartyID.ULID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found: transaction needs to be updated"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Create a prepared transaction to update the transaction and secure envelopes
	//FIXME: COMPLETE AUDIT LOG
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID, &models.ComplianceAuditLog{}); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}
	defer packet.DB.Rollback()

	// Update the transaction state based on the outgoing message
	if err = packet.Out.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(ctx, packet); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// Update the transaction state based on the incoming message from the counterparty
	if err = packet.In.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("transfer sent but unable to store secure envelopes locally"))
		return
	}

	// If the content requested is HTML (e.g. the web-front end), then
	// respond with a 204 no content response and the front-end will handle the
	// success message in the toast.
	if c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) == binding.MIMEHTML {
		htmx.Trigger(c, htmx.TransactionsUpdated)
		return
	}

	// If the content request is JSON (e.g. the API) then render the incoming envelope
	// as the response by decrypting it and sending it back to the user.
	if out, err = api.NewEnvelope(packet.In.Model(), packet.In.Envelope); err != nil {
		c.Error(fmt.Errorf("could not parse incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) RepairTransactionPreview(c *gin.Context) {
	var (
		err           error
		archived      bool
		status        enum.Status
		transactionID uuid.UUID
		errorEnv      *models.SecureEnvelope
		payloadEnv    *models.SecureEnvelope
		decrypted     *envelope.Envelope
		out           *api.Repair
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Check that transaction is in a repairable state.
	ctx := c.Request.Context()
	if archived, status, err = s.store.TransactionState(ctx, transactionID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if archived {
		c.JSON(http.StatusBadRequest, api.Error("transaction is archived and cannot be repaired"))
		return
	}

	if status != enum.StatusRepair {
		c.JSON(http.StatusBadRequest, api.Error("transaction not in a repairable state"))
		return
	}

	// Retrieve the latest secure envelope for the transaction from the database
	// this should be the error envelope that contains the required repair info.
	if errorEnv, err = s.store.LatestSecureEnvelope(ctx, transactionID, enum.DirectionIncoming); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Verify that the error envelope is actually an error
	if !errorEnv.IsError {
		c.JSON(http.StatusBadRequest, api.Error("transaction not in a repairable state"))
		return
	}

	// Retrieve the latest payload envelope from the database
	if payloadEnv, err = s.store.LatestPayloadEnvelope(ctx, transactionID, enum.DirectionAny); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Decrypt the payload envelope using private keys in the store
	if decrypted, err = s.Decrypt(payloadEnv); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("payload cannot be decrypted"))
		return
	}

	out = &api.Repair{}
	if out.Error, err = api.NewRejection(errorEnv); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not create repair preview"))
		return
	}

	if out.Envelope, err = api.NewEnvelope(payloadEnv, decrypted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not create repair preview"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/transactions/repair.html",
		HTMLData: scene.New(c).WithAPIData(out),
	})
}

func (s *Server) RepairTransaction(c *gin.Context) {
	var (
		err        error
		archived   bool
		status     enum.Status
		envelopeID uuid.UUID
		in         *api.Envelope
		payload    *trisa.Payload
		out        *api.Envelope
		packet     *postman.TRISAPacket
	)

	// Parse the envelopeID (also the transactionID) passed in from the URL
	if envelopeID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the envelope that the user wants to send from the JSON payload
	in = &api.Envelope{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse repair data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	if ts := in.ParseTransferState(); ts != trisa.TransferStateUnspecified && ts != trisa.TransferReview {
		c.JSON(http.StatusBadRequest, api.Error(fmt.Errorf("this endpoint does not accept the %q transfer state", ts.String())))
		return
	}

	// Create a secure envelope with a Payload
	if payload, err = in.Payload(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer repair"))
		return
	}

	// Check that transaction is in a repairable state.
	ctx := c.Request.Context()
	if archived, status, err = s.store.TransactionState(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if archived {
		c.JSON(http.StatusBadRequest, api.Error("transaction is archived and cannot be repaired"))
		return
	}

	if status != enum.StatusRepair {
		c.JSON(http.StatusBadRequest, api.Error("transaction not in a repairable state"))
		return
	}

	// Send the payload with the accept transfer state
	if packet, err = postman.SendTRISA(envelopeID, payload, trisa.TransferReview); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create outgoing packet for transfer repair"))
		return
	}

	// Ensure the logger is set!
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Lookup the transaction from the database
	if packet.Transaction, err = s.store.RetrieveTransaction(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Get the counterparty from the database
	if packet.Counterparty, err = s.store.RetrieveCounterparty(ctx, packet.Transaction.CounterpartyID.ULID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found: transaction needs to be updated"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Create a prepared transaction to update the transaction and secure envelopes
	//FIXME: COMPLETE AUDIT LOG
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID, &models.ComplianceAuditLog{}); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}
	defer packet.DB.Rollback()

	// Update the transaction based on the outgoing message from the API client
	if err = packet.Out.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(ctx, packet); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// Update transaction state based on response from counterparty
	if err = packet.In.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("transfer sent but unable to store secure envelopes locally"))
		return
	}

	// If the content requested is HTML (e.g. the web-front end), then redirect the user
	// to the transaction detail page and set a cookie to display a toast message
	if htmx.IsHTMXRequest(c) {
		detailURL, _ := url.JoinPath("/transactions", packet.Transaction.ID.String())
		s.AddToastMessage(c, "Transaction Repaired", "The transaction was repaired and successfully sent to the counterparty.", "success")
		htmx.Redirect(c, http.StatusSeeOther, detailURL)
		return
	}

	// If the content request is JSON (e.g. the API) then render the incoming envelope
	// as the response by decrypting it and sending it back to the user.
	if out, err = api.NewEnvelope(packet.In.Model(), packet.In.Envelope); err != nil {
		c.Error(fmt.Errorf("could not parse incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) CompleteTransactionPreview(c *gin.Context) {
	var (
		err           error
		archived      bool
		status        enum.Status
		transactionID uuid.UUID
		env           *models.SecureEnvelope
		decrypted     *envelope.Envelope
		wrapper       *api.Envelope
		out           *generic.Transaction
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Check that transaction is in a completable state.
	ctx := c.Request.Context()
	if archived, status, err = s.store.TransactionState(ctx, transactionID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if archived {
		c.JSON(http.StatusBadRequest, api.Error("transaction is archived and cannot be completed"))
		return
	}

	if status != enum.StatusAccepted {
		c.JSON(http.StatusBadRequest, api.Error("transaction not in an accepted state"))
		return
	}

	// Retrieve the latest secure envelope for the transaction from the database
	if env, err = s.store.LatestSecureEnvelope(ctx, transactionID, enum.DirectionAny); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Decrypt the secure envelope using the private keys in the key store
	if decrypted, err = s.Decrypt(env); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("this payload cannot be decrypted"))
		return
	}

	if wrapper, err = api.NewEnvelope(env, decrypted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if out = wrapper.TransactionPayload(); out == nil {
		c.Error(ErrNoTransactionPayload)
		c.JSON(http.StatusConflict, api.Error("transaction payload not found"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "partials/transactions/complete.html",
		HTMLData: scene.New(c).WithAPIData(out).With("EnvelopeID", transactionID),
	})
}

func (s *Server) CompleteTransaction(c *gin.Context) {
	var (
		err        error
		envelopeID uuid.UUID
		archived   bool
		status     enum.Status
		in         *generic.Transaction
		env        *models.SecureEnvelope
		decrypted  *envelope.Envelope
		payload    *trisa.Payload
		out        *api.Envelope
		packet     *postman.TRISAPacket
	)

	// Parse the envelopeID (also the transactionID) passed in from the URL
	if envelopeID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	in = &generic.Transaction{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse transaction data"))
		return
	}

	// Validate the transaction payload is sufficient to complete the transfer
	if err = api.ValidateTransactionPayload(in, enum.StatusCompleted); err != nil {
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Verify that the transaction is in a state that we can perform the action on.
	// Check that transaction is in a repairable state.
	ctx := c.Request.Context()
	if archived, status, err = s.store.TransactionState(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	if archived {
		c.JSON(http.StatusBadRequest, api.Error("transaction is archived and cannot be repaired"))
		return
	}

	if status != enum.StatusAccepted {
		c.JSON(http.StatusBadRequest, api.Error("transaction not in an accepted state"))
		return
	}

	// Get the last envelope payload from the transaction to add the completed
	// transaction payload information to.
	// Retrieve the latest secure envelope for the transaction from the database
	if env, err = s.store.LatestSecureEnvelope(ctx, envelopeID, enum.DirectionAny); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Decrypt the secure envelope using the private keys in the key store
	if decrypted, err = s.Decrypt(env); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("this payload cannot be decrypted"))
		return
	}

	if payload, err = decrypted.Payload(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to retrieve identity details"))
		return
	}

	// Update the payload transaction with the completed transaction
	if payload.Transaction, err = anypb.New(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process complete transaction request"))
		return
	}

	// Send the payload with the completed transfer state
	if packet, err = postman.SendTRISA(envelopeID, payload, trisa.TransferCompleted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process complete transaction request"))
		return
	}

	// Ensure the logger is set!
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Lookup the transaction from the database
	if packet.Transaction, err = s.store.RetrieveTransaction(ctx, envelopeID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Get the counterparty from the database
	if packet.Counterparty, err = s.store.RetrieveCounterparty(ctx, packet.Transaction.CounterpartyID.ULID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("counterparty not found: transaction needs to be updated"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// TODO: Handle Sunrise and TRP Counterparties
	if packet.Counterparty.Protocol != enum.ProtocolTRISA {
		c.Error(fmt.Errorf("%s protcol not supported for complete transaction", packet.Counterparty.Protocol))
		c.JSON(http.StatusBadRequest, api.Error("only the TRISA protocol is supported for this endpoint at this time"))
		return
	}

	// Create a prepared transaction to update the transaction and secure envelopes
	//FIXME: COMPLETE AUDIT LOG
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID, &models.ComplianceAuditLog{}); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}
	defer packet.DB.Rollback()

	// Update the transaction based on the outgoing message from the API client
	if err = packet.Out.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(ctx, packet); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// Update transaction state based on response from counterparty
	if err = packet.In.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("transfer sent but unable to store secure envelopes locally"))
		return
	}

	// If the content requested is HTML (e.g. the web-front end), then redirect the user
	// to the transaction detail page and set a cookie to display a toast message
	if htmx.IsHTMXRequest(c) {
		detailURL, _ := url.JoinPath("/transactions", packet.Transaction.ID.String())
		s.AddToastMessage(c, "Transaction Completed", "The transaction was completed and successfully sent to the counterparty.", "success")
		htmx.Redirect(c, http.StatusSeeOther, detailURL)
		return
	}

	// If the content request is JSON (e.g. the API) then render the incoming envelope
	// as the response by decrypting it and sending it back to the user.
	if out, err = api.NewEnvelope(packet.In.Model(), packet.In.Envelope); err != nil {
		c.Error(fmt.Errorf("could not parse incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) ArchiveTransaction(c *gin.Context) {
	var (
		err           error
		transactionID uuid.UUID
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.ArchiveTransaction(c.Request.Context(), transactionID, &models.ComplianceAuditLog{}); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Respond with a 204 no content response; use the HTMX trigger in the front-end
	// to handle the success message in the toast. Otherwise, just send the status.
	switch c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) {
	case binding.MIMEHTML:
		htmx.Trigger(c, htmx.TransactionsUpdated)
	default:
		c.Status(http.StatusNoContent)
	}
}

func (s *Server) UnarchiveTransaction(c *gin.Context) {
	var (
		err           error
		transactionID uuid.UUID
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	//FIXME: COMPLETE AUDIT LOG
	if err = s.store.UnarchiveTransaction(c.Request.Context(), transactionID, &models.ComplianceAuditLog{}); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("transaction not found"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error(err))
		return
	}

	// Respond with a 204 no content response; use the HTMX trigger in the front-end
	// to handle the success message in the toast. Otherwise, just send the status.
	switch c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) {
	case binding.MIMEHTML:
		htmx.Trigger(c, htmx.TransactionsUpdated)
	default:
		c.Status(http.StatusNoContent)
	}
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

	// TODO: handle archive queries
	code := http.StatusOK
	if in.Decrypt {
		envelopes := make([]*envelope.Envelope, 0, len(page.Envelopes))
		for i, model := range page.Envelopes {
			// Decrypt model and add it to the envelopes array
			var env *envelope.Envelope
			if env, err = s.Decrypt(model); err != nil {
				// If an envelope cannot be decrypted the error is logged but a null
				// envelope is returned instead of not returning any data.
				log.Debug().Err(err).Int("envelope", i).Msg("envelope decryption failure")
				code = http.StatusPartialContent
			}

			envelopes = append(envelopes, env)
		}

		if out, err = api.NewEnvelopeList(page, envelopes); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not process decrypted envelopes list request"))
			return
		}
	} else {
		if out, err = api.NewSecureEnvelopeList(page); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not process secure envelopes list request"))
			return
		}
	}

	c.Negotiate(code, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLData: scene.New(c).WithAPIData(out),
		HTMLName: "partials/transactions/envelopes.html",
	})
}

func (s *Server) SecureEnvelopeDetail(c *gin.Context) {
	var (
		err           error
		in            *api.EnvelopeQuery
		transactionID uuid.UUID
		envelopeID    ulid.ULID
		model         *models.SecureEnvelope
		out           any
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

	// TODO: handle archive queries
	if in.Decrypt {
		var env *envelope.Envelope
		if env, err = s.Decrypt(model); err != nil {
			// If we were unable to decrypt the envelope, use the partial content status
			c.Error(err)

			if out, err = api.NewSecureEnvelope(model); err != nil {
				c.Error(err)
				c.JSON(http.StatusInternalServerError, api.Error(err))
				return
			}

			c.Negotiate(http.StatusPartialContent, gin.Negotiate{
				Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
				Data:     out,
				HTMLName: "partials/transactions/undecrypted.html",
				HTMLData: scene.New(c).WithAPIData(out),
			})
			return
		}

		if out, err = api.NewEnvelope(model, env); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error(err))
			return
		}

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
		HTMLName: "partials/transactions/envelope.html",
		HTMLData: scene.New(c).WithAPIData(out).With("Decrypted", in.Decrypt),
	})
}

//===========================================================================
// Helpers
//===========================================================================

func CheckUUIDMatch(id, target uuid.UUID) error {
	if id == uuid.Nil {
		return ErrMissingID
	}

	if id != target {
		return ErrIDMismatch
	}

	return nil
}
