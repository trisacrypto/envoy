package web

import (
	"net/http"
	"self-hosted-node/pkg"

	"github.com/gin-gonic/gin"
)

// Home currently renders the primary landing page for the web ui.
// TODO: replace with dashboard or redirect as necessary.
func (s *Server) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{"Version": pkg.Version()})
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

func (s *Server) AuditLog(c *gin.Context) {
	c.HTML(http.StatusOK, "audit.html", gin.H{"Version": pkg.Version()})
}
