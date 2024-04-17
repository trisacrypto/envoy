package web

import (
	"errors"
	"net/http"

	"github.com/trisacrypto/envoy/pkg"

	"github.com/gin-gonic/gin"
)

var (
	ErrNoTRISAEndpoint   = errors.New("cannot construct trisa travel address: no trisa endpoint defined")
	ErrNoLocalCommonName = errors.New("invalid configuration: no common name in trisa endpoint configuration")
	ErrNoLocalparty      = errors.New("could not lookup local vasp counterparty from database, please try again later")
	ErrNotAccepted       = errors.New("the accepted formats are not offered by the server")
)

// Renders the "not found page"
func (s *Server) NotFound(c *gin.Context) {
	c.HTML(http.StatusNotFound, "404.html", gin.H{"Version": pkg.Version()})
}

// Renders the "invalid action page"
func (s *Server) NotAllowed(c *gin.Context) {
	c.HTML(http.StatusMethodNotAllowed, "405.html", gin.H{"Version": pkg.Version()})
}
