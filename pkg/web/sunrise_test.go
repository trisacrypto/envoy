package web_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
)

func TestSunriseIntegration(t *testing.T) {
	// Load local .env if it exists to make setting envvars easier.
	godotenv.Load()

	// This test sends a sunrise message to a locally running server; it is skipped if
	// the $SUNRISE_TEST_INTEGRATION environment variable is not set to a boolean true.
	SkipByEnvVar(t, "SUNRISE_TEST_INTEGRATION")
	CheckEnvVar(t, "ENVOY_CLIENT_ID")
	CheckEnvVar(t, "ENVOY_CLIENT_SECRET")

	// Create a client to connect to a running sunrise server
	client, err := api.New("http://localhost:8000")
	require.NoError(t, err, "could not create envoy api client")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Check to make sure the server is running
	status, err := client.Status(ctx)
	require.NoError(t, err, "could not connect to envoy api")
	require.Equal(t, "ok", status.Status, "envoy api is not ready for testing")

	// Authenticate the client
	_, err = client.Authenticate(ctx, &api.APIAuthentication{
		ClientID:     os.Getenv("ENVOY_CLIENT_ID"),
		ClientSecret: os.Getenv("ENVOY_CLIENT_SECRET"),
	})
	require.NoError(t, err, "could not authenticate the client")

	// Send a sunrise payload
	out, err := client.SendSunrise(ctx, &api.Sunrise{
		Email:        "benjamin@rotational.io",
		Counterparty: "SpudCoin Exchange",
		Originator: &api.Person{
			FirstName: "Alice",
			LastName:  "Murray",
			Identification: &api.Identification{
				TypeCode:    "SOCS",
				Number:      "800-00-8080",
				Country:     "US",
				DateOfBirth: "1982-03-14",
				BirthPlace:  "Carlsbad, CA",
			},
			AddrLine1:     "134 Deercove Drive",
			City:          "Arlington",
			State:         "TX",
			PostalCode:    "76011",
			Country:       "US",
			CryptoAddress: "n3oDpHRYue9Ene9neasSE9cchfXNdtfzYM",
		},
		Beneficiary: &api.Person{
			FirstName:     "Larissa Correia",
			LastName:      "Sousa",
			AddrLine1:     "Rua Cajamar, 673",
			City:          "Jandira-SP",
			PostalCode:    "06622-290",
			Country:       "BR",
			CryptoAddress: "mkGQh6QSYhAVHXBFaMDoDAd8zKQLBMayy4",
		},
		Transfer: &api.Transfer{
			Amount:  0.000005421,
			Network: "BTC",
		},
	})
	require.NoError(t, err, "could not send sunrise message")
	require.NotNil(t, out, "no transaction returned")
}

func SkipByEnvVar(t *testing.T, env string) {
	val := strings.ToLower(strings.TrimSpace(os.Getenv(env)))
	switch val {
	case "1", "t", "true":
		return
	default:
		t.Skipf("this test depends on the $%s envvar to run", env)
	}
}

func CheckEnvVar(t *testing.T, env string) {
	require.NotEmpty(t, os.Getenv(env), "required environment variable $%s not set", env)
}
