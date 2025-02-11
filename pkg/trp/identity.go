package trp

import (
	"encoding/pem"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/trisa/pkg/openvasp/trp/v3"
	"github.com/trisacrypto/trisa/pkg/trust"
)

func (s *Server) Identity(c *gin.Context) {
	c.JSON(http.StatusOK, s.identity)
}

func (s *Server) initializeIdentity() {
	s.identity = trp.Identity{
		Name: s.conf.TRP.Identity.VASPName,
		LEI:  s.conf.TRP.Identity.LEI,
	}

	if s.identity.Name == "" {
		s.identity.Name = s.conf.Organization
	}

	if s.conf.TRP.UseMTLS {
		// NOTE: ignoring errors assuming that mTLS has already been configured.
		var certs *trust.Provider
		switch {
		case s.conf.TRP.Certs != "":
			certs, _ = s.conf.TRP.LoadCerts()
		case s.conf.Node.Certs != "":
			certs, _ = s.conf.Node.LoadCerts()
		}

		x509, _ := certs.GetLeafCertificate()
		block := &pem.Block{Type: "CERTIFICATE", Bytes: x509.Raw}
		s.identity.X509 = string(pem.EncodeToMemory(block))
	}
}
