package web

import (
	"io/fs"
	"net/http"
	"time"

	"github.com/trisacrypto/envoy/pkg/logger"
	"github.com/trisacrypto/envoy/pkg/metrics"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	permiss "github.com/trisacrypto/envoy/pkg/web/auth/permissions"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// Sets up the server's middleware and routes.
func (s *Server) setupRoutes() (err error) {
	// Setup HTML template renderer
	templateFiles, _ := fs.Sub(content, "templates")
	if s.router.HTMLRender, err = NewRender(templateFiles); err != nil {
		return err
	}

	// Create CORS configuration
	corsConf := cors.Config{
		AllowMethods:     []string{"GET", "HEAD"},
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "Authorization", "X-CSRF-TOKEN"},
		AllowOrigins:     []string{s.conf.Web.Origin},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}

	// Application Middleware
	// NOTE: ordering is important to how middleware is handled
	middlewares := []gin.HandlerFunc{
		// Logging should be on the outside so we can record the correct latency of requests
		// NOTE: logging panics will not recover
		logger.GinLogger("web"),

		// Panic recovery middleware
		gin.Recovery(),

		// CORS configuration allows the front-end to make cross-origin requests
		cors.New(corsConf),

		// Maintenance mode middleware to return unavailable
		s.Maintenance(),

		// Web API and UI Enabled middleware
		s.APIEnabled(),

		s.UIEnabled(),
	}

	// Kubernetes liveness probes added before middleware.
	s.router.GET("/healthz", s.Healthz)
	s.router.GET("/livez", s.Healthz)
	s.router.GET("/readyz", s.Readyz)

	// Prometheus metrics handler added before middleware.
	// Note metrics will be served at /metrics
	metrics.Routes(s.router)

	// Add the middleware to the router
	for _, middleware := range middlewares {
		if middleware != nil {
			s.router.Use(middleware)
		}
	}

	// NotFound and NotAllowed routes
	s.router.NoRoute(s.NotFound)
	s.router.NoMethod(s.NotAllowed)

	// Error routes
	s.router.GET("/not-found", s.NotFound)
	s.router.GET("/not-allowed", s.NotAllowed)
	s.router.GET("/error", s.InternalError)

	// Static Files
	staticFiles, _ := fs.Sub(content, "static")
	s.router.StaticFS("/static", http.FS(staticFiles))

	// Authentication Middleware
	authenticate := auth.Authenticate(s.issuer)
	sunriseAuth := s.SunriseAuthenticate(s.issuer)

	// Authorization Helper
	authorize := func(permissions ...permiss.Permission) gin.HandlerFunc {
		perms := permiss.Permissions(permissions)
		return auth.Authorize(perms.String()...)
	}

	// Web UI Routes (Dashboards and Pages) - Unauthenticated
	s.router.GET("/login", s.LoginPage)
	s.router.GET("/logout", s.Logout)
	s.router.GET("/reset-password", s.ResetPasswordPage)
	s.router.GET("/reset-password/success", s.ResetPasswordSuccessPage)
	//TODO: "/reset-password/verification" for when they click the link

	// Web UI Routes (Dashboards and Pages) - Authenticated
	ui := s.router.Group("", authenticate)
	{
		ui.GET("/", s.Home)
		ui.GET("/about", s.AboutPage)
		ui.GET("/settings", s.SettingsPage)
		ui.GET("/counterparties", s.CounterpartiesListPage)
		ui.GET("/counterparties/:id", s.CounterpartyDetailPage)
		ui.GET("/users", s.UsersListPage)
		ui.GET("/apikeys", s.APIKeysListPage)
		ui.GET("/utilities/travel-address", s.TravelAddressUtility)

		// Accounts Pages
		accounts := ui.Group("/accounts")
		{
			accounts.GET("", authorize(permiss.AccountsView), s.AccountsListPage)
			accounts.GET("/:id", authorize(permiss.AccountsView), s.AccountDetailPage)
			accounts.GET("/:id/edit", authorize(permiss.AccountsManage), s.AccountEditPage)
			accounts.GET("/:id/transfers", authorize(permiss.TravelRuleView), s.AccountTransfersPage)
		}

		// Profile Pages
		profile := ui.Group("/profile")
		{
			profile.GET("", s.UserProfile)
			profile.GET("/account", s.UserAccount)
		}

		// Transactions Pages
		transactions := ui.Group("/transactions")
		{
			transactions.GET("", s.TransactionsListPage)
			transactions.GET("/:id", s.TransactionDetailPage)
			transactions.GET("/:id/accept", s.TransactionsAcceptPreview)
			transactions.GET("/:id/repair", s.TransactionsRepairPreview)
		}

		// Send Secure Message Forms
		send := ui.Group("/send", authorize(permiss.TravelRuleManage))
		{
			send.GET("", s.AvailableProtocols)
			send.GET("/trisa", s.SendTRISAForm)
			send.GET("/trp", s.SendTRPForm)
			// The send sunrise message page for authenticated envoy users.
			send.GET("/sunrise", s.SendSunriseForm)
		}
	}

	// Swagger documentation with Swagger UI hosted from a CDN
	// NOTE: should documentation require authentication?
	s.router.GET("/v1/docs/openapi.:ext", s.OpenAPI())
	s.router.GET("/v1/docs", s.APIDocs)

	// Sunrise Routes (can be disabled by the middleware)
	// These routes are intended for external users to access a sunrise message
	sunrise := s.router.Group("/sunrise", s.SunriseEnabled())
	{
		// Logs in a sunrise user to allow the external user to be sunrise authenticated.
		sunrise.GET("/verify", s.VerifySunriseUser)

		// The review form and handlers for external sunrise users.
		sunrise.GET("/review", sunriseAuth, s.SunriseMessageReview)
		sunrise.POST("/reject", sunriseAuth, s.SunriseMessageReject)
		sunrise.POST("/accept", sunriseAuth, s.SunriseMessageAccept)
		sunrise.GET("/download", sunriseAuth, s.SunriseMessageDownload)
	}

	// API Routes (Including Content Negotiated Partials)
	v1 := s.router.Group("/v1")
	{
		// Status/Heartbeat endpoint
		v1.GET("/status", s.Status)

		// Database Statistics
		v1.GET("/dbinfo", authenticate, authorize(permiss.ConfigView), s.DBInfo)

		// Authentication endpoints
		v1.POST("/login", s.Login)
		v1.POST("/authenticate", s.Authenticate)
		v1.POST("/reauthenticate", s.Reauthenticate)

		// User Profile Management
		v1.POST("/reset-password", s.ResetPassword)
		v1.POST("/change-password", authenticate, s.ChangePassword)

		// Accounts Resource
		accounts := v1.Group("/accounts", authenticate)
		{
			accounts.GET("", authorize(permiss.AccountsView), s.ListAccounts)
			accounts.POST("", authorize(permiss.AccountsManage), s.CreateAccount)
			accounts.GET("/lookup", authorize(permiss.AccountsView), s.LookupAccount)
			accounts.GET("/:id", authorize(permiss.AccountsView), s.AccountDetail)
			accounts.PUT("/:id", authorize(permiss.AccountsManage), s.UpdateAccount)
			accounts.DELETE("/:id", authorize(permiss.AccountsManage), s.DeleteAccount)

			accounts.GET("/:id/transfers", authorize(permiss.TravelRuleView), s.AccountTransfers)
			accounts.GET("/:id/qrcode", authorize(permiss.AccountsView), s.AccountQRCode)

			// CryptoAddress Resource (nested on Accounts)
			ca := accounts.Group("/:id/crypto-addresses")
			{
				ca.GET("", authorize(permiss.AccountsView), s.ListCryptoAddresses)
				ca.POST("", authorize(permiss.AccountsManage), s.CreateCryptoAddress)
				ca.GET("/:cryptoAddressID", authorize(permiss.AccountsView), s.CryptoAddressDetail)
				ca.PUT("/:cryptoAddressID", authorize(permiss.AccountsManage), s.UpdateCryptoAddress)
				ca.DELETE("/:cryptoAddressID", authorize(permiss.AccountsManage), s.DeleteCryptoAddress)
				ca.GET("/:cryptoAddressID/qrcode", authorize(permiss.AccountsView), s.CryptoAddressQRCode)
			}
		}

		// Transactions Resource
		transactions := v1.Group("/transactions", authenticate)
		{
			transactions.GET("", authorize(permiss.TravelRuleView), s.ListTransactions)
			transactions.POST("", authorize(permiss.TravelRuleManage), s.CreateTransaction)
			transactions.GET("/:id", authorize(permiss.TravelRuleView), s.TransactionDetail)
			transactions.PUT("/:id", authorize(permiss.TravelRuleManage), s.UpdateTransaction)
			transactions.DELETE("/:id", authorize(permiss.TravelRuleDelete), s.DeleteTransaction)

			// Primarily UI methods but are also API Helper Methods
			transactions.POST("/prepare", authorize(permiss.TravelRuleManage), s.PrepareTransaction)
			transactions.POST("/send-prepared", authorize(permiss.TravelRuleManage), s.SendPreparedTransaction)

			// Export method to export transactions to a CSV
			transactions.GET("/export", authorize(permiss.TravelRuleManage), s.ExportTransactions)

			// Transaction specific actions
			transactions.POST("/:id/send", authorize(permiss.TravelRuleManage), s.SendEnvelopeForTransaction)
			transactions.GET("/:id/payload", authorize(permiss.TravelRuleView), s.LatestPayloadEnvelope)
			transactions.GET("/:id/accept", authorize(permiss.TravelRuleView), s.AcceptTransactionPreview)
			transactions.POST("/:id/accept", authorize(permiss.TravelRuleManage), s.AcceptTransaction)
			transactions.POST("/:id/reject", authorize(permiss.TravelRuleManage), s.RejectTransaction)
			transactions.GET("/:id/repair", authorize(permiss.TravelRuleView), s.RepairTransactionPreview)
			transactions.POST("/:id/repair", authorize(permiss.TravelRuleManage), s.RepairTransaction)
			transactions.POST("/:id/archive", authorize(permiss.TravelRuleManage), s.ArchiveTransaction)
			transactions.POST("/:id/unarchive", authorize(permiss.TravelRuleManage), s.UnarchiveTransaction)

			// SecureEnvelope Resource (nested on Transactions)
			se := transactions.Group("/:id/secure-envelopes")
			{
				se.GET("", authorize(permiss.TravelRuleView), s.ListSecureEnvelopes)
				se.GET("/:envelopeID", authorize(permiss.TravelRuleView), s.SecureEnvelopeDetail)
			}
		}

		// Counterparties Resource
		counterparties := v1.Group("/counterparties", authenticate)
		{
			counterparties.GET("", authorize(permiss.CounterpartiesView), s.ListCounterparties)
			counterparties.POST("", authorize(permiss.CounterpartiesManage), s.CreateCounterparty)
			counterparties.GET("/search", authorize(permiss.CounterpartiesView), s.SearchCounterparties)
			counterparties.GET("/:id", authorize(permiss.CounterpartiesView), s.CounterpartyDetail)
			counterparties.GET("/:id/edit", authorize(permiss.CounterpartiesManage), s.UpdateCounterpartyPreview)
			counterparties.PUT("/:id", authorize(permiss.CounterpartiesManage), s.UpdateCounterparty)
			counterparties.DELETE("/:id", authorize(permiss.CounterpartiesManage), s.DeleteCounterparty)

			// Contacts Resource (nested on Counterparty)
			contacts := counterparties.Group("/:id/contacts")
			{
				contacts.GET("", authorize(permiss.CounterpartiesView), s.ListContacts)
				contacts.POST("", authorize(permiss.CounterpartiesManage), s.CreateContact)
				contacts.GET("/:contactID", authorize(permiss.CounterpartiesView), s.ContactDetail)
				contacts.PUT("/:contactID", authorize(permiss.CounterpartiesManage), s.UpdateContact)
				contacts.DELETE("/:contactID", authorize(permiss.CounterpartiesManage), s.DeleteContact)
			}
		}

		// Users Resource
		users := v1.Group("/users", authenticate)
		{
			users.GET("", authorize(permiss.UsersView), s.ListUsers)
			users.POST("", authorize(permiss.UsersManage), s.CreateUser)
			users.GET("/:id", authorize(permiss.UsersView), s.UserDetail)
			users.PUT("/:id", authorize(permiss.UsersManage), s.UpdateUser)
			users.DELETE("/:id", authorize(permiss.UsersManage), s.DeleteUser)
			users.POST("/:id/password", authorize(permiss.UsersManage), s.ChangeUserPassword)
		}

		// Profile Resource: Similar to user resource but for logged in user and does
		// not require the users:manage permission for access.
		// NOTE: this is undocumented in the API since it is only intended for the UI.
		profile := v1.Group("/profile", authenticate)
		{
			profile.GET("", s.ProfileDetail)
			profile.PUT("", s.UpdateProfile)
			profile.DELETE("", s.DeleteProfile)
			profile.POST("/password", s.ChangeProfilePassword)
		}

		// API Keys Resource
		apikeys := v1.Group("/apikeys", authenticate)
		{
			apikeys.GET("", authorize(permiss.APIKeysView), s.ListAPIKeys)
			apikeys.POST("", authorize(permiss.APIKeysManage), s.CreateAPIKey)
			apikeys.GET("/:id", authorize(permiss.APIKeysView), s.APIKeyDetail)
			apikeys.GET("/:id/edit", authorize(permiss.APIKeysManage), s.UpdateAPIKeyPreview)
			apikeys.PUT("/:id", authorize(permiss.APIKeysManage), s.UpdateAPIKey)
			apikeys.DELETE("/:id", authorize(permiss.APIKeysRevoke), s.DeleteAPIKey)
		}

		// Utilities
		utils := v1.Group("/utilities", authenticate)
		{
			utils.POST("/travel-address/encode", s.EncodeTravelAddress)
			utils.POST("/travel-address/decode", s.DecodeTravelAddress)
			utils.POST("ivms101-validator", s.ValidateIVMS101)
		}
	}

	return nil
}
