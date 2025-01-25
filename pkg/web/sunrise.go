package web

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/sunrise"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

const defaultSunriseExpiration = 14 * 24 * time.Hour

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
		packet          *postman.TRISAPacket
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
	if packet, err = postman.SendTRISA(envelopeID, payload, trisa.TransferStarted); err != nil {
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

	// Add the counterparty to the database associated with the transaction
	if err = packet.Out.UpdateTransaction(); err != nil {
		c.Error(err)
	}

	// TODO: this sequence should be part of the packet refactor
	// Fetch the contacts from the counterparty and check that at least one exists.
	var contacts []*models.Contact
	if contacts, err = packet.Counterparty.Contacts(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))

		return
	}

	if len(contacts) == 0 {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("no contacts are associated with counterparty, cannot send sunrise messages"))
		return
	}

	// Prepare to send email
	invite := emails.SunriseInviteData{
		OriginatorName:  in.Originator.FullName(),
		BeneficiaryName: in.Beneficiary.FullName(),
		BaseURL:         s.conf.Sunrise.InviteURL(),
	}

	// Create the sunrise tokens for all counterparty contacts and send emails
	for _, contact := range contacts {
		// Create a sunrise record for dataabase storage
		record := &models.Sunrise{
			EnvelopeID: envelopeID,
			Email:      contact.Email,
			Expiration: time.Now().Add(defaultSunriseExpiration),
			Status:     models.StatusDraft,
		}

		// This method will create the ID on the sunrise record
		if err = packet.DB.CreateSunrise(record); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
			return
		}

		// Create the HMAC verification token for the contact
		verification := sunrise.NewToken(record.ID, record.Expiration)

		// Sign the verification token
		if invite.Token, record.Signature, err = verification.Sign(); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
			return
		}

		// Send the email to the user
		var email *emails.Email
		if email, err = emails.NewSunriseInvite(contact.Address().String(), invite); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
			return
		}

		if err = email.Send(); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
			return
		}

		// Update the sunrise record in the database with the token and sent on timestamp
		record.SentOn = sql.NullTime{Valid: true, Time: time.Now()}
		if err = packet.DB.UpdateSunrise(record); err != nil {
			c.Error(err)
			c.JSON(http.StatusInternalServerError, api.Error("could not complete sunrise request"))
			return
		}
	}

	// TODO: Update the transaction with the "response" from the user; e.g. the sunrise record

	// Read the record from the database to return to the user
	if err = packet.RefreshTransaction(); err != nil {
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
