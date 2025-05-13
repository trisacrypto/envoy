/*
Scene provides well structured template contexts and functionality for HTML template
rendering. We chose the word "scene" to represent the context since "context" is an
overloaded term and milieu was too hard to spell.
*/
package scene

import (
	"github.com/gin-gonic/gin"
	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/config"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	"go.rtnl.ai/ulid"
)

var (
	// Compute the version of the package at runtime so it is static for all contexts.
	version      = pkg.Version(false)
	shortVersion = pkg.Version(true)

	// Configuration values set from the global configuration to be included in context.
	// Global information
	organization *string

	// Protocol configuration values
	trisaEnabled *bool
	trpEnabled   *bool

	// sunriseEnabled is true iff sunrise is enabled and email messaging is available
	sunriseEnabled *bool

	// daybreakEnabled is true iff daybreak is enabled by the configuration
	daybreakEnabled *bool
)

// Keys for default Scene context items
const (
	Version         = "Version"
	ShortVersion    = "ShortVersion"
	Page            = "Page"
	IsAuthenticated = "IsAuthenticated"
	User            = "User"
	APIData         = "APIData"
	Parent          = "Parent"
	Organization    = "Organization"
	TRISAEnabled    = "TRISAEnabled"
	TRPEnabled      = "TRPEnabled"
	SunriseEnabled  = "SunriseEnabled"
	DaybreakEnabled = "DaybreakEnabled"
)

type Scene map[string]interface{}

func New(c *gin.Context) Scene {
	if c == nil {
		return Scene{
			Version:      version,
			ShortVersion: shortVersion,
		}
	}

	// Create the basic context
	context := Scene{
		Version:      version,
		ShortVersion: shortVersion,
		Page:         c.Request.URL.Path,
	}

	// Does the user exist in the gin context?
	if claims, err := auth.GetClaims(c); err != nil {
		context[IsAuthenticated] = false
		context[User] = nil
	} else {
		context[IsAuthenticated] = true
		context[User] = claims
	}

	// Add configuration values
	if organization != nil {
		context[Organization] = *organization
	}

	if trisaEnabled != nil {
		context[TRISAEnabled] = *trisaEnabled
	}

	if trpEnabled != nil {
		context[TRPEnabled] = *trpEnabled
	}

	if sunriseEnabled != nil {
		context[SunriseEnabled] = *sunriseEnabled
	}

	if daybreakEnabled != nil {
		context[DaybreakEnabled] = *daybreakEnabled
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

func (s Scene) WithParent(parent ulid.ULID) Scene {
	s[Parent] = parent.String()
	return s
}

func (s Scene) With(key string, val interface{}) Scene {
	s[key] = val
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

func (s Scene) AccountPerson() Person {
	if data, ok := s[APIData]; ok {
		if account, ok := data.(*api.Account); ok {
			// Try to get the IVMS101 person
			if person, err := account.IVMS101(); err == nil {
				if np := person.GetNaturalPerson(); np != nil {
					return makePerson(np)
				}
			}

			// Otherwise get the account information available from the struct.
			return Person{
				Forename:       account.FirstName,
				Surname:        account.LastName,
				CustomerNumber: account.CustomerNumber(),
			}
		}
	}

	// Return an empty person so no checking has to be done.
	return Person{}
}

func (s Scene) CryptoAddressList() *api.CryptoAddressList {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.CryptoAddressList); ok {
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

func (s Scene) UserDetail() *api.User {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.User); ok {
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

func (s Scene) CounterpartyDetail() *api.Counterparty {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.Counterparty); ok {
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

func (s Scene) EnvelopeList() *api.EnvelopesList {
	if data, ok := s[APIData]; ok {
		if out, ok := data.(*api.EnvelopesList); ok {
			return out
		}
	}
	return nil
}

func (s Scene) SendEnabledForProtocol(protocol string) bool {
	switch protocol {
	case "trisa":
		return trisaEnabled != nil && *trisaEnabled
	case "trp":
		return trpEnabled != nil && *trpEnabled
	case "sunrise":
		return sunriseEnabled != nil && *sunriseEnabled
	default:
		return false
	}
}

//===========================================================================
// Set Global Scene for Context
//===========================================================================

func WithConf(conf *config.Config) {
	// Set the organization name for the configuration
	organization = &conf.Organization

	// Compute the sunriseEnabled boolean
	sEnabled := conf.Sunrise.Enabled && conf.Email.Available()
	sunriseEnabled = &sEnabled

	trisaEnabled = &conf.Node.Enabled
	trpEnabled = &conf.TRP.Enabled
	daybreakEnabled = &conf.Web.Daybreak.Enabled
}
