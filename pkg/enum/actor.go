package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// Actor allows us to decode the ActorID in a ComplianceAuditLog and is human-readable
type Actor uint8

const (
	ActorUnknown Actor = iota
	ActorUser
	ActorAPIKey
	ActorSunrise
)

var nameToActor map[string]Actor = map[string]Actor{
	"unknown": ActorUnknown,
	"user":    ActorUser,
	"api_key": ActorAPIKey,
	"sunrise": ActorSunrise,
}

var actorToName map[Actor]string = map[Actor]string{
	ActorUnknown: "unknown",
	ActorUser:    "user",
	ActorAPIKey:  "api_key",
	ActorSunrise: "sunrise",
}

func ParseActor(a interface{}) (Actor, error) {
	switch a := a.(type) {
	case string:
		a = strings.ToLower(a)
		if a == "" {
			return ActorUnknown, nil
		}

		if actor, ok := nameToActor[a]; ok {
			return actor, nil
		}

		return ActorUnknown, fmt.Errorf("invalid actor: %q", a)
	case uint8:
		return Actor(a), nil
	case Actor:
		return a, nil
	default:
		return ActorUnknown, fmt.Errorf("cannot parse %T into an actor", a)
	}
}

func (a Actor) String() string {
	if name, ok := actorToName[a]; ok {
		return name
	}
	return actorToName[ActorUnknown]
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (a Actor) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Actor) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return err
	}
	if *a, err = ParseActor(s); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (a *Actor) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*a, err = ParseActor(x)
		return err
	case []byte:
		*a, err = ParseActor(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into an actor", src)
}

func (a Actor) Value() (driver.Value, error) {
	return a.String(), nil
}
