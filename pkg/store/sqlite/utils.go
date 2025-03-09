package sqlite

import (
	"database/sql"
	"fmt"
	"strings"
)

// Used to parameterize a list of values for an IN clause in a SQLite query.
// Given a list of values, ["apple", "berry", "cherry"], and a prefix, "f",
// the function will return the query string "(f0, f1, f2)" and the params slice
// []interface{}{sql.Named("f0", "apple"), sql.Named("f1", "berry"), sql.Named("f2", "cherry")}.
// The placeholders are used to prevent SQL injection attacks and the query string can
// be appended to a query such as "SELECT * FROM fruits WHERE name IN ".
func listParametrize(values []string, prefix string) (query string, params []interface{}) {
	placeholders := make([]string, 0, len(values))
	params = make([]interface{}, 0, len(values))
	for i, param := range values {
		placeholder := fmt.Sprintf("%s%d", prefix, i)
		placeholders = append(placeholders, ":"+placeholder)
		params = append(params, sql.Named(placeholder, param))
	}
	return "(" + strings.Join(placeholders, ", ") + ")", params
}
