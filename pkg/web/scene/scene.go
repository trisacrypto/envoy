/*
Scene provides well structured template contexts and functionality for HTML template
rendering. We chose the word "scene" to represent the context since "context" is an
overloaded term and milieu was too hard to spell.
*/
package scene

import (
	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/web/auth"
)

// Compute the version of the package at runtime so it is static for all contexts.
var version = pkg.Version()

// Keys for default Scene context items
const (
	Version         = "Version"
	Page            = "Page"
	IsAuthenticated = "IsAuthenticated"
	User            = "User"
	UserAdmin       = "Admin"
	UserCompliance  = "Compliance"
	UserObserver    = "Observer"
)

type Scene map[string]interface{}

func New(c *gin.Context) Scene {
	if c == nil {
		return Scene{
			Version: version,
		}
	}

	// Create the basic context
	context := Scene{
		Version: version,
		Page:    c.Request.URL.Path,
	}

	// Does the user exist in the gin context?
	if claims, err := auth.GetClaims(c); err != nil {
		context[IsAuthenticated] = false
		context[User] = nil
	} else {
		context[IsAuthenticated] = true
		context[User] = claims
	}

	return context
}

func (s Scene) Update(o Scene) {
	for key, val := range o {
		s[key] = val
	}
}

func (s Scene) HasRole(role string) bool {
	if user := s.GetUser(); user != nil {
		return user.Role == role
	}
	return false
}

func (s Scene) GetUser() *auth.Claims {
	if s.IsAuthenticated() {
		if claims, ok := s[User]; ok {
			if user, ok := claims.(*auth.Claims); ok {
				return user
			}
		}
	}
	return nil
}

func (s Scene) IsAuthenticated() bool {
	if isauths, ok := s[IsAuthenticated]; ok {
		return isauths.(bool)
	}
	return false
}

func GetAuthUserRole(c *gin.Context) string {
	ctx := New(c)
	user := ctx.GetUser()
	return user.Role
}

func HasPermission(c *gin.Context) bool {
	role := GetAuthUserRole(c)
	return role != UserObserver
}
