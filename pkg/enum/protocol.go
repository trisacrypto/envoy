package enum

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
)

type Protocol uint8

const (
	ProtocolUnknown Protocol = iota
	ProtocolTRISA
	ProtocolTRP
	ProtocolSunrise
)

var protocolNames [4]string = [...]string{
	"unknown", "trisa", "trp", "sunrise",
}

func ValidProtocol(s interface{}) bool {
	if p, err := ParseProtocol(s); err != nil || p > ProtocolSunrise {
		return false
	}
	return true
}

func ParseProtocol(s interface{}) (Protocol, error) {
	switch s := s.(type) {
	case string:
		s = strings.ToLower(s)
		if s == "" {
			return ProtocolUnknown, nil
		}

		for i, name := range protocolNames {
			if name == s {
				return Protocol(i), nil
			}
		}
		return ProtocolUnknown, fmt.Errorf("invalid protocol: %q", s)
	case uint8:
		return Protocol(s), nil
	case Protocol:
		return s, nil
	default:
		return ProtocolUnknown, fmt.Errorf("cannot parse %T into a protocol", s)
	}
}

func (p Protocol) String() string {
	if p > ProtocolSunrise {
		return protocolNames[0]
	}
	return protocolNames[p]
}

//===========================================================================
// Serialization and Deserialization
//===========================================================================

func (p Protocol) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *Protocol) UnmarshalJSON(b []byte) (err error) {
	var s string
	if err = json.Unmarshal(b, &s); err != nil {
		return err
	}
	if *p, err = ParseProtocol(s); err != nil {
		return err
	}
	return nil
}

//===========================================================================
// Database Interaction
//===========================================================================

func (p *Protocol) Scan(src interface{}) (err error) {
	switch x := src.(type) {
	case nil:
		return nil
	case string:
		*p, err = ParseProtocol(x)
		return err
	case []byte:
		*p, err = ParseProtocol(string(x))
		return err
	}

	return fmt.Errorf("cannot scan %T into a protocol", src)
}

func (p Protocol) Value() (driver.Value, error) {
	return p.String(), nil
}
