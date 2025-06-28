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
)

var nameToAction map[string]Action = map[string]Action{
	"unknown": ActionUnknown,
	"create":  ActionCreate,
	"update":  ActionUpdate,
	"delete":  ActionDelete,
}

var actionToName map[Action]string = map[Action]string{
	ActionUnknown: "unknown",
	ActionCreate:  "create",
	ActionUpdate:  "update",
	ActionDelete:  "delete",
}

func ParseAction(a interface{}) (Action, error) {
	switch a := a.(type) {
	case string:
		a = strings.ToLower(a)
		if a == "" {
			return ActionUnknown, nil
		}

		if action, ok := nameToAction[a]; ok {
			return action, nil
		}

		return ActionUnknown, fmt.Errorf("invalid action: %q", a)
	case uint8:
		return Action(a), nil
	case Action:
		return a, nil
	default:
		return ActionUnknown, fmt.Errorf("cannot parse %T into an action", a)
	}
}

func (a Action) String() string {
	if name, ok := actionToName[a]; ok {
		return name
	}
	return actionToName[ActionUnknown]
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
