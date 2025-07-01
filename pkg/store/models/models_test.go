package models_test

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"
)

//==========================================================================
// Tests
//==========================================================================

//TODO

//==========================================================================
// Helpers
//==========================================================================

// Converts a name (like a field or column name) into a common format for comparison.
// Example: `some_variable`, `someVariable`, and `SomeVariable` will all become
// `somevariable` so they can be compared.
func ConvertNameForComparison(name string) string {
	// trim whitespace
	name = strings.TrimSpace(name)

	// remove underscores (for snake_case)
	name = strings.ReplaceAll(name, "_", "")

	// to lowercase (for CamelCase or camelCase)
	name = strings.ToLower(name)

	return name
}

// Returns a slice of strings that have the names of all public fields in the
// given interface. The names will be converted using `ConvertNameForComparison()`.
func GetPublicFieldNames(s interface{}) (fields []string) {
	t := reflect.TypeOf(s)
	if t.Kind() == reflect.Struct {
		for _, field := range reflect.VisibleFields(t) {
			if field.IsExported() && !field.Anonymous { // only collect public fields
				fields = append(fields, ConvertNameForComparison(field.Name))
			}
		}
		return fields
	} else {
		panic(fmt.Sprintf("input must be a struct, but the kind is %s", t.Kind().String()))
	}
}

// Has a `Params()` that returns a list of `sql.NamedArg`.
type Paramsable interface {
	Params() []any
}

// Returns a slice of strings that have the names from a model's `Params()`
// function. If a parameter name is found in the `exceptions` map as a key then
// the value from the map will be returned in its place.
func GetParamsNames(p Paramsable, exceptions map[string]string) (params []string) {
	params = make([]string, 0)
	for _, param := range p.Params() {
		if param, ok := param.(sql.NamedArg); ok {
			name := ConvertNameForComparison(param.Name)
			if replacement, ok := exceptions[name]; ok {
				if replacement == "" {
					continue
				}
				params = append(params, replacement)
			} else {
				params = append(params, name)
			}
		}
	}
	return params
}

// Returns a map that contains `true` for each string in the input list. The strings
// are first converted using `ConvertNameForComparison()`.
func MakeNameComparisonMap(input []string) (out map[string]bool) {
	out = make(map[string]bool)
	for _, s := range input {
		out[ConvertNameForComparison(s)] = true
	}
	return out
}
