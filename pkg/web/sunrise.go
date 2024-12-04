package web

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

func (s *Server) SendMessageForm(c *gin.Context) {
	c.HTML(http.StatusOK, "send_message.html", scene.New(c))
}

func (s *Server) VerifySunriseUser(c *gin.Context) {
	c.HTML(http.StatusOK, "verify.html", scene.New(c))
}

func (s *Server) SunriseMessagePreview(c *gin.Context) {
	c.HTML(http.StatusOK, "view_message.html", scene.New(c))
}

func (s *Server) SendSunrise(c *gin.Context) {
	var (
		err             error
		in              *api.Sunrise
		out             *api.Transaction
		beneficiaryVASP *models.Counterparty
		originatorVASP  *models.Counterparty
		payload         *trisa.Payload
		packet          *postman.Packet
	)

	in = &api.Sunrise{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse sunrise transaction data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Get originator VASP information from the database (e.g. the sending party)
	if originatorVASP, err = s.Localparty(c.Request.Context()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
		return
	}

	// Get or create the counterparty for the associated email address
	if beneficiaryVASP, err = s.SunriseCounterparty(c.Request.Context(), in.Email, in.Counterparty); err != nil {
		c.Error(err)
		c.JSON(http.StatusConflict, api.Error("could not find or create a counterparty with the specified name and/or email address"))
		return
	}

	// Convert the incoming data into the appropriate TRISA data structures
	if payload, err = in.Payload(originatorVASP, beneficiaryVASP); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer"))
		return
	}

	// Create the log with the envelope ID for debugging
	envelopeID := uuid.New()
	ctx := c.Request.Context()
	log := logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Create a packet to begin the sending process
	if packet, err = postman.Send(payload, envelopeID, trisa.TransferStarted, log); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
		return
	}

	// Associate the beneficiary VASP as the remote counterparty
	packet.Counterparty = beneficiaryVASP

	// Create the transaction in the database
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return
	}
	defer packet.DB.Rollback()

	// Add the counterparty to the database associated with the transaction
	if err = packet.Out.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	// TODO: this sequence should be part of the packet refactor
	// Create the sunrise tokens for all recipients in the database

	// Send required emails

	// Update the transaction with the "response" from the user; e.g. the sunrise record

	// Read the record from the database to return to the user
	if err = packet.RefreshTransaction(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return
	}

	// Commit the transaction to the database
	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return
	}

	// Create the API response to send back to the user
	if out, err = api.NewTransaction(packet.Transaction); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Send 200 or 201 depending on if the transaction was created or not.
	var status int
	if packet.DB.Created() {
		status = http.StatusCreated
	} else {
		status = http.StatusOK
	}

	c.Negotiate(status, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_sent.html",
	})
}

func (s *Server) SunriseCounterparty(ctx context.Context, email, counterparty string) (*models.Counterparty, error) {
	return nil, nil
}
