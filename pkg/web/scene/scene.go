/*
Scene provides well structured template contexts and functionality for HTML template
rendering. We chose the word "scene" to represent the context since "context" is an
overloaded term and milieu was too hard to spell.
*/
package scene

import (
	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
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
	APIData         = "APIData"
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

func (s Scene) Update(o Scene) Scene {
	for key, val := range o {
		s[key] = val
	}
	return s
}

func (s Scene) WithAPIData(data interface{}) Scene {
	s[APIData] = data
	return s
}

//===========================================================================
// Scene User Related Helpers
//===========================================================================

// Role string constants
const (
	RoleAdmin      = "Admin"
	RoleCompliance = "Compliance"
	RoleObserver   = "Observer"
)

func (s Scene) IsAuthenticated() bool {
	if isauths, ok := s[IsAuthenticated]; ok {
		return isauths.(bool)
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

func (s Scene) HasRole(role string) bool {
	if user := s.GetUser(); user != nil {
		return user.Role == role
	}
	return false
}

func (s Scene) IsAdmin() bool {
	return s.HasRole(RoleAdmin)
}

func (s Scene) IsViewOnly() bool {
	return !s.HasRole(RoleAdmin) && !s.HasRole(RoleCompliance)
}

//===========================================================================
// Scene API Data Related Helpers
//===========================================================================

func (s Scene) AccountsList() *api.AccountsList {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.AccountsList); ok {
			return out
		}
	}
	return nil
}

func (s Scene) AccountDetail() *api.Account {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.Account); ok {
			return out
		}
	}
	return nil
}

func (s Scene) UserList() *api.UserList {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.UserList); ok {
			return out
		}
	}
	return nil
}

func (s Scene) CounterpartyList() *api.CounterpartyList {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.CounterpartyList); ok {
			return out
		}
	}
	return nil
}

func (s Scene) TransactionsList() *api.TransactionsList {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.TransactionsList); ok {
			return out
		}
	}
	return nil
}

func (s Scene) TransactionDetail() *api.Transaction {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.Transaction); ok {
			return out
		}
	}
	return nil
}

func (s Scene) APIKeysList() *api.APIKeyList {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.APIKeyList); ok {
			return out
		}
	}
	return nil
}

func (s Scene) CreateAPIKey() *api.APIKey {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.APIKey); ok {
			return out
		}
	}
	return nil
}

func (s Scene) APIKeyDetail() *api.APIKey {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.APIKey); ok {
			return out
		}
	}
	return nil
}
