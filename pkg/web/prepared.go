package web

import (
	"net/http"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	trisa "github.com/trisacrypto/trisa/pkg/trisa/api/v1beta1"
	"github.com/trisacrypto/trisa/pkg/trisa/envelope"
)

// PrepareTransaction - Prepare transaction data for sending to a counterparty
//
//	@Summary		Prepare transaction data for sending
//	@Description	Prepare transaction data for sending to a counterparty
//	@ID				prepareTransaction
//	@Security		BearerAuth
//	@Tags			Transaction
//	@Accept			json
//	@Produce		json
//	@Param			prepare	body		api.Prepare		true	"Transaction data to prepare"
//	@Success		200		{object}	api.Prepared	"Successful operation"
//	@Failure		400		{object}	api.Reply		"Invalid input"
//	@Failure		422		{object}	api.Reply		"Validation exception or missing field"
//	@Router			/v1/transactions/prepare [post]
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
		c.JSON(http.StatusBadRequest, api.Error(err))
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
		HTMLName: "transaction_preview.html",
	})
}

// SendPreparedTransaction - Send prepared transaction data to a counterparty
//
//	@Summary		Send prepared transaction data to counterparty
//	@Description	Send prepared transaction data to a counterparty
//	@Tags			Transaction
//	@ID				sendPreparedTransaction
//	@Security		BearerAuth
//	@Accept			json
//	@Produce		json
//	@Param			prepared	body		api.Prepared	true	"Prepared transaction data to send"
//	@Success		200			{object}	api.Transaction	"Successful operation"
//	@Failure		400			{object}	api.Reply		"Invalid input"
//	@Failure		422			{object}	api.Reply		"Validation exception or missing field"
//	@Router			/v1/transactions/send-prepared [post]
func (s *Server) SendPreparedTransaction(c *gin.Context) {
	var (
		err          error
		in           *api.Prepared
		out          *api.Transaction
		model        *models.Transaction
		envelopeID   uuid.UUID
		db           models.PreparedTransaction
		counterparty *models.Counterparty
		payload      *trisa.Payload
		outgoing     *envelope.Envelope
	)

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

	// Lookup the counterparty from the travel address in the request
	if counterparty, err = s.CounterpartyFromTravelAddress(c, in.TravelAddress); err != nil {
		// NOTE: CounterpartyFromTravelAddress handles API response back to user.
		return
	}

	// Create the transaction in the database
	envelopeID = uuid.New()
	if db, err = s.store.PrepareTransaction(c.Request.Context(), envelopeID); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not create transfer"))
		return
	}
	defer db.Rollback()

	// Add the counterparty to the database associated with the transaction
	if err = db.AddCounterparty(counterparty); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not update transfer with counterparty"))
		return
	}

	// Create the outgoing payload and envelope
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

	// Send the secure envelope and get secure envelope response
	// NOTE: SendEnvelope handles storing the incoming and outgoing envelopes in the database
	if err = s.SendEnvelope(c.Request.Context(), outgoing, counterparty, db); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("unable to send transfer to remote counterparty"))
		return
	}

	// Read the record from the database to return to the user
	if model, err = db.Fetch(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// Commit the transaction to the database
	if err = db.Commit(); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	// TODO: update transaction state based on response from counterparty

	// Create the API response to send back to the user
	if out, err = api.NewTransaction(model); err != nil {
		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete send transfer request"))
		return
	}

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_sent.html",
	})
}
