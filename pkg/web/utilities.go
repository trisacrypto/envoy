package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

// Convert a URL or other data into a travel address.
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

// Decode a travel address into a URL or its other contents.
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
