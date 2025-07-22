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
	ActorCLI // command line interface

	// The terminator is used to determine the last value of the enum. It should be
	// the last value in the list and is automatically incremented when enums are
	// added above it.
	// NOTE: you should not reorder the enums, just append them to the list above
	// to add new values.
	actorTerminator
)

var actorNames = [5]string{
	"unknown",
	"user",
	"api_key",
	"sunrise",
	"cli",
}

// Returns true if the provided actor is valid (e.g. parseable), false otherwise.
func ValidActor(t interface{}) bool {
	if r, err := ParseActor(t); err != nil || r >= actorTerminator {
		return false
	}
	return true
}

// Returns true if the actor is equal to one of the target actors. Any parse
// errors for the actor are returned.
func CheckActor(t interface{}, targets ...Actor) (_ bool, err error) {
	var r Actor
	if r, err = ParseActor(t); err != nil {
		return false, err
	}

	for _, target := range targets {
		if r == target {
			return true, nil
		}
	}

	return false, nil
}

// Parse the actor from the provided value.
func ParseActor(t interface{}) (Actor, error) {
	switch t := t.(type) {
	case string:
		t = strings.ToLower(t)
		if t == "" {
			return ActorUnknown, nil
		}

		for i, name := range actorNames {
			if name == t {
				return Actor(i), nil
			}
		}
		return ActorUnknown, fmt.Errorf("invalid actor: %q", t)
	case uint8:
		return Actor(t), nil
	case Actor:
		return t, nil
	default:
		return ActorUnknown, fmt.Errorf("cannot parse %T into an actor", t)
	}
}

// Return a string representation of the actor.
func (r Actor) String() string {
	if r >= actorTerminator {
		return actorNames[0]
	}
	return actorNames[r]
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
