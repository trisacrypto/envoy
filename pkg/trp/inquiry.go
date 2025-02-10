package trp

import (
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func (s *Server) Inquiry(c *gin.Context) {
	log.Info().Msg("TRP inquiry received")
}

func (s *Server) Confirmation(c *gin.Context) {
	log.Info().Msg("TRP confirmation received")
}
