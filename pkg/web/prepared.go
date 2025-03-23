package web

import (
	"net/http"

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

	// Parse the routing object to identify the beneficiary VASP and lookup the
	// counterparty in the local database for IVMS101 information if any.
	// If this is a sunrise message, the counterparty is created if necessary.
	if beneficiaryVASP, err = s.ResolveCounterparty(c, in.Routing); err != nil {
		// NOTE: CounterpartyFromTravelAddress handles API response back to user.
		return
	}

	// Convert the incoming data into the appropriate TRISA data structures
	out = &api.Prepared{
		Routing: in.Routing,
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
	var (
		err     error
		in      *api.Prepared
		payload *trisa.Payload
		packet  *postman.Packet
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

	// Send the transfer to the counterparty and get the secure envelope response
	// NOTE: Send handles any error response that needs to be sent to the user.
	// WARNING: Send commits/rollsback the database transaction from the packet.
	if packet, err = s.Send(c, in.Routing, payload); err != nil {
		c.Error(err)
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
		htmx.Redirect(c, http.StatusSeeOther, "/transactions/"+packet.Transaction.ID.String())
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
