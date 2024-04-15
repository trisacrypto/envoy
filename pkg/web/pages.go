package web

import (
	"net/http"

	"self-hosted-node/pkg"
	"self-hosted-node/pkg/web/auth"
	"self-hosted-node/pkg/web/htmx"

	"github.com/gin-gonic/gin"
)

// Home currently renders the primary landing page for the web ui.
// TODO: replace with dashboard or redirect as necessary.
func (s *Server) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{"Version": pkg.Version()})
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

func (s *Server) Accounts(c *gin.Context) {
	c.HTML(http.StatusOK, "accounts.html", gin.H{"Version": pkg.Version()})
}

func (s *Server) CounterpartyVasps(c *gin.Context) {
	c.HTML(http.StatusOK, "counterparty.html", gin.H{"Version": pkg.Version()})
}

func (s *Server) SendEnvelopeForm(c *gin.Context) {
	c.HTML(http.StatusOK, "send_envelope.html", gin.H{"Version": pkg.Version()})
}
