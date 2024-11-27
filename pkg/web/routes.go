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
	includes := []string{"layouts/*.html", "components/*.html"}
	if s.router.HTMLRender, err = NewRender(templateFiles, "*.html", includes...); err != nil {
		return err
	}

	// NOTE: partials can't have the same names as top-level pages
	s.router.HTMLRender.(*Render).AddPattern(templateFiles, "partials/*/*.html")

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

	// Static Files
	staticFiles, _ := fs.Sub(content, "static")
	s.router.StaticFS("/static", http.FS(staticFiles))

	// Authentication Middleware
	authenticate := auth.Authenticate(s.issuer)

	// Authorization Helper
	authorize := func(permissions ...permiss.Permission) gin.HandlerFunc {
		perms := permiss.Permissions(permissions)
		return auth.Authorize(perms.String()...)
	}

	// Web UI Routes (Pages)
	s.router.GET("/", authenticate, s.Home)
	s.router.GET("/login", s.LoginPage)
	s.router.GET("/logout", s.Logout)
	s.router.GET("/about", authenticate, s.About)
	s.router.GET("/profile", authenticate, s.UserProfile)
	s.router.GET("/transactions", authenticate, s.Transactions)
	s.router.GET("/transactions/:id/accept", authenticate, s.TransactionsAcceptPreview)
	s.router.GET("/transactions/:id/repair", authenticate, s.TransactionsRepairPreview)
	s.router.GET("/transactions/:id/info", authenticate, s.TransactionsInfo)
	s.router.GET("/send-envelope", authenticate, s.SendEnvelopeForm)
	s.router.GET("/accounts", authenticate, s.Accounts)
	s.router.GET("/counterparty", authenticate, s.CounterpartyVasps)
	s.router.GET("/users", authenticate, s.UsersManagement)
	s.router.GET("/apikeys", authenticate, s.APIKeys)
	s.router.GET("/utilities/travel-address", authenticate, s.TravelAddressUtility)

	// Swagger documentation with Swagger UI hosted from a CDN
	// NOTE: should documentation require authentication?
	s.router.GET("/v1/docs/openapi.:ext", s.OpenAPI)
	s.router.GET("/v1/docs", s.APIDocs)

	// Sunrise Routes (can be disabled by the middleware)
	sunrise := s.router.Group("/sunrise", s.SunriseEnabled())
	{
		sunrise.GET("/verify", s.VerifySunriseUser)
		sunrise.GET("/accept", s.SunriseMessagePreview)
		sunrise.GET("/message", authenticate, authorize(permiss.TravelRuleManage), s.SendMessageForm)
	}

	// API Routes (Including Content Negotiated Partials)
	v1 := s.router.Group("/v1")
	{
		// Status/Heartbeat endpoint
		v1.GET("/status", s.Status)

		// NOTE: uncomment this for debugging only
		// v1.POST("/debug", s.Debug)

		// Authentication endpoints
		v1.POST("/login", s.Login)
		v1.POST("/authenticate", s.Authenticate)
		v1.POST("/reauthenticate", s.Reauthenticate)

		// User Profile Management
		v1.POST("/change-password", authenticate, s.ChangePassword)

		// Accounts Resource
		accounts := v1.Group("/accounts", authenticate)
		{
			accounts.GET("", authorize(permiss.AccountsView), s.ListAccounts)
			accounts.POST("", authorize(permiss.AccountsManage), s.CreateAccount)
			accounts.GET("/:id", authorize(permiss.AccountsView), s.AccountDetail)
			accounts.GET("/:id/edit", authorize(permiss.AccountsManage), s.UpdateAccountPreview)
			accounts.PUT("/:id", authorize(permiss.AccountsManage), s.UpdateAccount)
			accounts.DELETE("/:id", authorize(permiss.AccountsManage), s.DeleteAccount)

			// CryptoAddress Resource (nested on Accounts)
			ca := accounts.Group("/:id/crypto-addresses")
			{
				ca.GET("", authorize(permiss.AccountsView), s.ListCryptoAddresses)
				ca.POST("", authorize(permiss.AccountsManage), s.CreateCryptoAddress)
				ca.GET("/:cryptoAddressID", authorize(permiss.AccountsView), s.CryptoAddressDetail)
				ca.PUT("/:cryptoAddressID", authorize(permiss.AccountsManage), s.UpdateCryptoAddress)
				ca.DELETE("/:cryptoAddressID", authorize(permiss.AccountsManage), s.DeleteCryptoAddress)
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
		}
	}

	return nil
}
