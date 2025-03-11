package web

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/trisacrypto/envoy/pkg/emails"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/sunrise"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
	"github.com/trisacrypto/trisa/pkg/trisa/keys"
	"go.rtnl.ai/ulid"
)

const GenericComplianceName = "A VASP Compliance Team using TRISA Envoy"

// The incoming request should be coming from a compliance officer at a VASP who has
// received a sunrise message. The request should include a verification token,
// otherwise a 404 is returned. If the verification token is valid, this endpoint will
// send a OTP to the user's email address, which the user must type in as a secondary
// check that they do indeed have access to the email address the message was sent to.
// If the token is expired, the user can request a new verification token sent to their
// email address. If the token is completed, e.g. the user has already logged in with
// it at least once; they can do the OTP passcheck and receive access to the data.
func (s *Server) VerifySunriseUser(c *gin.Context) {
	var (
		err   error
		in    *api.SunriseVerification
		log   zerolog.Logger
		model *models.Sunrise
		token sunrise.VerificationToken
	)

	in = &api.SunriseVerification{}
	ctx := c.Request.Context()
	log = logger.Tracing(ctx)

	if err = c.BindQuery(in); err != nil {
		// TODO: do we need to handle UI 400 errors?
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse query string"))
		return
	}

	if err = in.Validate(); err != nil {
		// If the token is invalid or missing, return a 404.
		// NOTE: do not log an error as this is very verbose, instead just a debug message
		log.Debug().Err(err).Msg("sunrise request with invalid token")
		c.HTML(http.StatusNotFound, "sunrise_404.html", scene.New(c))
		return
	}

	token = in.VerificationToken()

	// Get the sunrise record from the database
	if model, err = s.store.RetrieveSunrise(ctx, token.SunriseID()); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.HTML(http.StatusNotFound, "sunrise_404.html", scene.New(c))
			return
		}

		c.Error(err)
		c.HTML(http.StatusInternalServerError, "500.html", scene.New(c))
		return
	}

	// Check that the token is valid
	if secure, err := model.Signature.Verify(token); err != nil || !secure {
		// If the token is not secure or verifiable, return a 404 but be freaked out
		log.Warn().Err(err).Bool("secure", secure).Msg("a sunrise verification request was made to an existing message but hmac verification failed")
		c.HTML(http.StatusNotFound, "sunrise_404.html", scene.New(c))
		return
	}

	if model.IsExpired() {
		// The token is expired and has not yet been completed.
		// TODO: resend a new verification token
		log.Debug().Msg("received a request with an expired verification token")
		c.JSON(http.StatusGone, api.Error("verification token has expired"))
		return
	}

	// If an OTP is not required, set the validation cookie and redirect the user to
	// the sunrise message preview.
	if !s.conf.Sunrise.RequireOTP {
		if err = s.SetSunriseAuthCookies(c, model); err != nil {
			c.Error(err)
			c.HTML(http.StatusInternalServerError, "500.html", scene.New(c))
			return
		}

		c.Redirect(http.StatusTemporaryRedirect, "/sunrise/review")
		return
	}

	// TODO: send one time code to the user's email address

	// Render the OTP form
	c.HTML(http.StatusOK, "verify.html", scene.New(c))
}

func (s *Server) SunriseMessageReview(c *gin.Context) {
	var (
		err         error
		claims      *auth.Claims
		subjectType auth.SubjectType
		sunriseID   ulid.ULID
		sunriseMsg  *models.Sunrise
		env         *models.SecureEnvelope
		decrypted   *envelope.Envelope
		out         *api.Envelope
	)

	ctx := c.Request.Context()
	log := logger.Tracing(ctx)

	if claims, err = auth.GetClaims(c); err != nil {
		c.Error(err)
		c.HTML(http.StatusInternalServerError, "500.html", scene.New(c))
		return
	}

	// Get the sunrise record ID from the subject of the claims
	if subjectType, sunriseID, err = claims.SubjectID(); err != nil {
		c.Error(err)
		c.HTML(http.StatusInternalServerError, "500.html", scene.New(c))
		return
	}

	// Validate the subject type
	if subjectType != auth.SubjectSunrise {
		log.Debug().Str("subject_type", subjectType.String()).Msg("invalid subject type for sunrise review")
		c.HTML(http.StatusNotFound, "sunrise_404.html", scene.New(c))
		return
	}

	// Retrieve the sunrise record from the database
	if sunriseMsg, err = s.store.RetrieveSunrise(ctx, sunriseID); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.HTML(http.StatusNotFound, "sunrise_404.html", scene.New(c))
			return
		}

		c.Error(err)
		c.HTML(http.StatusInternalServerError, "500.html", scene.New(c))
		return
	}

	// Retrieve the latest secure envelope from the database
	if env, err = s.store.LatestSecureEnvelope(ctx, sunriseMsg.EnvelopeID, models.DirectionOut); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.HTML(http.StatusNotFound, "sunrise_404.html", scene.New(c))
			return
		}

		c.Error(err)
		c.HTML(http.StatusInternalServerError, "500.html", scene.New(c))
		return
	}

	// Decrypt the secure envelope using the private keys in the key store
	if decrypted, err = s.Decrypt(env); err != nil {
		c.Error(err)
		c.HTML(http.StatusInternalServerError, "500.html", scene.New(c))
		return
	}

	if out, err = api.NewEnvelope(env, decrypted); err != nil {
		c.Error(err)
		c.HTML(http.StatusInternalServerError, "500.html", scene.New(c))
		return
	}

	c.HTML(http.StatusOK, "review_message.html", scene.New(c).WithAPIData(out))
}

func (s *Server) SunriseMessageReject(c *gin.Context) {
	var (
		err        error
		in         *api.Rejection
		claims     *auth.Claims
		sunriseID  ulid.ULID
		sunriseMsg *models.Sunrise
		packet     *postman.SunrisePacket
	)

	in = &api.Rejection{}
	if err = c.BindJSON(&in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse request"))
		return
	}

	if err = in.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	if claims, err = auth.GetClaims(c); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	if _, sunriseID, err = claims.SubjectID(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Retrieve the sunrise record from the database
	ctx := c.Request.Context()
	if sunriseMsg, err = s.store.RetrieveSunrise(ctx, sunriseID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Create the reject packet
	if packet, err = postman.ReceiveSunriseReject(sunriseMsg.EnvelopeID, in.Proto()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Ensure logger is set!
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", sunriseMsg.EnvelopeID.String()).Logger()

	// Fetch the transaction from the database
	if packet.Transaction, err = s.store.RetrieveTransaction(ctx, sunriseMsg.EnvelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// If the transaction state is not in pending, return an error (prevent multiple rejects)
	if packet.Transaction.Status != models.StatusPending {
		c.Error(err)
		c.JSON(http.StatusConflict, api.Error("could not complete request"))
		return
	}

	// Get the counterparty from the database
	if packet.Counterparty, err = s.store.RetrieveCounterparty(ctx, packet.Transaction.CounterpartyID.ULID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Create a prepared transaction to create secure envelopes
	if packet.DB, err = s.store.PrepareTransaction(ctx, packet.Transaction.ID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}
	defer packet.DB.Rollback()

	// Fetch the storage keys for the envelopes
	var storageKey keys.PublicKey
	if storageKey, err = s.trisa.StorageKey("", "sunrise"); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Save the secure envelopes and the transaction, and refresh the transaction.
	if err = packet.Save(storageKey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Finalize the work on the transaction
	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// This is currently an HTMX response so simply respond with a 200 so that the
	// success toast message pops up in the front end.
	c.JSON(http.StatusOK, api.Reply{Success: true})
}

func (s *Server) SunriseMessageAccept(c *gin.Context) {
	c.HTML(http.StatusOK, "sunrise_accept.html", scene.New(c))
	if true {
		return
	}

	var (
		err        error
		in         *api.Envelope
		claims     *auth.Claims
		sunriseID  ulid.ULID
		sunriseMsg *models.Sunrise
		env        *models.SecureEnvelope
		decrypted  *envelope.Envelope
		payload    *trisa.Payload
		packet     *postman.SunrisePacket
	)

	in = &api.Envelope{}
	if err = c.BindJSON(&in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse request"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	if claims, err = auth.GetClaims(c); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	if _, sunriseID, err = claims.SubjectID(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Retrieve the sunrise record from the database
	ctx := c.Request.Context()
	if sunriseMsg, err = s.store.RetrieveSunrise(ctx, sunriseID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Load the latest secure envelope from the database to populate the complete
	// details since the incoming envelope will only have beneficiary info.
	if env, err = s.store.LatestSecureEnvelope(ctx, sunriseMsg.EnvelopeID, models.DirectionOut); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	if decrypted, err = s.Decrypt(env); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	var orig *api.Envelope
	if orig, err = api.NewEnvelope(env, decrypted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	now := time.Now()
	in.ID = orig.ID
	in.EnvelopeID = orig.EnvelopeID
	in.Transaction = orig.Transaction
	in.Pending = orig.Pending
	in.Sunrise = orig.Sunrise
	in.SentAt = orig.SentAt
	in.ReceivedAt = &now
	in.Identity.OriginatingVasp = orig.Identity.OriginatingVasp
	in.Identity.Originator = orig.Identity.Originator
	in.Identity.TransferPath = orig.Identity.TransferPath
	in.Identity.PayloadMetadata = orig.Identity.PayloadMetadata

	// Create a secure envelope with a Payload
	if payload, err = in.Payload(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not process request"))
		return
	}

	if packet, err = postman.ReceiveSunriseAccept(sunriseMsg.EnvelopeID, payload); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process request"))
		return
	}

	// Ensure logger is set!
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", sunriseMsg.EnvelopeID.String()).Logger()

	// Fetch the transaction from the database
	if packet.Transaction, err = s.store.RetrieveTransaction(ctx, sunriseMsg.EnvelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// If the transaction state is not in pending, return an error (prevent multiple rejects)
	if packet.Transaction.Status != models.StatusPending {
		c.Error(err)
		c.JSON(http.StatusConflict, api.Error("could not complete request"))
		return
	}

	// Get the counterparty from the database
	if packet.Counterparty, err = s.store.RetrieveCounterparty(ctx, packet.Transaction.CounterpartyID.ULID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Create a prepared transaction to create secure envelopes
	if packet.DB, err = s.store.PrepareTransaction(ctx, packet.Transaction.ID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}
	defer packet.DB.Rollback()

	// Update the counterparty information from the beneficiary VASP information.
	if err = packet.UpdateCounterparty(in.BeneficiaryVASP()); err != nil {
		// Do not stop processing here: the counterparty information is not critical
		packet.Log.Warn().Err(err).Msg("could not update counterparty information")
	}

	// Fetch the storage keys for the envelopes
	var storageKey keys.PublicKey
	if storageKey, err = s.trisa.StorageKey("", "sunrise"); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Save the secure envelopes and the transaction, and refresh the transaction.
	if err = packet.Save(storageKey); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Finalize the work on the transaction
	if err = packet.DB.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// This is currently an HTMX response so simply respond with a 200 so that the
	// success toast message pops up in the front end.
	c.HTML(http.StatusOK, "sunrise_accept.html", scene.New(c))
}

func (s *Server) SunriseMessageDownload(c *gin.Context) {
	var (
		err        error
		claims     *auth.Claims
		sunriseID  ulid.ULID
		sunriseMsg *models.Sunrise
		env        *models.SecureEnvelope
		decrypted  *envelope.Envelope
		out        *api.Envelope
	)

	if claims, err = auth.GetClaims(c); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	if _, sunriseID, err = claims.SubjectID(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Retrieve the sunrise record from the database
	ctx := c.Request.Context()
	if sunriseMsg, err = s.store.RetrieveSunrise(ctx, sunriseID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Load the latest secure envelope from the database to populate the complete
	// details since the incoming envelope will only have beneficiary info.
	if env, err = s.store.LatestSecureEnvelope(ctx, sunriseMsg.EnvelopeID, models.DirectionAny); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	if decrypted, err = s.Decrypt(env); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	if out, err = api.NewEnvelope(env, decrypted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Remove the secure envelope since it cannot be decrypted by the user
	out.SecureEnvelope = nil

	// Marshal the envelope to JSON
	var data []byte
	if data, err = json.MarshalIndent(out, "", "  "); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete request"))
		return
	}

	// Download the JSON data of the envelope
	fileName := fmt.Sprintf("travel-rule-message-%s.json", sunriseMsg.EnvelopeID.String())
	c.Header("Content-Disposition", "attachment; filename="+fileName)
	c.Header("ACcept-Length", fmt.Sprintf("%d", len(data)))
	c.Data(http.StatusOK, binding.MIMEJSON, data)
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
		ComplianceName:  s.GetComplianceName(),
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
	if err = packet.Create(storageKey); err != nil {
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

func (s *Server) GetComplianceName() string {
	if name := s.conf.Email.GetSenderName(); name != "" {
		return name
	}

	if name := s.conf.Organization; name != "Envoy" && name != "" {
		return name
	}

	return GenericComplianceName
}

func (s *Server) SetSunriseAuthCookies(c *gin.Context, model *models.Sunrise) (err error) {
	var (
		claims       *auth.Claims
		accessToken  string
		refreshToken string
	)

	if claims, err = auth.NewClaims(c.Request.Context(), model); err != nil {
		return err
	}

	if accessToken, refreshToken, err = s.issuer.CreateTokens(claims); err != nil {
		return err
	}

	if err = auth.SetAuthCookies(c, accessToken, refreshToken, s.conf.Web.Auth.CookieDomain); err != nil {
		return err
	}

	return nil
}
