package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

// EncodeTravelAddress - Encodes a travel address into a TRP-formatted string
//	@Summary		Encode travel address
//	@Description	Encodes a travel address into a TRP-formatted string
//	@Tags			Utility
//	@ID				encodeTravelAddress
//	@Accept			json
//	@Produce		json
//	@Param			travelAddress	body		api.TravelAddress	true	"Travel address to encode"
//	@Success		200				{object}	api.TravelAddress	"Successful operation"
//	@Failure		400				{object}	api.Reply			"Invalid input"
//	@Router			/v1/utilities/travel-address/encode [post]
func (s *Server) EncodeTravelAddress(c *gin.Context) {
	var (
		err error
		in  *api.TravelAddress
	)

	in = &api.TravelAddress{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse travel address encoding data"))
		return
	}

	if err = in.ValidateEncode(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	if in.Encoded, err = traddr.Encode(in.Decoded); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	c.JSON(http.StatusOK, in)
}

// DecodeTravelAddress - Decodes a TRP-formatted travel address into its component parts
//	@Summary		Decode travel address
//	@Description	Decodes a TRP-formatted travel address into its component parts
//	@Tags			Utility
//	@ID				decodeTravelAddress
//	@Accept			json
//	@Produce		json
//	@Param			travelAddress	body		api.TravelAddress	true	"TRP-formatted travel address to decode"
//	@Success		200				{object}	api.TravelAddress	"Successful operation"
//	@Failure		400				{object}	api.Reply			"Invalid input"
//	@Router			/v1/utilities/travel-address/decode [post]
func (s *Server) DecodeTravelAddress(c *gin.Context) {
	var (
		err error
		in  *api.TravelAddress
	)

	in = &api.TravelAddress{}
	if err = c.BindJSON(in); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error("could not parse travel address encoding data"))
		return
	}

	if err = in.ValidateDecode(); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	if in.Decoded, err = traddr.Decode(in.Encoded); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	c.JSON(http.StatusOK, in)
}
