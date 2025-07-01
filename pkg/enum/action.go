package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

// Action is the type of change made in the database for a ComplianceAuditLog
type Action uint8

const (
	ActionUnknown Action = iota
	ActionCreate
	ActionUpdate
	ActionDelete

	// The terminator is used to determine the last value of the enum. It should be
	// the last value in the list and is automatically incremented when enums are
	// added above it.
	// NOTE: you should not reorder the enums, just append them to the list above
	// to add new values.
	actionTerminator
)

var actionNames = [4]string{
	"unknown",
	"create",
	"update",
	"delete",
}

// Returns true if the provided action is valid (e.g. parseable), false otherwise.
func ValidAction(t interface{}) bool {
	if r, err := ParseAction(t); err != nil || r >= actionTerminator {
		return false
	}
	return true
}

// Returns true if the action is equal to one of the target actions. Any parse
// errors for the action are returned.
func CheckAction(t interface{}, targets ...Action) (_ bool, err error) {
	var r Action
	if r, err = ParseAction(t); err != nil {
		return false, err
	}

	for _, target := range targets {
		if r == target {
			return true, nil
		}
	}

	return false, nil
}

// Parse the action from the provided value.
func ParseAction(t interface{}) (Action, error) {
	switch t := t.(type) {
	case string:
		t = strings.ToLower(t)
		if t == "" {
			return ActionUnknown, nil
		}

		for i, name := range actionNames {
			if name == t {
				return Action(i), nil
			}
		}
		return ActionUnknown, fmt.Errorf("invalid action: %q", t)
	case uint8:
		return Action(t), nil
	case Action:
		return t, nil
	default:
		return ActionUnknown, fmt.Errorf("cannot parse %T into an action", t)
	}
}

// Return a string representation of the action.
func (r Action) String() string {
	if r >= actionTerminator {
		return actionNames[0]
	}
	return actionNames[r]
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (a Action) MarshalJSON() ([]byte, error) {
	return json.Marshal(a.String())
}

func (a *Action) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return err
	}
	if *a, err = ParseAction(s); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (a *Action) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*a, err = ParseAction(x)
		return err
	case []byte:
		*a, err = ParseAction(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into an action", src)
}

func (a Action) Value() (driver.Value, error) {
	return a.String(), nil
}
