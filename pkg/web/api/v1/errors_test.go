package api_test

import (
	"fmt"
	"testing"

	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/stretchr/testify/require"
)

func TestValidationErrors(t *testing.T) {

	t.Run("Nil", func(t *testing.T) {
		require.NoError(t, api.ValidationError(nil, nil, nil, nil, nil))
	})

	t.Run("Single", func(t *testing.T) {
		testCases := []struct {
			err      error
			errs     []*api.FieldError
			expected string
		}{
			{
				nil,
				[]*api.FieldError{api.MissingField("foo")},
				"missing foo: this field is required",
			},
			{
				make(api.ValidationErrors, 0),
				[]*api.FieldError{api.MissingField("foo")},
				"missing foo: this field is required",
			},
			{
				nil,
				[]*api.FieldError{nil, api.MissingField("foo"), nil},
				"missing foo: this field is required",
			},
		}

		for i, tc := range testCases {
			err := api.ValidationError(tc.err, tc.errs...)
			require.EqualError(t, err, tc.expected, "test case %d failed", i)
		}
	})

	t.Run("Multi", func(t *testing.T) {
		testCases := []struct {
			err      error
			errs     []*api.FieldError
			expected string
		}{
			{
				nil,
				[]*api.FieldError{api.MissingField("foo"), api.MissingField("bar")},
				"2 validation errors occurred:\n  missing foo: this field is required\n  missing bar: this field is required",
			},
			{
				nil,
				[]*api.FieldError{nil, api.MissingField("foo"), nil, api.MissingField("bar"), nil},
				"2 validation errors occurred:\n  missing foo: this field is required\n  missing bar: this field is required",
			},
			{
				api.ValidationErrors([]*api.FieldError{api.MissingField("foo")}),
				[]*api.FieldError{nil, api.MissingField("bar"), nil},
				"2 validation errors occurred:\n  missing foo: this field is required\n  missing bar: this field is required",
			},
		}

		for i, tc := range testCases {
			err := api.ValidationError(tc.err, tc.errs...)
			require.EqualError(t, err, tc.expected, "test case %d failed", i)
		}
	})

}

func ExampleValidationErrors() {
	err := api.ValidationError(
		nil,
		api.MissingField("name"),
		api.IncorrectField("ssn", "ssn should be 8 digits only"),
		nil,
		api.MissingField("date_of_birth"),
		nil,
	)

	fmt.Println(err)
	// Output:
	// 	3 validation errors occurred:
	//   missing name: this field is required
	//   invalid field ssn: ssn should be 8 digits only
	//   missing date_of_birth: this field is required
}
