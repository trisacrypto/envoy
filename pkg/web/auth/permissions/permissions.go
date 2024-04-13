package permissions

import (
	"encoding/json"
	"fmt"
	"strings"
)

type Permission uint8

// Permissions constants from the database associated with the primary Key
const (
	Unknown Permission = iota
	UsersManage
	UsersView
	APIKeysManage
	APIKeysView
	APIKeysRevoke
	CounterpartiesManage
	CounterpartiesView
	AccountsManage
	AccountsView
	TravelRuleManage
	TravelRuleDelete
	TravelRuleView
	ConfigManage
	ConfigView
	PKIManage
	PKIDelete
	PKIView
)

var names = [18]string{
	"unknown",
	"users:manage", "users:view",
	"apikeys:manage", "apikeys:view", "apikeys:revoke",
	"counterparties:manage", "counterparties:view",
	"accounts:manage", "accounts:view",
	"travelrule:manage", "travelrule:delete", "travelrule:view",
	"config:manage", "config:view",
	"pki:manage", "pki:delete", "pki:view",
}

func Parse(p any) (Permission, error) {
	switch t := p.(type) {
	case uint8:
		return Permission(t), nil
	case int64:
		return Permission(t), nil
	case string:
		t = strings.ToLower(strings.TrimSpace(t))
		for idx, name := range names {
			if t == name {
				return Permission(idx), nil
			}
		}
		return Unknown, fmt.Errorf("%q is not a valid permission name", t)
	case Permission:
		return t, nil
	default:
		return Unknown, fmt.Errorf("cannot parse type %T into a permission", t)
	}
}

func (p Permission) String() string {
	if idx := int(p); idx < len(names) {
		return names[idx]
	}
	return names[0]
}

func (p Permission) MarshalJSON() ([]byte, error) {
	return json.Marshal(p.String())
}

func (p *Permission) UnmarshalJSON(data []byte) (err error) {
	var str string
	if err := json.Unmarshal(data, &str); err != nil {
		return err
	}
	if *p, err = Parse(str); err != nil {
		return err
	}
	return nil
}
