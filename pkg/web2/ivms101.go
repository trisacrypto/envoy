package web

import (
	"net/http"

	"github.com/gin-gonic/gin"
	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

func (s *Server) ValidateIVMS101(c *gin.Context) {
	var (
		payload *ivms101.IdentityPayload
		err     error
	)

	payload = &ivms101.IdentityPayload{}
	if err = c.ShouldBindJSON(payload); err != nil {
		c.Error(err)
		c.JSON(http.StatusBadRequest, api.Error(err))
		return
	}

	if err = payload.Validate(); err != nil {
		// Convert IVMS101 validation error into an API field error.
		validationErrors := err.(ivms101.ValidationErrors)
		c.JSON(http.StatusUnprocessableEntity, api.Error(api.ConvertIVMS101Errors(validationErrors)))
		return
	}

	c.JSON(http.StatusOK, payload)
}
