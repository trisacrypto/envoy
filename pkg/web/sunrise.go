package web

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
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
		packet          *postman.SunrisePacket
	)

	in = &api.Sunrise{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse sunrise transaction data"))
		return
	}

	if err = in.Validate(); err != nil {
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
	if beneficiaryVASP, err = s.store.GetOrCreateSunriseCounterparty(c.Request.Context(), in.Email, in.Counterparty); err != nil {
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

	// Create a packet to begin the sending process
	envelopeID := uuid.New()
	if packet, err = postman.SendSunrise(envelopeID, payload); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
		return
	}

	// Create the log with the envelope ID for debugging
	ctx := c.Request.Context()
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Associate the beneficiary VASP as the remote counterparty
	packet.Counterparty = beneficiaryVASP

	// Create the transaction in the database
	if packet.DB, err = s.store.PrepareTransaction(ctx, envelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
		return
	}
	defer packet.DB.Rollback()

	// Fetch the contacts from the counterparty and check that at least one exists.
	var contacts []*models.Contact
	if contacts, err = packet.Counterparty.Contacts(); err != nil {
		c.Error(err)
		if errors.Is(err, postman.ErrNoContacts) {
			c.JSON(http.StatusBadRequest, api.Error(err))
		} else {
			c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
		}
		return
	}

	// Prepare to send email
	invite := emails.SunriseInviteData{
		ComplianceName:  s.conf.Email.GetSenderName(),
		OriginatorName:  in.Originator.FullName(),
		BeneficiaryName: in.Beneficiary.FullName(),
		BaseURL:         s.conf.Sunrise.InviteURL(),
		SupportEmail:    s.conf.Email.SupportEmail,
		ComplianceEmail: s.conf.Email.ComplianceEmail,
	}

	// Create the sunrise tokens for all counterparty contacts and send emails
	for _, contact := range contacts {
		if err = packet.SendEmail(contact, invite); err != nil {
			c.Error(fmt.Errorf("could not send sunrise message to %s: %w", contact.Email, err))
			c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
			return
		}
	}

	// Fetch the storage key for the envelopes
	var storageKey keys.PublicKey
	if storageKey, err = s.trisa.StorageKey("", "sunrise"); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
		return
	}

	// Save the secure envelopes and the transaction, and refresh the transaction.
	if err = packet.Save(storageKey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
		return
	}

	// Commit the transaction to the database
	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
		return
	}

	// Create the API response to send back to the user
	if out, err = api.NewTransaction(packet.Transaction); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
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
