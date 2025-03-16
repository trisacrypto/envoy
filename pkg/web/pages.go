package web

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"github.com/trisacrypto/envoy/pkg/web/scene"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// Home is the root landing page of the server. If the request is for the HTML UI -
// then it redirects to the transactions inbox page. If the request is an API request,
// it redirects to the API documentation.
func (s *Server) Home(c *gin.Context) {
	switch c.NegotiateFormat(binding.MIMEHTML, binding.MIMEJSON) {
	case binding.MIMEHTML:
		htmx.Redirect(c, http.StatusFound, "/transactions")
	case binding.MIMEJSON:
		c.JSON(http.StatusNotFound, api.NotFound)
	default:
		c.AbortWithError(http.StatusNotAcceptable, ErrNotAccepted)
	}
}

// LoginPage displays the login form for the UI so that the user can enter their
// account credentials and access the compliance interface.
func (s *Server) LoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/login/login.html", scene.New(c))
}

// ResetPasswordPage displays the reset password form for the UI so that the user can
// enter their email address and receive a password reset link.
func (s *Server) ResetPasswordPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/reset/password.html", scene.New(c))
}

// ResetPasswordSuccessPage displays the confirmation message after a user has
// requested a password reset link. This page is displayed no matter the outcome.
func (s *Server) ResetPasswordSuccessPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/reset/success.html", scene.New(c))
}

//===========================================================================
// Transactions Pages
//===========================================================================

func (s *Server) TransactionsListPage(c *gin.Context) {
	// Count the number of transactions in the database (ignore errors)
	counts, _ := s.store.CountTransactions(c.Request.Context())

	ctx := scene.New(c).WithAPIData(counts)
	ctx["Archives"] = strings.ToLower(c.Query("archives"))

	c.HTML(http.StatusOK, "dashboard/transactions/list.html", ctx)
}

func (s *Server) AvailableProtocols(c *gin.Context) {
	ctx := scene.New(c)
	ctx["TRISAEnabled"] = s.conf.Node.Enabled
	ctx["TRPEnabled"] = s.conf.TRP.Enabled
	c.HTML(http.StatusOK, "pages/send/choose.html", ctx)
}

func (s *Server) SendTRISAForm(c *gin.Context) {
	ctx := scene.New(c)
	ctx["Protocol"] = "trisa"
	c.HTML(http.StatusOK, "pages/send/send.html", ctx)
}

func (s *Server) SendTRPForm(c *gin.Context) {
	ctx := scene.New(c)
	ctx["Protocol"] = "trp"
	c.HTML(http.StatusOK, "pages/send/send.html", ctx)
}

func (s *Server) SendSunriseForm(c *gin.Context) {
	ctx := scene.New(c)
	ctx["Protocol"] = "sunrise"
	ctx["PageTitle"] = "Send a Sunrise Email"
	c.HTML(http.StatusOK, "pages/send/send.html", ctx)
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

func (s *Server) TransactionDetailPage(c *gin.Context) {
	// Get the transaction ID from the URL path and make available to the template.
	// The transaction detail is loaded using htmx.
	txID := c.Param("id")

	// Validate that the transaction ID is a valid UUID.
	if _, err := uuid.Parse(txID); err != nil {
		htmx.Redirect(c, http.StatusTemporaryRedirect, "/not-found")
		return
	}

	ctx := scene.New(c)
	ctx["ID"] = txID
	c.HTML(http.StatusOK, "pages/transactions/detail.html", ctx)
}

//===========================================================================
// Customer Accounts Pages
//===========================================================================

func (s *Server) AccountsListPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard/accounts/list.html", scene.New(c))
}

//===========================================================================
// Counterparty VASP Pages
//===========================================================================

func (s *Server) CounterpartiesListPage(c *gin.Context) {
	ctx := scene.New(c)
	ctx["Source"] = strings.ToLower(c.Query("source"))
	c.HTML(http.StatusOK, "dashboard/counterparties/list.html", ctx)
}

//===========================================================================
// Utility Pages
//===========================================================================

func (s *Server) TravelAddressUtility(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/utilities/traveladdress.html", scene.New(c))
}

//===========================================================================
// User Management Pages
//===========================================================================

func (s *Server) UsersListPage(c *gin.Context) {
	ctx := scene.New(c)
	ctx["Role"] = strings.ToLower(c.Query("role"))
	c.HTML(http.StatusOK, "dashboard/users/list.html", ctx)
}

//===========================================================================
// API Key Management Pages
//===========================================================================

func (s *Server) APIKeysListPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard/apikeys/list.html", scene.New(c))
}

//===========================================================================
// Node Info Pages
//===========================================================================

func (s *Server) AboutPage(c *gin.Context) {
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

	c.HTML(http.StatusOK, "pages/settings/about.html", ctx)
}

func (s *Server) SettingsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/settings/settings.html", scene.New(c))
}

//===========================================================================
// User Profile Pages
//===========================================================================

func (s *Server) UserProfile(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/profile/detail.html", scene.New(c))
}

func (s *Server) UserAccount(c *gin.Context) {
	c.HTML(http.StatusOK, "pages/profile/account.html", scene.New(c))
}
