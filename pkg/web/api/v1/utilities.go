package api

import (
	"regexp"
	"strings"
)

type TravelAddress struct {
	Encoded string `json:"encoded,omitempty"`
	Decoded string `json:"decoded,omitempty"`
}

var travelAddressRegex = regexp.MustCompile(`ta[123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz]+`)

func (t *TravelAddress) ValidateEncode() (err error) {
	t.Encoded = strings.TrimSpace(t.Encoded)
	t.Decoded = strings.TrimSpace(t.Decoded)

	if t.Decoded == "" {
		err = ValidationError(err, MissingField("decoded"))
	}

	if t.Encoded != "" {
		err = ValidationError(err, ReadOnlyField("encoded"))
	}

	return err
}

func (t *TravelAddress) ValidateDecode() (err error) {
	t.Encoded = strings.TrimSpace(t.Encoded)
	t.Decoded = strings.TrimSpace(t.Decoded)

	if t.Encoded == "" {
		err = ValidationError(err, MissingField("encoded"))
	} else if !travelAddressRegex.MatchString(t.Encoded) {
		err = ValidationError(err, IncorrectField("encoded", "input does not match travel address format"))
	}

	if t.Decoded != "" {
		err = ValidationError(err, ReadOnlyField("decoded"))
	}

	return err
}
