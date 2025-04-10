package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type Source uint8

const (
	SourceUnknown Source = iota
	SourceDirectorySync
	SourceUserEntry
	SourcePeer
	SourceLocal
	SourceRemote
	SourceDaybreak
)

var sourceNames [7]string = [...]string{
	"unknown", "gds", "user", "peer", "local", "remote", "daybreak",
}

func ValidSource(s interface{}) bool {
	if p, err := ParseSource(s); err != nil || p > SourceDaybreak {
		return false
	}
	return true
}

func CheckSource(s interface{}, targets ...Source) (_ bool, err error) {
	var source Source
	if source, err = ParseSource(s); err != nil {
		return false, err
	}

	for _, target := range targets {
		if source == target {
			return true, nil
		}
	}

	return false, nil
}

func ParseSource(s interface{}) (Source, error) {
	switch s := s.(type) {
	case string:
		s = strings.ToLower(s)
		if s == "" {
			return SourceUnknown, nil
		}

		for i, name := range sourceNames {
			if name == s {
				return Source(i), nil
			}
		}
		return SourceUnknown, fmt.Errorf("invalid source: %q", s)
	case uint8:
		return Source(s), nil
	case Source:
		return s, nil
	default:
		return SourceUnknown, fmt.Errorf("cannot parse %T into a source", s)
	}
}

func (s Source) String() string {
	if s > SourceRemote {
		return sourceNames[0]
	}
	return sourceNames[s]
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (s Source) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Source) UnmarshalJSON(b []byte) (err error) {
	var src string
	if err = json.Unmarshal(b, &src); err != nil {
		return err
	}
	if *s, err = ParseSource(src); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (s *Source) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*s, err = ParseSource(x)
		return err
	case []byte:
		*s, err = ParseSource(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into a source", src)
}

func (s Source) Value() (driver.Value, error) {
	return s.String(), nil
}
