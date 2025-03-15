package api

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/trisa/pkg/iso3166"
	"github.com/trisacrypto/trisa/pkg/ivms101"
)

// Find the index in the name identifiers of the legal name of either a legal or natural person.
func FindLegalName(person interface{}) int {
	switch p := person.(type) {
	case *ivms101.Person:
		if np := p.GetNaturalPerson(); np != nil {
			return FindLegalName(np)
		}

		if lp := p.GetLegalPerson(); lp != nil {
			return FindLegalName(lp)
		}

		log.Debug().Str("person", p.String()).Msg("unhandled person identifier type")
		return -1
	case *ivms101.LegalPerson:
		if p.Name != nil {
			for i, name := range p.Name.NameIdentifiers {
				if name.LegalPersonNameIdentifierType == ivms101.LegalPersonLegal {
					return i
				}
			}
		}

		log.Debug().Msg("could not find legal name on legal person")
		return -1
	case *ivms101.NaturalPerson:
		if p.Name != nil {
			for i, name := range p.Name.NameIdentifiers {
				if name.NameIdentifierType == ivms101.NaturalPersonLegal {
					return i
				}
			}
		}

		log.Debug().Msg("could not find legal name on natural person")
		return -1
	default:
		log.Debug().Type("person", person).Msg("unhandled type to find person name")
		return -1
	}
}

// Find primary geographic address of a person in the IVMS101 dataset; the address is
// returned as a series of address lines to simplify the representation.
func FindPrimaryAddress(person interface{}) *ivms101.Address {
	switch p := person.(type) {
	case *ivms101.Person:
		if np := p.GetNaturalPerson(); np != nil {
			return FindPrimaryAddress(np)
		}

		if lp := p.GetLegalPerson(); lp != nil {
			return FindPrimaryAddress(lp)
		}

		log.Debug().Str("person", p.String()).Msg("unhandled person identifier type")
		return nil

	case *ivms101.LegalPerson:
		if len(p.GeographicAddresses) > 0 {
			for _, addr := range p.GeographicAddresses {
				if addr.AddressType == ivms101.AddressTypeBusiness {
					return addr
				}
			}

			// Otherwise just return the first address in the list
			return p.GeographicAddresses[0]
		}
		return nil

	case *ivms101.NaturalPerson:
		if len(p.GeographicAddresses) > 0 {
			for _, addr := range p.GeographicAddresses {
				if addr.AddressType == ivms101.AddressTypeHome {
					return addr
				}
			}

			// Otherwise just return the first address in the list
			return p.GeographicAddresses[0]
		}
		return nil
	default:
		log.Debug().Type("person", person).Msg("unhandled type to find person primary address")
		return nil
	}
}

func MakeAddressLines(addr *ivms101.Address) (address []string) {
	if addr == nil {
		return nil
	}

	// Handle the simple case where there are address lines.
	if len(addr.AddressLine) > 0 {
		address = make([]string, 0, len(addr.AddressLine)+2)
		address = append(address, AddressTypeRepr(addr.AddressType))
		address = append(address, addr.AddressLine...)
		address = append(address, CountryName(addr.Country))
		return filterSpaces(address)
	}

	// Otherwise, construct the address from the individual components.
	// TODO: ensure all components are included and correctly formatted for the country
	address = make([]string, 0, 8)
	address = append(address, AddressTypeRepr(addr.AddressType))
	address = append(address, AddrLineRepr(fmt.Sprintf("%s %s %s", addr.BuildingNumber, addr.BuildingName, addr.StreetName)))
	address = append(address, AddrLineRepr(addr.PostBox))
	address = append(address, AddrLineRepr(fmt.Sprintf("%s %s", addr.Department, addr.SubDepartment)))
	address = append(address, AddrLineRepr(fmt.Sprintf("%s %s", addr.Floor, addr.Room)))
	address = append(address, AddrLineRepr(fmt.Sprintf("%s %s %s %s", addr.TownLocationName, addr.TownName, addr.DistrictName, addr.CountrySubDivision)))
	address = append(address, AddrLineRepr(addr.PostCode))
	address = append(address, CountryName(addr.Country))

	return filterSpaces(address)
}

func AddressTypeRepr(t ivms101.AddressTypeCode) string {
	switch t {
	case ivms101.AddressTypeGeographic:
		return "Geographic Address"
	case ivms101.AddressTypeBusiness:
		return "Business Address"
	case ivms101.AddressTypeHome:
		return "Home Address"
	case ivms101.AddressTypeMisc:
		return "Other Address"
	default:
		return "Address"
	}
}

var dupspace = regexp.MustCompile(`\s+`)

func AddrLineRepr(line string) string {
	line = dupspace.ReplaceAllString(line, " ")
	return strings.TrimSpace(line)
}

func CountryName(country string) string {
	if code, err := iso3166.Find(country); err == nil {
		return code.Country
	}
	return country
}

//===========================================================================
// Helper Functions
//===========================================================================

func filterSpaces(arr []string) []string {
	i := 0
	for _, s := range arr {
		if strings.TrimSpace(s) != "" {
			arr[i] = s
			i++
		}
	}
	return arr[:i]
}
