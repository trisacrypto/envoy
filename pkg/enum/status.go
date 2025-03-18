package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type Status uint8

const (
	StatusUnspecified Status = iota
	StatusDraft
	StatusPending
	StatusReview
	StatusRepair
	StatusAccepted
	StatusCompleted
	StatusRejected
)

var statusNames [8]string = [...]string{
	"unspecified", "draft", "pending", "review", "repair", "accepted", "completed", "rejected",
}

func ValidStatus(s interface{}) bool {
	if p, err := ParseStatus(s); err != nil || p > StatusRejected {
		return false
	}
	return true
}

func CheckStatus(s interface{}, targets ...Status) (_ bool, err error) {
	var status Status
	if status, err = ParseStatus(s); err != nil {
		return false, err
	}

	for _, target := range targets {
		if status == target {
			return true, nil
		}
	}

	return false, nil
}

func ParseStatus(s interface{}) (Status, error) {
	switch s := s.(type) {
	case string:
		s = strings.TrimSpace(strings.ToLower(s))
		if s == "" {
			return StatusUnspecified, nil
		}

		for i, name := range statusNames {
			if name == s {
				return Status(i), nil
			}
		}
		return StatusUnspecified, fmt.Errorf("invalid status: %q", s)
	case uint8:
		return Status(s), nil
	case Status:
		return s, nil
	default:
		return StatusUnspecified, fmt.Errorf("cannot parse %T into a status", s)
	}
}

func (s Status) String() string {
	if s > StatusRejected {
		return statusNames[0]
	}
	return statusNames[s]
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (s Status) MarshalJSON() ([]byte, error) {
	return json.Marshal(s.String())
}

func (s *Status) UnmarshalJSON(b []byte) (err error) {
	var src string
	if err = json.Unmarshal(b, &src); err != nil {
		return err
	}
	if *s, err = ParseStatus(src); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (s *Status) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*s, err = ParseStatus(x)
		return err
	case []byte:
		*s, err = ParseStatus(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into a status", src)
}

func (s Status) Value() (driver.Value, error) {
	return s.String(), nil
}
