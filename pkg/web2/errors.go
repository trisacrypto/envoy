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
	ErrSunriseSubject    = errors.New("invalid subject type for sunrise review")
	ErrMissingID         = errors.New("id required for this resource")
	ErrIDMismatch        = errors.New("resource id does not match target")
)

// Logs the error with c.Error and negotiates the response. If HTML is requested by the
// Accept header, then a 500 error page is displayed. If JSON is requested, then the
// error is rendered as a JSON response. If a non error is passed as err then no error
// is logged to the context and it is treated as a message to the user.
// TODO: add the error to the scene and the support email if any.
func (s *Server) Error(c *gin.Context, err error) {
	c.Error(err)
	c.Negotiate(http.StatusInternalServerError, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/500.html",
		HTMLData: scene.New(c).Error(err).WithEmail(s.conf.Email.SupportEmail),
		JSONData: api.Error(err),
	})
}

// Renders the "not found page"
func (s *Server) NotFound(c *gin.Context) {
	c.Negotiate(http.StatusNotFound, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/404.html",
		HTMLData: nil,
		JSONData: api.NotFound,
	})
}

// Renders the "invalid action page"
func (s *Server) NotAllowed(c *gin.Context) {
	c.Negotiate(http.StatusMethodNotAllowed, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/405.html",
		HTMLData: nil,
		JSONData: api.NotAllowed,
	})
}

// Renders the "internal server error page"
func (s *Server) InternalError(c *gin.Context) {
	c.Negotiate(http.StatusInternalServerError, gin.Negotiate{
		Offered:  []string{binding.MIMEJSON, binding.MIMEHTML},
		HTMLName: "errors/status/500.html",
		HTMLData: scene.New(c).Error(nil).WithEmail(s.conf.Email.SupportEmail),
		JSONData: api.InternalError,
	})
}
