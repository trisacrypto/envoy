package web

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"github.com/trisacrypto/envoy/pkg/web/scene"
)

const (
	ToastCookie              = "toast_messages"
	ToastCookieTTL           = 10 * time.Minute
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

func (s *Server) AddToastMessage(c *gin.Context, heading, message, toastType string) {
	messages := s.ToastMessages(c)
	messages = append(messages, scene.ToastMessage{Heading: heading, Message: message, Type: toastType})
	cookie := messages.MarshalCookie()
	s.SetCookie(c, ToastCookie, cookie, "/", int(ToastCookieTTL.Seconds()), false)
}

func (s *Server) ToastMessages(c *gin.Context) scene.ToastMessages {
	cookie, err := c.Cookie(ToastCookie)
	if err != nil || cookie == "" {
		return nil
	}

	var messages scene.ToastMessages
	messages.UnmarshalCookie(cookie)
	return messages
}

func (s *Server) ClearToastMessages(c *gin.Context) {
	s.ClearCookie(c, ToastCookie, "/", false)
}
