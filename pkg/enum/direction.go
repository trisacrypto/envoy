package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type Direction uint8

const (
	DirectionUnknown Direction = iota
	DirectionIncoming
	DirectionOutgoing
	DirectionAny
)

var directionNames map[string]Direction = map[string]Direction{
	"unknown":  DirectionUnknown,
	"in":       DirectionIncoming,
	"incoming": DirectionIncoming,
	"out":      DirectionOutgoing,
	"outgoing": DirectionOutgoing,
	"any":      DirectionAny,
}

func ParseDirection(s interface{}) (Direction, error) {
	switch s := s.(type) {
	case string:
		s = strings.ToLower(s)
		if s == "" {
			return DirectionUnknown, nil
		}

		if direction, ok := directionNames[s]; ok {
			return direction, nil
		}

		return DirectionUnknown, fmt.Errorf("invalid direction: %q", s)
	case uint8:
		return Direction(s), nil
	case Direction:
		return s, nil
	default:
		return DirectionUnknown, fmt.Errorf("cannot parse %T into a direction", s)
	}
}

func (d Direction) String() string {
	switch d {
	case DirectionIncoming:
		return "in"
	case DirectionOutgoing:
		return "out"
	case DirectionAny:
		return "any"
	default:
		return "unknown"
	}
}

func (d Direction) Verbose() string {
	switch d {
	case DirectionIncoming:
		return "incoming"
	case DirectionOutgoing:
		return "outgoing"
	default:
		return d.String()
	}
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (d Direction) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.String())
}

func (d *Direction) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return err
	}
	if *d, err = ParseDirection(s); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (d *Direction) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*d, err = ParseDirection(x)
		return err
	case []byte:
		*d, err = ParseDirection(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into a direction", src)
}

func (d Direction) Value() (driver.Value, error) {
	return d.String(), nil
}
