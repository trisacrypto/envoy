package permissions_test

import (
	"encoding/json"
	"testing"

	"self-hosted-node/pkg/web/auth/permissions"

	"github.com/stretchr/testify/require"
)

func TestParse(t *testing.T) {

	t.Run("Valid", func(t *testing.T) {
		tests := []struct {
			input    any
			expected permissions.Permission
		}{
			{uint8(0), permissions.Unknown},
			{uint8(1), permissions.UsersManage},
			{uint8(2), permissions.UsersView},
			{uint8(3), permissions.APIKeysManage},
			{uint8(4), permissions.APIKeysView},
			{uint8(5), permissions.APIKeysRevoke},
			{uint8(6), permissions.CounterpartiesManage},
			{uint8(7), permissions.CounterpartiesView},
			{uint8(8), permissions.AccountsManage},
			{uint8(9), permissions.AccountsView},
			{uint8(10), permissions.TravelRuleManage},
			{uint8(11), permissions.TravelRuleDelete},
			{uint8(12), permissions.TravelRuleView},
			{uint8(13), permissions.ConfigManage},
			{uint8(14), permissions.ConfigView},
			{uint8(15), permissions.PKIManage},
			{uint8(16), permissions.PKIDelete},
			{uint8(17), permissions.PKIView},
			{int64(0), permissions.Unknown},
			{int64(1), permissions.UsersManage},
			{int64(2), permissions.UsersView},
			{int64(3), permissions.APIKeysManage},
			{int64(4), permissions.APIKeysView},
			{int64(5), permissions.APIKeysRevoke},
			{int64(6), permissions.CounterpartiesManage},
			{int64(7), permissions.CounterpartiesView},
			{int64(8), permissions.AccountsManage},
			{int64(9), permissions.AccountsView},
			{int64(10), permissions.TravelRuleManage},
			{int64(11), permissions.TravelRuleDelete},
			{int64(12), permissions.TravelRuleView},
			{int64(13), permissions.ConfigManage},
			{int64(14), permissions.ConfigView},
			{int64(15), permissions.PKIManage},
			{int64(16), permissions.PKIDelete},
			{int64(17), permissions.PKIView},
			{"unknown", permissions.Unknown},
			{"users:manage", permissions.UsersManage},
			{"users:view", permissions.UsersView},
			{"apikeys:manage", permissions.APIKeysManage},
			{"apikeys:view", permissions.APIKeysView},
			{"apikeys:revoke", permissions.APIKeysRevoke},
			{"counterparties:manage", permissions.CounterpartiesManage},
			{"counterparties:view", permissions.CounterpartiesView},
			{"accounts:manage", permissions.AccountsManage},
			{"accounts:view", permissions.AccountsView},
			{"travelrule:manage", permissions.TravelRuleManage},
			{"travelrule:delete", permissions.TravelRuleDelete},
			{"travelrule:view", permissions.TravelRuleView},
			{"config:manage", permissions.ConfigManage},
			{"config:view", permissions.ConfigView},
			{"pki:manage", permissions.PKIManage},
			{"pki:delete", permissions.PKIDelete},
			{"pki:view", permissions.PKIView},
			{"TRAVELRULE:DELETE", permissions.TravelRuleDelete},
			{"APIKeys:Revoke", permissions.APIKeysRevoke},
			{"  config:manage   ", permissions.ConfigManage},
			{permissions.UsersManage, permissions.UsersManage},
		}

		for i, tc := range tests {
			actual, err := permissions.Parse(tc.input)
			require.NoError(t, err, "error occurred parsing on test case %d", i)
			require.Equal(t, tc.expected, actual, "test case %d failed", i)
		}
	})

	t.Run("Invalid", func(t *testing.T) {
		tests := []struct {
			input any
			err   string
		}{
			{"foo", `"foo" is not a valid permission name`},
			{false, `cannot parse type bool into a permission`},
		}

		for i, tc := range tests {
			_, err := permissions.Parse(tc.input)
			require.EqualError(t, err, tc.err, "expected error parsing test case %d", i)
		}
	})
}

func TestJSON(t *testing.T) {
	all := []permissions.Permission{
		permissions.UsersManage,
		permissions.UsersView,
		permissions.APIKeysManage,
		permissions.APIKeysView,
		permissions.APIKeysRevoke,
		permissions.CounterpartiesManage,
		permissions.CounterpartiesView,
		permissions.AccountsManage,
		permissions.AccountsView,
		permissions.TravelRuleManage,
		permissions.TravelRuleDelete,
		permissions.TravelRuleView,
		permissions.ConfigManage,
		permissions.ConfigView,
		permissions.PKIManage,
		permissions.PKIDelete,
		permissions.PKIView,
	}

	for _, perm := range all {
		data, err := json.Marshal(perm)
		require.NoError(t, err, "could not marshal permission %s", perm)

		var p permissions.Permission
		err = json.Unmarshal(data, &p)
		require.NoError(t, err, "could not unmarshal permission")

		require.Equal(t, perm, p, "unmarshal did not return the right permission")
	}

}
