package web

import (
	"errors"
	"net/http"
	dberr "self-hosted-node/pkg/store/errors"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/web/api/v1"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/trisacrypto/trisa/pkg/ivms101"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

func (s *Server) PrepareTransaction(c *gin.Context) {
	var (
		err             error
		in              *api.Prepare
		out             *api.Prepared
		dst             string
		dstURL          *traddr.URL
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

	// Parse the TravelAddress to identify the beneficiary
	if dst, err = traddr.Decode(in.TravelAddress); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse the travel address"))
		return
	}

	if dstURL, err = traddr.Parse(dst); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse travel address url"))
		return
	}

	// Lookup the beneficiary VASP information from the travel address
	if beneficiaryVASP, err = s.store.LookupCounterparty(c.Request.Context(), dstURL.Hostname()); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			c.JSON(http.StatusNotFound, api.Error("could not find counterparty from travel address"))
			return
		}

		c.Error(err)
		c.JSON(http.StatusInternalServerError, api.Error("could not complete prepare request"))
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

func (s *Server) SendPreparedTransaction(c *gin.Context) {
	var (
		err error
		in  *api.Prepared
		out *api.Transaction
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

	// TODO: create the transaction
	// TODO: send the secure envelope
	// TODO: return the transaction and the secure envelope

	c.Negotiate(http.StatusOK, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		Data:     out,
		HTMLName: "transaction_detail.html",
	})
}
