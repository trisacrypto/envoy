package web

import (
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/postman"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"github.com/trisacrypto/envoy/pkg/web/scene"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
)

func (s *Server) PrepareTransaction(c *gin.Context) {
	var (
		err             error
		in              *api.Prepare
		out             *api.Prepared
		beneficiaryVASP *models.Counterparty
		originatorVASP  *models.Counterparty
	)

	in = &api.Prepare{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse prepare transaction data"))
		return
	}

	if err = in.Validate(); err != nil {
		c.Error(err)
		c.JSON(http.StatusUnprocessableEntity, api.Error(err))
		return
	}

	// Get originator VASP information from database
	if originatorVASP, err = s.Localparty(c.Request.Context()); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete prepare request"))
		return
	}

	// Parse the TravelAddress to identify the beneficiary VASP and lookup the
	// counterparty in the local database for IVMS101 information if any.
	if beneficiaryVASP, err = s.CounterpartyFromTravelAddress(c, in.TravelAddress); err != nil {
		// NOTE: CounterpartyFromTravelAddress handles API response back to user.
		return
	}

	// Convert the incoming data into the appropriate TRISA data structures
	out = &api.Prepared{
		TravelAddress: in.TravelAddress,
		Identity: &ivms101.IdentityPayload{
			Originator: &ivms101.Originator{
				OriginatorPersons: []*ivms101.Person{
					in.Originator.NaturalPerson(),
				},
				AccountNumbers: []string{
					in.Originator.CryptoAddress,
				},
			},
			Beneficiary: &ivms101.Beneficiary{
				BeneficiaryPersons: []*ivms101.Person{
					in.Beneficiary.NaturalPerson(),
				},
				AccountNumbers: []string{
					in.Beneficiary.CryptoAddress,
				},
			},
			OriginatingVasp: &ivms101.OriginatingVasp{
				OriginatingVasp: &ivms101.Person{
					Person: &ivms101.Person_LegalPerson{
						LegalPerson: originatorVASP.IVMSRecord,
					},
				},
			},
			BeneficiaryVasp: &ivms101.BeneficiaryVasp{
				BeneficiaryVasp: &ivms101.Person{
					Person: &ivms101.Person_LegalPerson{
						LegalPerson: beneficiaryVASP.IVMSRecord,
					},
				},
			},
			TransferPath:    nil,
			PayloadMetadata: nil,
		},
		Transaction: in.Transaction(),
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLData: scene.New(c).WithAPIData(out),
		HTMLName: "partials/send/preview.html",
	})
}

func (s *Server) SendPreparedTransaction(c *gin.Context) {
	time.Sleep(1 * time.Second) // Simulate a delay for testing
	if true {
		c.JSON(http.StatusBadRequest, api.Error("unable to send transfer to remote counterparty, please try again later"))
		return
	}

	var (
		err     error
		in      *api.Prepared
		payload *trisa.Payload
		packet  *postman.TRISAPacket
		out     *api.Transaction
	)

	// Handle the user input to the request
	in = &api.Prepared{}
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

	if payload, err = in.Payload(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not create payload for transfer"))
		return
	}

	// Create a packet to begin the sending process
	envelopeID := uuid.New()
	if packet, err = postman.SendTRISA(envelopeID, payload, trisa.TransferStarted); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not process send prepared transaction request"))
		return
	}

	// Add the log to the packet for debugging
	ctx := c.Request.Context()
	packet.Log = logger.Tracing(ctx).With().Str("envelope_id", envelopeID.String()).Logger()

	// Lookup the counterparty from the travel address in the request
	if packet.Counterparty, err = s.CounterpartyFromTravelAddress(c, in.TravelAddress); err != nil {
		// NOTE: CounterpartyFromTravelAddress handles API response back to user.
		return
	}

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

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(ctx, packet); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("unable to send transfer to remote counterparty, please try again later"))
		return
	}

	// Update transaction state based on response from counterparty
	if err = packet.In.UpdateTransaction(); err != nil {
		c.Error(err)
	}

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

	// If this is a UI request, then redirect the user to the transaction detail page
	if htmx.IsHTMXRequest(c) {
		htmx.Redirect(c, http.StatusFound, "/transactions/"+packet.Transaction.ID.String())
		return
	}

	// Send a JSON response back to the user.
	// Send 200 or 201 depending on if the transaction was created or not.
	var status int
	if packet.DB.Created() {
		status = http.StatusCreated
	} else {
		status = http.StatusOK
	}

	c.JSON(status, out)
}
