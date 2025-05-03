package web_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/trisa/pkg/openvasp/traddr"
)

func TestBeneficiaryTravelAddressCreation(t *testing.T) {
	ta, err := traddr.Encode("//api.bob.vaspbot.com:443/?t=i")
	require.NoError(t, err, "could not encode endpoint as travel address")
	require.Equal(t, "ta24sKtCmQ92AFBJjkbFW6DxDigXsqZQbyq7k1ePJpN57gffo", ta)
}
