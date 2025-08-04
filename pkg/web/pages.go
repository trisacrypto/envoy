package web

import (
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/enum"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/htmx"
	"github.com/trisacrypto/envoy/pkg/web/scene"
	"go.rtnl.ai/ulid"

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

// ForgotPasswordPage displays the reset password form for the UI so that the user can
// enter their email address and receive a password reset link.
func (s *Server) ForgotPasswordPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/reset/forgot.html", scene.New(c))
}

// ForgotPasswordSentPage displays the success page for the reset password
// request. Rather than using an HTMX partial, we redirect the user to this page to
// ensure they close the window (e.g. if they were logged in) and to prevent a conflict
// when cookies are reset during the password reset process.
func (s *Server) ForgotPasswordSentPage(c *gin.Context) {
	c.HTML(http.StatusOK, "auth/reset/sent.html", scene.New(c))
}

// ResetPasswordPage allows the user to enter a new password if the reset password link
// is verified and change their password as necessary.
func (s *Server) ResetPasswordPage(c *gin.Context) {
	// Read the token string from the URL parameters.
	in := &api.URLVerification{}
	if err := c.BindQuery(in); err != nil {
		// Debug an error here but don't worry about erroring; the token will be
		// blank and will cause a validation error when the form is submitted.
		log.Debug().Err(err).Msg("could not parse query string")
	}

	// Set the token into a cookie so that it can be parsed when the form is submitted.
	// A cookie is more secure than using a hidden form because it cannot be accessed
	// by XSS attacks (though it could be fetched by the window.location object).
	// NOTE: no verification is performed here, just on reset-password.
	s.SetResetPasswordTokenCookie(c, in.Token)

	// Render the verify and change page
	c.HTML(http.StatusOK, "auth/reset/password.html", scene.New(c))
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

func (s *Server) SendForm(c *gin.Context) {
	var (
		in       *api.RoutingQuery
		err      error
		protocol enum.Protocol
		routing  *api.Routing
	)

	if protocol, err = enum.ParseProtocol(c.Param("protocol")); err != nil {
		s.NotFound(c)
		return
	}

	in = &api.RoutingQuery{}
	if err := c.BindQuery(in); err != nil {
		// Log the error but don't inform user; the form just will not have a
		// counterparty record pre-filled.
		c.Error(err)
	}

	if routing, err = s.CounterpartyRouting(c.Request.Context(), in); err != nil {
		// Log the error but don't inform user; the form just will not have a
		// counterparty record pre-filled.
		c.Error(err)
	}

	// If routing is nil, then we need to create an empty routing object otherwise
	// the template will not render the counterparty form.
	if routing == nil {
		routing = &api.Routing{}
	}

	ctx := scene.New(c).With("Protocol", protocol.String()).With("Routing", routing)
	if protocol == enum.ProtocolSunrise {
		ctx["PageTitle"] = "Send a Sunrise Email"
	}

	c.HTML(http.StatusOK, "pages/send/send.html", ctx)
}

func (s *Server) TransactionsAcceptPreview(c *gin.Context) {
	var (
		err error
		out *api.Transaction
	)

	if out, err = s.retrieveTransaction(c); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			s.NotFound(c)
			return
		}

		s.Error(c, err)
		return
	}

	ctx := scene.New(c).WithAPIData(out)
	c.HTML(http.StatusOK, "pages/transactions/accept.html", ctx)
}

func (s *Server) TransactionsRepairPreview(c *gin.Context) {
	var (
		err error
		out *api.Transaction
	)

	if out, err = s.retrieveTransaction(c); err != nil {
		if errors.Is(err, dberr.ErrNotFound) {
			s.NotFound(c)
			return
		}

		s.Error(c, err)
		return
	}

	ctx := scene.New(c).WithAPIData(out)
	c.HTML(http.StatusOK, "pages/transactions/repair.html", ctx)
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

	ctx := scene.New(c).WithToastMessages(c)
	ctx["ID"] = txID

	s.ClearToastMessages(c)
	c.HTML(http.StatusOK, "pages/transactions/detail.html", ctx)
}

//===========================================================================
// Customer Accounts Pages
//===========================================================================

func (s *Server) AccountsListPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard/accounts/list.html", scene.New(c))
}

func (s *Server) AccountDetailPage(c *gin.Context) {
	s.AccountDetailTemplate(c, "pages/accounts/detail.html")
}

func (s *Server) AccountEditPage(c *gin.Context) {
	s.AccountDetailTemplate(c, "pages/accounts/edit.html")
}

func (s *Server) AccountTransfersPage(c *gin.Context) {
	s.AccountDetailTemplate(c, "pages/accounts/transfers.html")
}

func (s *Server) AccountDetailTemplate(c *gin.Context, template string) {
	var (
		err     error
		account *models.Account
		ctx     scene.Scene
	)

	// Retrieve the account from the database and handle errors
	if account, err = s.RetrieveAccount(c); err != nil {
		if errors.Is(err, ErrNotFound) {
			s.NotFound(c)
			return
		}

		s.Error(c, err)
		return
	}

	// Create a scene with the account model
	ctx = scene.New(c).WithAPIData(account)
	c.HTML(http.StatusOK, template, ctx)
}

//===========================================================================
// Counterparty VASP Pages
//===========================================================================

func (s *Server) CounterpartiesListPage(c *gin.Context) {
	ctx := scene.New(c)
	ctx["Source"] = strings.ToLower(c.Query("source"))
	c.HTML(http.StatusOK, "dashboard/counterparties/list.html", ctx)
}

func (s *Server) CounterpartyDetailPage(c *gin.Context) {
	// Get the counterparty ID from the URL path and make available to the template.
	// The counterparty detail is loaded using htmx.
	counterpartyID := c.Param("id")

	// Validate that the counterparty ID is a valid UUID.
	if _, err := ulid.Parse(counterpartyID); err != nil {
		htmx.Redirect(c, http.StatusTemporaryRedirect, "/not-found")
		return
	}

	ctx := scene.New(c)
	ctx["ID"] = counterpartyID

	c.HTML(http.StatusOK, "pages/counterparties/detail.html", ctx)
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
// Audit Log Management Pages
//===========================================================================

func (s *Server) ComplianceAuditLogListPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard/auditlogs/list.html", scene.New(c))
}

func (s *Server) ComplianceAuditLogDetailPage(c *gin.Context) {
	// Get the audit log ID from the URL path and make available to the template.
	// The audit log detail is loaded using htmx.
	logID := c.Param("id")

	// Validate that the audit log ID is a valid UUID.
	if _, err := ulid.Parse(logID); err != nil {
		htmx.Redirect(c, http.StatusTemporaryRedirect, "/not-found")
		return
	}

	ctx := scene.New(c)
	ctx["ID"] = logID

	c.HTML(http.StatusOK, "pages/auditlogs/detail.html", ctx)
}

//===========================================================================
// Node Info Pages
//===========================================================================

func (s *Server) AboutPage(c *gin.Context) {
	var (
		err        error
		localparty *models.Counterparty
	)

	ctx := scene.New(c)
	ctx.Update(scene.Scene{
		"Version":  fmt.Sprintf("%d.%d.%d", pkg.VersionMajor, pkg.VersionMinor, pkg.VersionPatch),
		"Revision": pkg.GitVersion,
		"Release":  fmt.Sprintf("%s-%d", pkg.VersionReleaseLevel, pkg.VersionReleaseNumber),
		"Region":   s.conf.RegionInfo,
		"Config":   s.conf,
		"TRISA":    s.conf.Node,
		"Certificates": map[string]string{
			"CommonName": s.conf.Node.CommonName(),
			"IssuedAt":   s.conf.Node.IssuedAt().Format("Jan 2, 2006 at 15:04:05 MST"),
			"Expires":    s.conf.Node.Expires().Format("Jan 2, 2006 at 15:04:05 MST"),
		},
		"DirectorySync": s.conf.DirectorySync,
	})

	if localparty, err = s.Localparty(c.Request.Context()); err != nil {
		log.Error().Err(err).Msg("could not retrieve counterparty local party information")
	}

	ctx = ctx.WithLocalparty(localparty, err)
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
