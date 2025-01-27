// FSI stands for "foreign service institute" e.g. where envoys are trained.
package main

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/trisacrypto/directory/pkg/gds/config"
	"github.com/trisacrypto/directory/pkg/store"
	dbconf "github.com/trisacrypto/directory/pkg/store/config"
	"github.com/trisacrypto/directory/pkg/utils/logger"
	pb "github.com/trisacrypto/trisa/pkg/trisa/gds/models/v1beta1"

	"github.com/trisacrypto/envoy/pkg"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/urfave/cli/v2"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

var (
	db                 store.Store
	conf               config.Config
	envoyClient        api.Client
	counterpartyClient api.Client
)

//go:embed fixtures/*
var fixtures embed.FS

func init() {
	// Initializes zerolog with our default logging requirements
	zerolog.TimeFieldFormat = time.TimeOnly
	zerolog.TimestampFieldName = "time"
	zerolog.MessageFieldName = "msg"
	zerolog.SetGlobalLevel(zerolog.DebugLevel)

	log.Logger = zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).With().Timestamp().Logger()
}

func main() {
	godotenv.Load()

	app := cli.NewApp()
	app.Name = "fsi"
	app.Usage = "initialize the local development environment for testing purposes"
	app.Version = pkg.Version()
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:    "log-level",
			Aliases: []string{"l"},
			Usage:   "set the log level for the test runner",
			EnvVars: []string{"FSI_LOG_LEVEL"},
			Value:   "debug",
		},
		&cli.StringFlag{
			Name:    "envoy-endpoint",
			Aliases: []string{"E"},
			Usage:   "the endpoint of the envoy node",
			EnvVars: []string{"FSI_ENVOY_ENDPOINT"},
		},
		&cli.StringFlag{
			Name:    "envoy-client-id",
			Aliases: []string{"ECID"},
			Usage:   "the client id of the envoy node",
			EnvVars: []string{"FSI_ENVOY_CLIENT_ID"},
		},
		&cli.StringFlag{
			Name:    "envoy-secret",
			Aliases: []string{"ES"},
			Usage:   "the api client secret of the envoy node",
			EnvVars: []string{"FSI_ENVOY_SECRET"},
		},
		&cli.StringFlag{
			Name:    "counterparty-endpoint",
			Aliases: []string{"CE"},
			Usage:   "the endpoint of the counterparty node",
			EnvVars: []string{"FSI_COUNTERPARTY_ENDPOINT"},
		},
		&cli.StringFlag{
			Name:    "counterparty-client-id",
			Aliases: []string{"CCID"},
			Usage:   "the client id of the counterparty node",
			EnvVars: []string{"FSI_COUNTERPARTY_CLIENT_ID"},
		},
		&cli.StringFlag{
			Name:    "counterparty-secret",
			Aliases: []string{"CS"},
			Usage:   "the api client secret of the counterparty node",
			EnvVars: []string{"FSI_COUNTERPARTY_SECRET"},
		},
	}
	app.Commands = []*cli.Command{
		{
			Name:     "gds:init",
			Usage:    "populate the local GDS with the docker-compose node information",
			Action:   initGDS,
			Before:   connectDB,
			After:    closeDB,
			Category: "localhost",
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "reset",
					Aliases: []string{"r"},
					Usage:   "delete an existing database before resetting it",
				},
			},
		},
		{
			Name:     "gds:inspect",
			Usage:    "check the contents of the local GDS",
			Action:   inspectGDS,
			Before:   connectDB,
			After:    closeDB,
			Category: "localhost",
		},
		{
			Name:     "db:reset",
			Usage:    "delete database rows except for authentication and api keys",
			Action:   dbReset,
			Category: "localhost",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "dsn",
					Aliases:  []string{"d"},
					Usage:    "the DSN to the database file, e.g. sqlite3:///tmp/envoy/trisa.db",
					Required: true,
					EnvVars:  []string{"TRISA_DATABASE_URL"},
				},
				&cli.StringSliceFlag{
					Name:    "exclude",
					Aliases: []string{"e"},
					Usage:   "tables to exclude from truncations (must specify all)",
					Value:   cli.NewStringSlice("migrations", "roles", "permissions", "role_permissions", "users", "api_keys", "api_key_permissions"),
				},
			},
		},
		{
			Name:     "fixtures:contacts",
			Usage:    "create the contacts fixtures using the envoy and counterparty APIs",
			Before:   connectClients,
			Action:   fixturesContacts,
			Category: "localhost",
		},
		{
			Name:     "tests:run",
			Usage:    "run all integration tests with specified configuration",
			Action:   integrationTests,
			Before:   connectClients,
			Category: "tests",
		},
		{
			Name:     "tests:sunrise",
			Usage:    "initiate a sunrise transaction with the envoy node and an email",
			Action:   sendSunrise,
			Before:   connectEnvoy,
			Category: "tests",
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "email",
					Aliases:  []string{"e"},
					Usage:    "the email address to send the sunrise message to",
					Required: true,
				},
				&cli.StringFlag{
					Name:    "counterparty",
					Aliases: []string{"c"},
					Usage:   "the counterparty of the person with the email address",
					Value:   "Sunrise VASP",
				},
			},
		},
	}

	app.Run(os.Args)
}

//===========================================================================
// Localhost Actions
//===========================================================================

func initGDS(c *cli.Context) (err error) {
	ctx := context.Background()
	envoyID, counterpartyID := "", ""

	if c.Bool("reset") {
		if err = closeDB(c); err != nil {
			return err
		}

		var dsn *store.DSN
		if dsn, err = store.ParseDSN(conf.Database.URL); err != nil {
			return cli.Exit(err, 1)
		}

		if dsn.Scheme != "leveldb" {
			return cli.Exit("can only delete leveldb databases", 1)
		}

		if dsn.Path == "" {
			return cli.Exit("cannot identify path to database", 1)
		}

		if err = os.RemoveAll(dsn.Path); err != nil {
			return cli.Exit(err, 1)
		}

		if err = connectDB(c); err != nil {
			return err
		}
	}

	// Create VASP record for envoy node
	var envoy *pb.VASP
	if envoy, err = envoyVASP(); err != nil {
		return cli.Exit(fmt.Errorf("could not read envoy record: %w", err), 1)
	}

	if envoyID, err = db.CreateVASP(ctx, envoy); err != nil {
		return cli.Exit(fmt.Errorf("could not create envoy record: %w", err), 1)
	}
	fmt.Printf("created envoy record in local gds with id: %s\n", envoyID)

	// Create VASP record for counterparty node
	var counterparty *pb.VASP
	if counterparty, err = counterpartyVASP(); err != nil {
		return cli.Exit(fmt.Errorf("could not read counterparty record: %w", err), 1)
	}

	if counterpartyID, err = db.CreateVASP(ctx, counterparty); err != nil {
		return cli.Exit(fmt.Errorf("could not create counterparty record: %w", err), 1)
	}
	fmt.Printf("created counterparty record in local gds with id: %s\n", counterpartyID)

	return nil
}

func inspectGDS(c *cli.Context) (err error) {
	iter := db.ListVASPs(context.Background())
	for iter.Next() {
		vasp, err := iter.VASP()
		if err != nil {
			return cli.Exit(err, 1)
		}
		fmt.Printf("%s %s\n", vasp.CommonName, vasp.VerificationStatus)
	}
	return nil
}

func dbReset(c *cli.Context) (err error) {
	var db *sql.DB
	if db, err = openDB(c.String("dsn")); err != nil {
		return cli.Exit(err, 1)
	}
	defer db.Close()

	var tx *sql.Tx
	if tx, err = db.Begin(); err != nil {
		return cli.Exit(fmt.Errorf("could not start transaction: %w", err), 1)
	}
	defer tx.Rollback()

	if err = resetDB(tx, c.StringSlice("exclude")); err != nil {
		return cli.Exit(err, 1)
	}

	if err = tx.Commit(); err != nil {
		return cli.Exit(fmt.Errorf("could not commit transaction: %w", err), 1)
	}
	return nil
}

func fixturesContacts(c *cli.Context) (err error) {
	if err = createContacts(envoyClient, "US"); err != nil {
		return cli.Exit(err, 1)
	}

	if err = createContacts(counterpartyClient, "DE"); err != nil {
		return cli.Exit(err, 1)
	}

	return nil
}

//===========================================================================
// Integration Tests
//===========================================================================

func integrationTests(c *cli.Context) (err error) {
	log.Debug().Msg("running integration tests")
	passed, failed := 0, 0

	tests := map[string]func() error{
		"trisa workflow approved": testTRISAWorkflow_Approve,
		"trisa workflow rejected": testTRISAWorkflow_Reject,
		"trisa workflow repair":   testTRISAWorkflow_Repair,
	}

	for name, testf := range tests {
		if err = testf(); err != nil {
			failed++
			log.Warn().Err(err).Bool("failed", true).Str("name", name).Msg("test failed")
		} else {
			passed++
			log.Info().Bool("passed", true).Str("name", name).Msg("test passed")
		}
	}

	log.Info().Int("passed", passed).Int("failed", failed).Msg("integration tests complete")
	return nil
}

func sendSunrise(c *cli.Context) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()

	network := networks[rand.Intn(len(networks))]

	sunrise := &api.Sunrise{
		Email:        c.String("email"),
		Counterparty: c.String("counterparty"),
		Originator:   makeRandPerson("US", network),
		Beneficiary:  makeRandPerson("DE", network),
		Transfer:     makeTransfer(network),
	}

	var txn *api.Transaction
	if txn, err = envoyClient.SendSunrise(ctx, sunrise); err != nil {
		return cli.Exit(fmt.Errorf("could not send sunrise transaction: %w", err), 1)
	}

	fmt.Printf("sunrise transaction %s created\n", txn.ID)
	return nil
}

//===========================================================================
// Before and After
//===========================================================================

func configure(*cli.Context) (err error) {
	conf = config.Config{
		DirectoryID: "trisatest.dev",
		Maintenance: true,
		Database: dbconf.StoreConfig{
			URL:           "leveldb:///tmp/gds/db",
			ReindexOnBoot: false,
			Insecure:      true,
		},
	}

	logger.Discard()
	return nil
}

func connectDB(c *cli.Context) (err error) {
	// Configure the connection to the local database
	if err = configure(c); err != nil {
		return err
	}

	// Connect to the trtl server and create a store to access data directly like GDS
	if db, err = store.Open(conf.Database); err != nil {
		return cli.Exit(fmt.Errorf("could not open store: %w", err), 1)
	}
	return nil
}

func closeDB(c *cli.Context) (err error) {
	if err = db.Close(); err != nil {
		return cli.Exit(err, 2)
	}
	return nil
}

func connectClients(c *cli.Context) (err error) {
	if err = setLogLevel(c); err != nil {
		return cli.Exit(err, 1)
	}

	if err = connectEnvoy(c); err != nil {
		return cli.Exit(fmt.Errorf("could not connect to envoy: %w", err), 1)
	}

	if err = connectCounterparty(c); err != nil {
		return cli.Exit(fmt.Errorf("could not connect to counterparty: %w", err), 1)
	}

	return nil
}

func setLogLevel(c *cli.Context) (err error) {
	var level logger.LevelDecoder
	if err = level.Decode(c.String("log-level")); err != nil {
		return fmt.Errorf("could not set log level: %w", err)
	}

	zerolog.SetGlobalLevel(zerolog.Level(level))
	return nil
}

func connectEnvoy(c *cli.Context) (err error) {
	var endpoint string
	if endpoint = c.String("envoy-endpoint"); endpoint == "" {
		return cli.Exit("missing endpoint", 1)
	}

	log.Trace().Str("endpoint", endpoint).Msg("connecting to envoy")
	if envoyClient, err = api.New(endpoint); err != nil {
		return cli.Exit(err, 1)
	}

	creds := &api.APIAuthentication{
		ClientID:     c.String("envoy-client-id"),
		ClientSecret: c.String("envoy-secret"),
	}

	if creds.ClientID == "" || creds.ClientSecret == "" {
		return cli.Exit("missing client id or client secret", 1)
	}

	if _, err = envoyClient.Authenticate(context.Background(), creds); err != nil {
		return cli.Exit(fmt.Errorf("could not authenticate: %w", err), 1)
	}

	log.Debug().Str("endpoint", endpoint).Msg("connected to envoy")
	return nil
}

func connectCounterparty(c *cli.Context) (err error) {
	var endpoint string
	if endpoint = c.String("counterparty-endpoint"); endpoint == "" {
		return cli.Exit("missing endpoint", 1)
	}

	log.Trace().Str("endpoint", endpoint).Msg("connecting to counterparty")
	if counterpartyClient, err = api.New(endpoint); err != nil {
		return cli.Exit(err, 1)
	}

	creds := &api.APIAuthentication{
		ClientID:     c.String("counterparty-client-id"),
		ClientSecret: c.String("counterparty-secret"),
	}

	if creds.ClientID == "" || creds.ClientSecret == "" {
		return cli.Exit("missing client id or client secret", 1)
	}

	if _, err = counterpartyClient.Authenticate(context.Background(), creds); err != nil {
		return cli.Exit(fmt.Errorf("could not authenticate with counterparty: %w", err), 1)
	}

	log.Debug().Str("endpoint", endpoint).Msg("connected to counterparty")
	return nil
}

//===========================================================================
// Helper Functions
//===========================================================================

func unmarshalPBFixture(name string, obj proto.Message) (err error) {
	if !strings.HasPrefix(name, "fixtures") {
		name = "fixtures/" + name
	}

	var data []byte
	if data, err = fixtures.ReadFile(name); err != nil {
		return err
	}

	json := protojson.UnmarshalOptions{
		AllowPartial:   true,
		DiscardUnknown: false,
	}

	return json.Unmarshal(data, obj)
}

func unmarshalJSONFixture(name string, obj any) (err error) {
	if !strings.HasPrefix(name, "fixtures") {
		name = "fixtures/" + name
	}

	var data []byte
	if data, err = fixtures.ReadFile(name); err != nil {
		return err
	}

	return json.Unmarshal(data, obj)
}
