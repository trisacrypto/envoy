package web

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/rs/zerolog/log"
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
		HTMLData: scene.Scene{"TransactionID": transactionID},
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

func (s *Server) SendEnvelopeForTransaction(c *gin.Context) {
	var (
		err          error
		in           *api.Envelope
		out          *api.Envelope
		envelopeID   uuid.UUID
		transaction  *models.Transaction
		counterparty *models.Counterparty
		db           models.PreparedTransaction
		payload      *trisa.Payload
		outgoing     *envelope.Envelope
		incoming     *models.SecureEnvelope
		decrypted    *envelope.Envelope
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
		if outgoing, err = envelope.WrapError(in.Error, envelope.WithEnvelopeID(envelopeID.String())); err != nil {
			c.Error(err)
			c.JSON(http.StatusBadRequest, api.Error("could not create outgoing envelope for transfer"))
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

	// Create a prepared transaction to update the transaction and secure envelopes
	// TODO: ensure that the transaction has not been created!
	if db, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}
	defer db.Rollback()

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(ctx, outgoing, counterparty, db); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// TODO: update transaction state based on response from counterparty
	if err = db.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("transfer sent but unable to store secure envelopes locally"))
		return
	}

	detailURL, _ := url.JoinPath("/transactions", transaction.ID.String(), "info")
	// Set a cookie to show a toast message on the page redirect.
	setToastCookie(c, "transaction_send_success", "true", detailURL, s.conf.Web.Auth.CookieDomain)

	// If the content requested is HTML (e.g. the web-front end), then redirect the user
	// to the transaction detail page.
	if c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) == binding.MIMEHTML {
		htmx.Redirect(c, http.StatusFound, detailURL)
		return
	}

	// Retrieve the secure envelope model for the incoming envelope
	if incoming, err = s.store.LatestSecureEnvelope(ctx, transaction.ID, models.DirectionIncoming); err != nil {
		c.Error(fmt.Errorf("could not retrieve incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	// Decrypt the incoming secure envelope
	// TODO: why are we decrypting the incoming secure envelope again?
	if decrypted, err = s.Decrypt(incoming); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	// If the content request is JSON (e.g. the API) then render the incoming envelope
	// as the response by decrypting it and sending it back to the user.
	if out, err = api.NewEnvelope(incoming, decrypted); err != nil {
		c.Error(fmt.Errorf("could not parse incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	c.JSON(http.StatusOK, out)
}

func (s *Server) AcceptTransaction(c *gin.Context) {
	c.AbortWithError(http.StatusNotImplemented, dberr.ErrNotImplemented)
}

func (s *Server) RejectTransaction(c *gin.Context) {
	var (
		err          error
		envelopeID   uuid.UUID
		in           *api.Rejection
		out          *api.Envelope
		db           models.PreparedTransaction
		transaction  *models.Transaction
		counterparty *models.Counterparty
		outgoing     *envelope.Envelope
		incoming     *models.SecureEnvelope
		decrypted    *envelope.Envelope
	)

	ctx := c.Request.Context()

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

	if outgoing, err = envelope.WrapError(in.Proto(), envelope.WithEnvelopeID(envelopeID.String())); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create outgoing envelope for transfer"))
		return
	}

	// Create a prepared transaction to update the transaction and secure envelopes
	// TODO: ensure that the transaction has not been created!
	if db, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("unable to send transfer to remote counterparty"))
		return
	}
	defer db.Rollback()

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(ctx, outgoing, counterparty, db); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// TODO: update transaction state based on response from counterparty
	if err = db.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("transfer sent but unable to store secure envelopes locally"))
		return
	}

	// If the content requested is HTML (e.g. the web-front end), then set a toast
	// cookie and respond with a 204 no content response
	if c.NegotiateFormat(binding.MIMEJSON, binding.MIMEHTML) == binding.MIMEHTML {
		// Set a cookie to show a toast message on the page redirect.
		detailURL, _ := url.JoinPath("/transactions", transaction.ID.String(), "info")
		setToastCookie(c, "transaction_reject_success", "true", detailURL, s.conf.Web.Auth.CookieDomain)

		c.Status(http.StatusNoContent)
		return
	}

	// Retrieve the secure envelope model for the incoming envelope
	if incoming, err = s.store.LatestSecureEnvelope(ctx, transaction.ID, models.DirectionIncoming); err != nil {
		c.Error(fmt.Errorf("could not retrieve incoming secure envelope: %w", err))
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	// Decrypt the incoming secure envelope
	// TODO: why are we decrypting the incoming secure envelope again?
	if decrypted, err = s.Decrypt(incoming); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not return incoming response from counterparty"))
		return
	}

	// If the content request is JSON (e.g. the API) then render the incoming envelope
	// as the response by decrypting it and sending it back to the user.
	if out, err = api.NewEnvelope(incoming, decrypted); err != nil {
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
	if in.Decrypt {
		envelopes := make([]*envelope.Envelope, 0, len(page.Envelopes))
		for i, model := range page.Envelopes {
			// Decrypt model and add it to the envelopes array
			var env *envelope.Envelope
			if env, err = s.Decrypt(model); err != nil {
				// If an envelope cannot be decrypted the error is logged but a null
				// envelope is returned instead of not returning any data.
				log.Debug().Err(err).Int("envelope", i).Msg("envelope decryption failure")
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
	c.SetCookie(name, value, 60, path, domain, secure, false)
}
