package models

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/trisacrypto/envoy/pkg/store/errors"

	"github.com/google/uuid"
	"github.com/oklog/ulid/v2"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

// Factory function that can create travel addresses from models or IDs. This function
// should be able to handle any model defined in this package or a ULID or UUID.
type TravelAddressFactory func(any) (string, error)

// Create a travel address factory with the endpoint and protocol.
func NewTravelAddressFactory(endpoint, protocol string) (TravelAddressFactory, error) {
	baseURL, err := traddr.Parse(endpoint)
	if err != nil {
		return nil, err
	}

	if baseURL.Host == "" {
		return nil, errors.ErrNoEndpoint
	}

	return func(a any) (string, error) {
		var path string
		switch t := a.(type) {
		case *Account:
			path, _ = url.JoinPath("accounts", t.ID.String())
		case *CryptoAddress:
			path, _ = url.JoinPath("wallets", t.ID.String())
		case ulid.ULID:
			path, _ = url.JoinPath("/", t.String())
		case uuid.UUID:
			path, _ = url.JoinPath("/", t.String())
		default:
			return "", fmt.Errorf("cannot create travel address for unhandled type %T", t)
		}

		params := make(url.Values)
		params.Set("t", "i")
		if protocol != "" {
			params.Set("mode", protocol)
		}

		uri := baseURL.ResolveReference(&url.URL{Path: path, RawQuery: params.Encode()})
		return traddr.Encode(strings.TrimPrefix(uri.String(), "//"))
	}, nil
}
