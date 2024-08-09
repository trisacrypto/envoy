package web

import (
	"fmt"
	"net/http"

	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"github.com/trisacrypto/envoy/pkg/web/scene"

	"github.com/gin-gonic/gin"
)

// Home currently renders the primary landing page for the web ui.
func (s *Server) Home(c *gin.Context) {
	htmx.Redirect(c, http.StatusFound, "/transactions")
}

func (s *Server) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", scene.New(c))
}

func (s *Server) Logout(c *gin.Context) {
	// Clear the client cookies
	auth.ClearAuthCookies(c, s.conf.Web.Auth.CookieDomain)

	// Send the user to the login page
	htmx.Redirect(c, http.StatusFound, "/login")
}

func (s *Server) About(c *gin.Context) {
	ctx := scene.New(c)
	ctx.Update(scene.Scene{
		"Version":       fmt.Sprintf("%d.%d.%d", pkg.VersionMajor, pkg.VersionMinor, pkg.VersionPatch),
		"Revision":      pkg.GitVersion,
		"Release":       fmt.Sprintf("%s-%d", pkg.VersionReleaseLevel, pkg.VersionReleaseNumber),
		"Region":        s.conf.RegionInfo,
		"Config":        s.conf,
		"TRISA":         s.conf.Node,
		"DirectorySync": s.conf.DirectorySync,
	})

	c.HTML(http.StatusOK, "about.html", ctx)
}

func (s *Server) Transactions(c *gin.Context) {
	c.HTML(http.StatusOK, "transactions.html", scene.New(c))
}

func (s *Server) TransactionsAcceptPreview(c *gin.Context) {
	// Get the transaction ID from the URL path and make available to the template.
	ctx := scene.New(c)
	ctx["ID"] = c.Param("id")

	c.HTML(http.StatusOK, "transactions_accept.html", ctx)
}

func (s *Server) TransactionsRepairPreview(c *gin.Context) {
	// Get the transaction ID from the URL path and make available to the template.
	ctx := scene.New(c)
	ctx["ID"] = c.Param("id")

	c.HTML(http.StatusOK, "transactions_repair.html", ctx)
}

func (s *Server) TransactionsInfo(c *gin.Context) {
	// Get the transaction ID from the URL path and make available to the template.
	ctx := scene.New(c)
	ctx["ID"] = c.Param("id")

	c.HTML(http.StatusOK, "transactions_info.html", ctx)
}

func (s *Server) Accounts(c *gin.Context) {
	c.HTML(http.StatusOK, "accounts.html", scene.New(c))
}

func (s *Server) CounterpartyVasps(c *gin.Context) {
	c.HTML(http.StatusOK, "counterparty.html", scene.New(c))
}

func (s *Server) SendEnvelopeForm(c *gin.Context) {
	c.HTML(http.StatusOK, "send_envelope.html", scene.New(c))
}

func (s *Server) TravelAddressUtility(c *gin.Context) {
	c.HTML(http.StatusOK, "traveladdress.html", scene.New(c))
}

func (s *Server) UsersManagement(c *gin.Context) {
	c.HTML(http.StatusOK, "users_management.html", scene.New(c))
}

func (s *Server) UserProfile(c *gin.Context) {
	c.HTML(http.StatusOK, "user_profile.html", scene.New(c))
}
