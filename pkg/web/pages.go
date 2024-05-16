package web

import (
	"net/http"

	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/htmx"

	"github.com/gin-gonic/gin"
)

// Home currently renders the primary landing page for the web ui.
func (s *Server) Home(c *gin.Context) {
	htmx.Redirect(c, http.StatusFound, "/transactions")
}

func (s *Server) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{"Version": pkg.Version()})
}

func (s *Server) Logout(c *gin.Context) {
	// Clear the client cookies
	auth.ClearAuthCookies(c, s.conf.Auth.CookieDomain)

	// Send the user to the login page
	htmx.Redirect(c, http.StatusFound, "/login")
}

func (s *Server) Transactions(c *gin.Context) {
	c.HTML(http.StatusOK, "transactions.html", gin.H{"Version": pkg.Version()})
}

func (s *Server) TransactionsAcceptPreview(c *gin.Context) {
	// Get the transaction ID from the URL path and make available to the template.
	id := c.Param("id")
	c.HTML(http.StatusOK, "transactions_accept.html", gin.H{"Version": pkg.Version(), "ID": id})
}

func (s *Server) TransactionsInfo(c *gin.Context) {
	// Get the transaction ID from the URL path and make available to the template.
	id := c.Param("id")
	c.HTML(http.StatusOK, "transactions_info.html", gin.H{"Version": pkg.Version(), "ID": id})
}

func (s *Server) Accounts(c *gin.Context) {
	c.HTML(http.StatusOK, "accounts.html", gin.H{"Version": pkg.Version()})
}

func (s *Server) CounterpartyVasps(c *gin.Context) {
	c.HTML(http.StatusOK, "counterparty.html", gin.H{"Version": pkg.Version()})
}

func (s *Server) SendEnvelopeForm(c *gin.Context) {
	c.HTML(http.StatusOK, "send_envelope.html", gin.H{"Version": pkg.Version()})
}

func (s *Server) TravelAddressUtility(c *gin.Context) {
	c.HTML(http.StatusOK, "traveladdress.html", gin.H{"Version": pkg.Version()})
}
