package web

import (
	"errors"
	"net/http"

	api "github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/scene"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

var (
	ErrNoTRISAEndpoint   = errors.New("cannot construct trisa travel address: no trisa endpoint defined")
	ErrNoLocalCommonName = errors.New("invalid configuration: no common name in trisa endpoint configuration")
	ErrNoLocalparty      = errors.New("could not lookup local vasp counterparty from database, please try again later")
	ErrNotAccepted       = errors.New("the accepted formats are not offered by the server")
	ErrNoPublicKey       = errors.New("no public key associated with secure envelope")
)

// Renders the "not found page"
func (s *Server) NotFound(c *gin.Context) {
	c.Negotiate(http.StatusNotFound, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "404.html",
		HTMLData: scene.New(c),
		JSONData: api.NotFound,
	})
}

// Renders the "invalid action page"
func (s *Server) NotAllowed(c *gin.Context) {
	c.Negotiate(http.StatusNotFound, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "405.html",
		HTMLData: scene.New(c),
		JSONData: api.NotAllowed,
	})
}
