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
)

var nameToResource map[string]Resource = map[string]Resource{
	"unknown":      ResourceUnknown,
	"transaction":  ResourceTransaction,
	"user":         ResourceUser,
	"api_key":      ResourceAPIKey,
	"counterparty": ResourceCounterparty,
	"account":      ResourceAccount,
	"sunrise":      ResourceSunrise,
}

var resourceToName map[Resource]string = map[Resource]string{
	ResourceUnknown:      "unknown",
	ResourceTransaction:  "transaction",
	ResourceUser:         "user",
	ResourceAPIKey:       "api_key",
	ResourceCounterparty: "counterparty",
	ResourceAccount:      "account",
	ResourceSunrise:      "sunrise",
}

func ParseResource(a interface{}) (Resource, error) {
	switch a := a.(type) {
	case string:
		a = strings.ToLower(a)
		if a == "" {
			return ResourceUnknown, nil
		}

		if resource, ok := nameToResource[a]; ok {
			return resource, nil
		}

		return ResourceUnknown, fmt.Errorf("invalid resource: %q", a)
	case uint8:
		return Resource(a), nil
	case Resource:
		return a, nil
	default:
		return ResourceUnknown, fmt.Errorf("cannot parse %T into a resource", a)
	}
}

func (r Resource) String() string {
	if name, ok := resourceToName[r]; ok {
		return name
	}
	return resourceToName[ResourceUnknown]
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
