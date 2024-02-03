package web

import (
	"net/http"
	"self-hosted-node/pkg/config"
	"time"

	"github.com/gin-gonic/gin"
)

// Create a new web server that serves the compliance and admin web user interface.
func New(conf config.WebConfig) (s *Server, err error) {
	if err = conf.Validate(); err != nil {
		return nil, err
	}

	s = &Server{
		conf: conf,
	}

	// If not enabled, return just the server stub
	if !conf.Enabled {
		return s, nil
	}

	// Configure the gin router if enabled
	s.router = gin.New()
	s.router.RedirectTrailingSlash = true
	s.router.RedirectFixedPath = false
	s.router.HandleMethodNotAllowed = true
	s.router.ForwardedByClientIP = true
	s.router.UseRawPath = false
	s.router.UnescapePathValues = true
	if err = s.setupRoutes(); err != nil {
		return nil, err
	}

	// Create the http server if enabled
	s.srv = &http.Server{
		Addr:         s.conf.BindAddr,
		Handler:      s.router,
		ErrorLog:     nil,
		ReadTimeout:  20 * time.Second,
		WriteTimeout: 20 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return s, nil
}

// Debug returns a server that uses the specified http server instead of creating one.
// This function is primarily used to create test servers easily.
func Debug(conf config.WebConfig, srv *http.Server) (s *Server, err error) {
	if s, err = New(conf); err != nil {
		return nil, err
	}

	// Replace the http server with the one specified
	s.srv = nil
	s.srv = srv
	s.srv.Handler = s.router
	return s, nil
}

// Home currently renders the primary landing page for the web ui.
// TODO: replace with dashboard or redirect as necessary.
func (s *Server) Home(c *gin.Context) {
	c.HTML(http.StatusOK, "index.html", gin.H{})
}
