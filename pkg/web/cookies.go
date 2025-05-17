package web

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/web/auth"
)

const (
	ResetPasswordTokenCookie = "reset_password_token"
	ResetPasswordPath        = "/v1/reset-password"
	ResetPasswordTokenTTL    = 15 * time.Minute
)

func (s *Server) SetCookie(c *gin.Context, name, value, path string, maxAge int, httpOnly bool) {
	domain := s.conf.Web.Auth.CookieDomain
	secure := !auth.IsLocalhost(domain)

	c.SetCookie(name, value, maxAge, path, domain, secure, httpOnly)
}

func (s *Server) ClearCookie(c *gin.Context, name, path string, httpOnly bool) {
	s.SetCookie(c, name, "", path, -1, httpOnly)
}

func (s *Server) SetResetPasswordTokenCookie(c *gin.Context, token string) {
	s.SetCookie(c, ResetPasswordTokenCookie, token, ResetPasswordPath, int(ResetPasswordTokenTTL.Seconds()), true)
}

func (s *Server) ClearResetPasswordTokenCookie(c *gin.Context) {
	s.ClearCookie(c, ResetPasswordTokenCookie, ResetPasswordPath, true)
}

func setToastCookie(c *gin.Context, name, value, path, domain string) {
	secure := !auth.IsLocalhost(domain)
	c.SetCookie(name, value, 1, path, domain, secure, false)
}
