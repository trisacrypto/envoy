package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// Resource allows us to decode the ResourceID in a ComplianceAuditLog and is human-readable
type Resource uint8

const (
	ResourceUnknown Resource = iota
	ResourceTransaction
	ResourceUser
	ResourceAPIKey
	ResourceCounterparty
	ResourceAccount
	ResourceSunrise
	ResourceSecureEnvelope
	ResourceCryptoAddress
	ResourceContact

	// The terminator is used to determine the last value of the enum. It should be
	// the last value in the list and is automatically incremented when enums are
	// added above it.
	// NOTE: you should not reorder the enums, just append them to the list above
	// to add new values.
	resourceTerminator
)

var resourceNames = [10]string{
	"unknown",
	"transaction",
	"user",
	"api_key",
	"counterparty",
	"account",
	"sunrise",
	"secure_envelope",
	"crypto_address",
	"contact",
}

// Returns true if the provided resource is valid (e.g. parseable), false otherwise.
func ValidResource(t interface{}) bool {
	if r, err := ParseResource(t); err != nil || r >= resourceTerminator {
		return false
	}
	return true
}

// Returns true if the resource is equal to one of the target resources. Any parse
// errors for the resource are returned.
func CheckResource(t interface{}, targets ...Resource) (_ bool, err error) {
	var r Resource
	if r, err = ParseResource(t); err != nil {
		return false, err
	}

	for _, target := range targets {
		if r == target {
			return true, nil
		}
	}

	return false, nil
}

// Parse the resource from the provided value.
func ParseResource(t interface{}) (Resource, error) {
	switch t := t.(type) {
	case string:
		t = strings.ToLower(t)
		if t == "" {
			return ResourceUnknown, nil
		}

		for i, name := range resourceNames {
			if name == t {
				return Resource(i), nil
			}
		}
		return ResourceUnknown, fmt.Errorf("invalid resource: %q", t)
	case uint8:
		return Resource(t), nil
	case Resource:
		return t, nil
	default:
		return ResourceUnknown, fmt.Errorf("cannot parse %T into a resource", t)
	}
}

// Return a string representation of the resource.
func (r Resource) String() string {
	if r >= resourceTerminator {
		return resourceNames[0]
	}
	return resourceNames[r]
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (r Resource) MarshalJSON() ([]byte, error) {
	return json.Marshal(r.String())
}

func (r *Resource) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return err
	}
	if *r, err = ParseResource(s); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (r *Resource) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*r, err = ParseResource(x)
		return err
	case []byte:
		*r, err = ParseResource(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into a resource", src)
}

func (r Resource) Value() (driver.Value, error) {
	return r.String(), nil
}
