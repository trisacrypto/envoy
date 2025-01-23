package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/ulids"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"

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
	in.Source = models.SourceLocal

	// Mark the transaction as a draft until a secure envelope is sent.
	in.Status = models.StatusDraft

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

	c.Negotiate(http.StatusCreated, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_create.html",
	})
}

func (s *Server) TransactionDetail(c *gin.Context) {
	var (
		err           error
		transactionID uuid.UUID
		query         *api.TransactionQuery
		transaction   *models.Transaction
		out           *api.Transaction
	)

	// Parse the transactionID passed in from the URL
	if transactionID, err = uuid.Parse(c.Param("id")); err != nil {
		c.JSON(http.StatusNotFound, api.Error("transaction not found"))
		return
	}

	// Parse the transaction query
	query = &api.TransactionQuery{}
	if err = c.BindQuery(query); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse transaction query in request"))
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

	// Determine the HTML template to use based on the query
	var template string
	switch query.Detail {
	case api.DetailFull:
		template = "transaction_detail.html"
	case api.DetailPreview:
		template = "transaction_info.html"
	default:
		c.Error(fmt.Errorf("unhandled detail query '%q'", query.Detail))
		template = "transaction_detail.html"
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: template,
		HTMLData: scene.New(c).WithAPIData(out),
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
		c.JSON(http.StatusBadRequest, api.Error("could not parse envelope data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Create the log with the envelope ID for debugging
	log := logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Create the outgoing packet
	if in.Error != nil {
		// Create a secure envelope with an error
		if packet, err = postman.SendTRISAReject(in.Error, envelopeID, log); err != nil {
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

		if packet, err = postman.SendTRISA(payload, envelopeID, in.ParseTransferState(), log); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create outgoing packet for transfer"))
			return
		}
	}

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
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
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

func (s *Server) LatestPayloadEnvelope(c *gin.Context) {
	var (
		err           error
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

	// Retrieve the latest secure envelope with a payload for the transaction from the database
	ctx := c.Request.Context()
	if env, err = s.store.LatestPayloadEnvelope(ctx, transactionID, models.DirectionAny); err != nil {
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
		HTMLName: "transaction_payload.html",
	})
}

func (s *Server) AcceptTransactionPreview(c *gin.Context) {
	var (
		err           error
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

	// Retrieve the latest secure envelope for the transaction from the database
	ctx := c.Request.Context()
	if env, err = s.store.LatestSecureEnvelope(ctx, transactionID, models.DirectionAny); err != nil {
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
		HTMLName: "transaction_accept.html",
	})
}

func (s *Server) AcceptTransaction(c *gin.Context) {
	var (
		err        error
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

	ctx := c.Request.Context()
	log := logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Create a secure envelope with a Payload
	if payload, err = in.Payload(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer accept"))
		return
	}

	// Send the payload with the accept transfer state
	if packet, err = postman.SendTRISA(payload, envelopeID, trisa.TransferAccepted, log); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create outgoing packet for transfer accept"))
		return
	}

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
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
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
	if c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) == binding.MIMEHTML {
		detailURL, _ := url.JoinPath("/transactions", packet.Transaction.ID.String(), "info")
		setToastCookie(c, "transaction_send_success", "true", detailURL, s.conf.Web.Auth.CookieDomain)

		htmx.Redirect(c, http.StatusFound, detailURL)
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

	ctx := c.Request.Context()
	log := logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	if packet, err = postman.SendTRISAReject(in.Proto(), envelopeID, log); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process reject transaction request"))
		return
	}

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
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
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
		htmx.Trigger(c, "transactionRejected")
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

	// Retrieve the latest secure envelope for the transaction from the database
	// this should be the error envelope that contains the required repair info.
	ctx := c.Request.Context()
	if errorEnv, err = s.store.LatestSecureEnvelope(ctx, transactionID, models.DirectionIncoming); err != nil {
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
	if payloadEnv, err = s.store.LatestPayloadEnvelope(ctx, transactionID, models.DirectionAny); err != nil {
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
		HTMLName: "transaction_repair.html",
	})
}

func (s *Server) RepairTransaction(c *gin.Context) {
	var (
		err        error
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

	ctx := c.Request.Context()
	log := logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Create a secure envelope with a Payload
	if payload, err = in.Payload(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer repair"))
		return
	}

	// Send the payload with the accept transfer state
	if packet, err = postman.SendTRISA(payload, envelopeID, trisa.TransferReview, log); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create outgoing packet for transfer repair"))
		return
	}

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
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
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
	if c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) == binding.MIMEHTML {
		detailURL, _ := url.JoinPath("/transactions", packet.Transaction.ID.String(), "info")
		setToastCookie(c, "transaction_send_success", "true", detailURL, s.conf.Web.Auth.CookieDomain)

		htmx.Redirect(c, http.StatusFound, detailURL)
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
	code := http.StatusOK
	if in.Decrypt {
		var env *envelope.Envelope
		if env, err = s.Decrypt(model); err != nil {
			// If we were unable to decrypt the envelope, use the partial content status
			c.Error(err)
			code = http.StatusPartialContent
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

	c.Negotiate(code, gin.Negotiate{
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

func setToastCookie(c *gin.Context, name, value, path, domain string) {
	secure := !auth.IsLocalhost(domain)
	c.SetCookie(name, value, 1, path, domain, secure, false)
}
