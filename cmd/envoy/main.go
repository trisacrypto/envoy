package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"github.com/trisacrypto/envoy/pkg/audit"
	"github.com/trisacrypto/envoy/pkg/enum"
	"github.com/trisacrypto/envoy/pkg/node"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/store/dsn"
	dberr "github.com/trisacrypto/envoy/pkg/store/errors"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/store/sqlite"
	"github.com/trisacrypto/envoy/pkg/trisa/keychain"
	"github.com/trisacrypto/envoy/pkg/web/api/v1"
	"github.com/trisacrypto/envoy/pkg/web/auth/passwords"
	permiss "github.com/trisacrypto/envoy/pkg/web/auth/permissions"

	"github.com/trisacrypto/envoy/pkg/config"

	"github.com/trisacrypto/envoy/pkg"

	"github.com/joho/godotenv"
	confire "github.com/rotationalio/confire/usage"
	"github.com/urfave/cli/v2"
	"go.rtnl.ai/ulid"
)

var (
	db   store.Store
	conf config.Config
)

func main() {
	godotenv.Load()

	app := cli.NewApp()
	app.Name = "envoy"
	app.Usage = "serve and manage the TRISA Envoy self-hosted node"
	app.Version = pkg.Version(false)
	app.Flags = []cli.Flag{}
	app.Commands = []*cli.Command{
		{
			Name:     "serve",
			Usage:    "serve the TRISA Envoy node server configured from the environment",
			Action:   serve,
			Category: "server",
		},
		{
			Name:     "config",
			Usage:    "print TRISA Envoy node server configuration guide",
			Category: "server",
			Action:   usage,
			Flags: []cli.Flag{
				&cli.BoolFlag{
					Name:    "list",
					Aliases: []string{"l"},
					Usage:   "print in list mode instead of table mode",
				},
			},
		},
		{
			Name:     "remigrate",
			Usage:    "attempt to re-apply the schema and recover the original data",
			Category: "server",
			Action:   remigrate,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "backup",
					Aliases: []string{"b"},
					Usage:   "backup location for database, defaults to same directory with .bak extension",
				},
				&cli.StringFlag{
					Name:    "db",
					Aliases: []string{"d"},
					Usage:   "path to database, defaults to configured location",
				},
			},
		},
		{
			Name:     "createuser",
			Usage:    "create a new user to access Envoy with",
			Category: "admin",
			Before:   openDB,
			Action:   createUser,
			After:    closeDB,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "name",
					Aliases: []string{"n"},
					Usage:   "full name of user",
				},
				&cli.StringFlag{
					Name:     "email",
					Aliases:  []string{"e"},
					Required: true,
					Usage:    "email address of user",
				},
				&cli.StringFlag{
					Name:    "role",
					Aliases: []string{"r"},
					Value:   "compliance",
					Usage:   "user role for permissions [admin, compliance, observer]",
				},
			},
		},
		{
			Name:     "pwreset",
			Usage:    "reset a user's password",
			Category: "admin",
			Before:   openDB,
			Action:   resetPassword,
			After:    closeDB,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "email",
					Aliases:  []string{"e"},
					Required: true,
					Usage:    "email address of user",
				},
			},
		},
		{
			Name:      "createapikey",
			Usage:     "create a new api key with the specified permissions",
			Category:  "admin",
			Before:    openDB,
			Action:    createAPIKey,
			After:     closeDB,
			Args:      true,
			ArgsUsage: "all | permission [permission ...]",
		},
		{
			Name:     "tokenkey",
			Usage:    "generate an RSA token key pair and ulid for JWT token signing",
			Category: "admin",
			Action:   generateTokenKey,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "out",
					Aliases: []string{"o"},
					Usage:   "path to write keys out to (optional, will be saved as ulid.pem by default)",
				},
				&cli.IntFlag{
					Name:    "size",
					Aliases: []string{"s"},
					Usage:   "number of bits for the generated keys",
					Value:   4096,
				},
			},
		},
		{
			Name:     "hmackey",
			Usage:    "generate an HMAC key and keyID for webhook authentication",
			Category: "admin",
			Action:   generateHMACKey,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:    "out",
					Aliases: []string{"o"},
					Usage:   "path to write json file with key and keyID out to",
				},
				&cli.IntFlag{
					Name:    "size",
					Aliases: []string{"s"},
					Usage:   "number of bytes for the generated key",
					Value:   32,
				},
			},
		},
		{
			Name:     "daybreak:import",
			Usage:    "Import Daybreak counterparties from a JSON file that contains a list of Counterparty objects",
			Category: "admin",
			Before:   openDB,
			Action:   daybreakImport,
			After:    closeDB,
			Flags: []cli.Flag{
				&cli.StringFlag{
					Name:     "in",
					Aliases:  []string{"i"},
					Usage:    "Specify the path to a JSON file that contains a list of Counterparty objects",
					Required: true,
				},
			},
		},
		{
			Name:     "daybreak:retire",
			Usage:    "Deletes any Daybreak counterparties and contacts which do not have any transactions associated with them",
			Category: "admin",
			Before:   openDB,
			Action:   daybreakRetire,
			After:    closeDB,
		},
	}

	app.Run(os.Args)
}

//===========================================================================
// Server Commands
//===========================================================================

func serve(c *cli.Context) (err error) {
	var conf config.Config
	if conf, err = config.New(); err != nil {
		return cli.Exit(err, 1)
	}

	var trisa *node.Node
	if trisa, err = node.New(conf); err != nil {
		return cli.Exit(err, 1)
	}

	if err = trisa.Serve(); err != nil {
		return cli.Exit(err, 1)
	}
	return nil
}

func usage(c *cli.Context) error {
	tabs := tabwriter.NewWriter(os.Stdout, 1, 0, 4, ' ', 0)
	format := confire.DefaultTableFormat
	if c.Bool("list") {
		format = confire.DefaultListFormat
	}

	var conf config.Config
	if err := confire.Usagef(config.Prefix, &conf, tabs, format); err != nil {
		return cli.Exit(err, 1)
	}

	tabs.Flush()
	return nil
}

func remigrate(c *cli.Context) (err error) {
	var orig, back string
	if orig = c.String("db"); orig == "" {
		// Load the source from the envoy config
		if conf, err = config.New(); err != nil {
			return cli.Exit(err, 1)
		}

		var uri *dsn.DSN
		if uri, err = dsn.Parse(conf.DatabaseURL); err != nil {
			return cli.Exit(err, 1)
		}

		orig = uri.Path
		if orig == "" {
			return cli.Exit("cannot determine path to source database", 1)
		}
	}

	if back = c.String("backup"); back == "" {
		// Rename the existing database with the .bak extension
		back = orig + ".bak"
	}

	// Copy the src to the destination
	if err = os.Rename(orig, back); err != nil {
		return cli.Exit(err, 1)
	}

	var (
		srcdb, dstdb *sqlite.Store
		srctx, dsttx *sqlite.Tx
	)

	// Connect to both databases
	if srcdb, srctx, err = connectSqlite3(back); err != nil {
		return cli.Exit(fmt.Errorf("could not connect to backup database: %w", err), 1)
	}
	defer srctx.Rollback()
	defer srcdb.Close()

	if dstdb, dsttx, err = connectSqlite3(orig); err != nil {
		return cli.Exit(fmt.Errorf("could not connect to remigrated database: %w", err), 1)
	}
	defer dsttx.Rollback()
	defer dstdb.Close()

	// List all the tables that are in the src db
	var tables []string
	if tables, err = sqlite3Tables(srctx); err != nil {
		return cli.Exit(fmt.Errorf("could not list tables from source database: %w", err), 1)
	}

	repairTable := func(table string) (err error) {
		// Skip internally managed tables
		if table == "migrations" || table == "permissions" || table == "roles" || table == "role_permissions" {
			return nil
		}

		migrated, errored := 0, 0
		query := fmt.Sprintf("SELECT * FROM %s", table)

		var rows *sql.Rows
		if rows, err = srctx.Query(query); err != nil {
			return err
		}
		defer rows.Close()

		var columnNames []string
		if columnNames, err = rows.Columns(); err != nil {
			return err
		}

		insert := fmt.Sprintf(
			"INSERT INTO %s (%s) VALUES (%s)",
			table,
			strings.Join(columnNames, ","),
			strings.TrimRight(strings.Repeat("?,", len(columnNames)), ","),
		)

		for rows.Next() {
			values := make([]interface{}, len(columnNames))
			for i := range values {
				values[i] = valueForColumn(table, columnNames[i])
			}

			if err = rows.Scan(values...); err != nil {
				return err
			}

			if _, err = dsttx.Exec(insert, values...); err != nil {
				fmt.Println(err.Error())
				errored++
				continue
			}

			migrated++
		}

		fmt.Printf("migrated %d rows from %s (%d errors)\n", migrated, table, errored)
		return rows.Err()
	}

	for _, table := range tables {
		if err = repairTable(table); err != nil {
			return cli.Exit(fmt.Errorf("could not repair table %s: %w", table, err), 1)
		}
	}

	// Commit the remigration
	srctx.Commit()
	dsttx.Commit()
	fmt.Printf("envoy db remigrated from %s; backup saved at %s\n", orig, back)
	return nil
}

//===========================================================================
// Administrative Commands
//===========================================================================

var roles = map[string]int64{
	"admin":      1,
	"compliance": 2,
	"observer":   3,
}

func createUser(c *cli.Context) (err error) {
	user := &models.User{
		Name:  sql.NullString{Valid: c.String("name") != "", String: c.String("name")},
		Email: c.String("email"),
	}

	var ok bool
	if user.RoleID, ok = roles[strings.TrimSpace(strings.ToLower(c.String("role")))]; !ok {
		return cli.Exit("specify admin, compliance, or observer as the role", 1)
	}

	password := passwords.AlphaNumeric(12)
	if user.Password, err = passwords.CreateDerivedKey(password); err != nil {
		return cli.Exit(err, 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup the audit log
	if ctx, err = setupAuditLog([]byte("createUser"), ctx); err != nil {
		return cli.Exit(err, 1)
	}

	if err = db.CreateUser(ctx, user, &models.ComplianceAuditLog{
		ChangeNotes: sql.NullString{Valid: true, String: "createUser()"},
	}); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Printf("created user %s with role %s\npassword: %s\n", user.Email, c.String("role"), password)
	return nil
}

func resetPassword(c *cli.Context) (err error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var user *models.User
	if user, err = db.RetrieveUser(ctx, c.String("email")); err != nil {
		return cli.Exit(fmt.Errorf("could not retrieve user %q: %w", c.String("email"), err), 1)
	}

	password := passwords.AlphaNumeric(12)
	if user.Password, err = passwords.CreateDerivedKey(password); err != nil {
		return cli.Exit(fmt.Errorf("could not create derived key for user password: %w", err), 1)
	}

	if err = db.SetUserPassword(ctx, user.ID, user.Password); err != nil {
		return cli.Exit(fmt.Errorf("could not store password: %w", err), 1)
	}

	fmt.Printf("updated user %s with password: %s\n", user.Email, password)
	return nil
}

func createAPIKey(c *cli.Context) (err error) {
	if c.NArg() == 0 {
		return cli.Exit("specify permissions as arguments or \"all\" for all permissions", 1)
	}

	key := &models.APIKey{
		ClientID: passwords.KeyID(),
	}

	secret := passwords.Secret()
	if key.Secret, err = passwords.CreateDerivedKey(secret); err != nil {
		return cli.Exit(err, 1)
	}

	var permissions permiss.Permissions
	if c.NArg() == 1 && strings.ToLower(c.Args().First()) == "all" {
		permissions = permiss.AllPermissions[:]
	} else {
		for _, arg := range c.Args().Slice() {
			var permission permiss.Permission
			if permission, err = permiss.Parse(arg); err != nil || permission == permiss.Unknown {
				return cli.Exit(fmt.Errorf("%q is not a valid permission: %w", arg, err), 1)
			}
			permissions = append(permissions, permission)
		}
	}
	key.SetPermissions(permissions.String())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Setup the audit log
	if ctx, err = setupAuditLog([]byte("createAPIKey"), ctx); err != nil {
		return cli.Exit(err, 1)
	}

	if err = db.CreateAPIKey(ctx, key, &models.ComplianceAuditLog{
		ChangeNotes: sql.NullString{Valid: true, String: "createAPIKey()"},
	}); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Printf("created api key with %d permissions\nclient id:\t%s\nclient secret:\t%s\n", len(permissions), key.ClientID, secret)
	return nil
}

func generateTokenKey(c *cli.Context) (err error) {
	// Create ULID and determine outpath
	keyid := ulid.Make()

	var out string
	if out = c.String("out"); out == "" {
		out = fmt.Sprintf("%s.pem", keyid)
	}

	// Generate RSA keys using crypto random
	var key *rsa.PrivateKey
	if key, err = rsa.GenerateKey(rand.Reader, c.Int("size")); err != nil {
		return cli.Exit(err, 1)
	}

	// Open file to PEM encode keys to
	var f *os.File
	if f, err = os.OpenFile(out, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600); err != nil {
		return cli.Exit(err, 1)
	}

	if err = pem.Encode(f, &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	}); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Printf("RSA key id: %s -- saved with PEM encoding to %s\n", keyid, out)
	return nil
}

func generateHMACKey(c *cli.Context) (err error) {
	key := make([]byte, c.Int("size"))
	if _, err = rand.Read(key); err != nil {
		return cli.Exit(err, 1)
	}

	keyID := ulid.Make()
	keys := hex.EncodeToString(key)

	if out := c.String("out"); out != "" {
		data := map[string]string{
			"key":   keys,
			"keyID": keyID.String(),
		}

		var f *os.File
		if f, err = os.Create(out); err != nil {
			return cli.Exit(err, 1)
		}
		defer f.Close()

		if err = json.NewEncoder(f).Encode(data); err != nil {
			return cli.Exit(err, 1)
		}

	} else {
		fmt.Printf("Key ID: %s\nKey: %s\n", keyID, keys)
	}

	return nil
}

func daybreakImport(c *cli.Context) (err error) {
	// Load the JSON file into Counterparty objects
	var in string
	if in = c.String("in"); in == "" {
		return cli.Exit("must pass argument 'in' as a path to a JSON file with a list of Counterparty objects", 1)
	}

	var jb []byte
	if jb, err = os.ReadFile(in); err != nil {
		return cli.Exit(err, 1)
	}

	var cpartyImports []*api.Counterparty
	if err = json.Unmarshal(jb, &cpartyImports); err != nil {
		return cli.Exit(err, 1)
	}

	// Make sure we have a daybreak database
	ddb, ok := db.(store.DaybreakStore)
	if !ok {
		return cli.Exit("configured database does not support daybreak operations", 1)
	}

	// Create outer context for database interactions and source info.
	// NOTE: 120 seconds should be enough for ~2500 entries; if the context
	// times out, increase it
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// Setup the audit log
	if ctx, err = setupAuditLog([]byte("daybreakImport"), ctx); err != nil {
		return cli.Exit(err, 1)
	}

	// NOTE: this operation happens in its own transaction, if two daybreak imports
	// happen concurrently, then this map would not accurately reflect the state of the
	// database, so its important to ensure only a single import runs at at time.
	var srcMap map[string]*models.CounterpartySourceInfo
	if srcMap, err = ddb.ListDaybreak(ctx); err != nil {
		return cli.Exit(fmt.Sprintf("could not list daybreak couunterparties: %s", err), 1)
	}

	// Begin import; all counterparties and associated contacts that can be created will
	// be; otherwise the entire counterparty will be skipped.
	log.Info().Msgf("starting to import %d Daybreak Counterparties...", len(cpartyImports))

	var (
		updated int
		created int
	)

	for _, apiCounterparty := range cpartyImports {
		// Convert the api.Counterparty to a model.Counterparty
		var modelCounterparty *models.Counterparty
		if modelCounterparty, err = apiCounterparty.Model(); err != nil {
			log.Warn().Err(err).Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiCounterparty.Name).Msg("could not convert counterparty to model")
			continue
		}

		// Validate that this counterparty is meant for Daybreak
		if modelCounterparty.Source != enum.SourceDaybreak {
			log.Warn().Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiCounterparty.Name).Msg("source must be 'daybreak' to import")
			continue
		}
		if modelCounterparty.Protocol != enum.ProtocolSunrise {
			log.Warn().Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiCounterparty.Name).Msg("protocol must be 'sunrise' to import")
			continue
		}

		if !modelCounterparty.RegisteredDirectory.Valid || modelCounterparty.RegisteredDirectory.String != "daybreak.rotational.io" {
			log.Warn().Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiCounterparty.Name).Msg("registered directory must be 'daybreak.rotational.io' to import")
			continue
		}

		// Ensure the website has a protocol (only return if a valid string is unparseable)
		if modelCounterparty.Website.String, err = modelCounterparty.NormalizedWebsite(); err != nil {
			if !errors.Is(err, dberr.ErrNullString) {
				log.Warn().Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiCounterparty.Name).Str("website", modelCounterparty.Website.String).Msg("cannot parse counterparty website string")
				continue
			}
		}

		// Set the contacts onto the model to be added to the database.
		contacts := make([]*models.Contact, 0, len(apiCounterparty.Contacts))
		for _, apiContact := range apiCounterparty.Contacts {
			var modelContact *models.Contact
			if modelContact, err = apiContact.Model(modelCounterparty); err != nil {
				log.Warn().Err(err).Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiContact.Name).Msg("could not convert contact to model")
				continue
			}
			contacts = append(contacts, modelContact)
		}

		if len(contacts) == 0 {
			// Daybreak counterparties should always have contacts, otherwise where do emails go?
			log.Warn().Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiCounterparty.Name).Msg("counterparty has no contacts")
			continue
		}

		modelCounterparty.SetContacts(contacts)

		if cSrc, ok := srcMap[modelCounterparty.DirectoryID.String]; ok {
			// Counterparty is present in DB, so we update it -- if any part of the
			// update fails, then the transaction will be rolled back and the
			// counterparty will be left in its original state.
			modelCounterparty.ID = cSrc.ID
			apiCounterparty.ID = cSrc.ID

			if err = ddb.UpdateDaybreak(ctx, modelCounterparty, &models.ComplianceAuditLog{
				ChangeNotes: sql.NullString{Valid: true, String: "daybreakImport()"},
			}); err != nil {
				log.Warn().Err(err).Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiCounterparty.Name).Msg("could not update counterparty")
				continue
			}

			updated++
		} else {
			// Create the counterparty and all contacts together - if any part of the
			// update fails, then the transaction will be rolled back and no partial
			// record will be inserted into the database.
			if err = ddb.CreateDaybreak(ctx, modelCounterparty, &models.ComplianceAuditLog{
				ChangeNotes: sql.NullString{Valid: true, String: "daybreakImport()"},
			}); err != nil {
				log.Warn().Err(err).Str("directory_id", apiCounterparty.DirectoryID).Str("name", apiCounterparty.Name).Msg("could not create counterparty")
				continue
			}

			created++
		}

	}

	log.Info().
		Int("imported", updated+created).
		Int("errors", len(cpartyImports)-updated-created).
		Int("created", created).
		Int("updated", updated).
		Int("total", len(cpartyImports)).
		Float64("percent_success", (float64(created+updated) / float64(len(cpartyImports)) * 100.0)).
		Msg("daybreak import complete")

	return nil
}

func daybreakRetire(c *cli.Context) (err error) {
	// Make sure we have a daybreak database
	ddb, ok := db.(store.DaybreakStore)
	if !ok {
		return cli.Exit("configured database does not support daybreak operations", 2)
	}

	// Create outer context for database interactions and source info.
	// NOTE: 60 seconds should be enough for ~2500 entries; if the context
	// times out, increase it
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Setup the audit log
	if ctx, err = setupAuditLog([]byte("daybreakRetire"), ctx); err != nil {
		return cli.Exit(err, 1)
	}

	// Get all Daybreak Counterparties
	var srcMap map[string]*models.CounterpartySourceInfo
	if srcMap, err = ddb.ListDaybreak(ctx); err != nil {
		return cli.Exit(fmt.Sprintf("could not list daybreak couunterparties: %s", err), 1)
	}

	// Delete all of the counterparties (`ignoreTxns` is `false`)
	deleted := 0
	hasTxns := 0
	for _, counterparty := range srcMap {
		if err = ddb.DeleteDaybreak(ctx, counterparty.ID, false, &models.ComplianceAuditLog{
			ChangeNotes: sql.NullString{Valid: true, String: "daybreakRetire()"},
		}); err != nil {
			if err == dberr.ErrDaybreakHasTxns {
				hasTxns += 1
				log.Info().Str("id", counterparty.ID.String()).Str("directory_id", counterparty.DirectoryID.String).Msg("daybreak counterparty not deleted because it has associated transactions")
				continue
			}
			return err
		}
		deleted += 1
	}

	log.Info().
		Int("deleted", deleted).
		Int("hasTxns", hasTxns).
		Int("errors", len(srcMap)-deleted-hasTxns).
		Int("total", len(srcMap)).
		Float64("percent_success", (float64(deleted+hasTxns) / float64(len(srcMap)) * 100.0)).
		Msg("daybreak retirement complete")

	return nil
}

//===========================================================================
// Helper Functions
//===========================================================================

// Setup the context with the given actor ID (usually the function or command
// name) and sets up the keychain to sign audit logs.
func setupAuditLog(actorId []byte, ctx context.Context) (newCtx context.Context, err error) {
	// Add actor information to the context.
	newCtx = audit.WithActor(ctx, []byte(actorId), enum.ActorCLI)

	// Load the keychain from the environment config
	var (
		conf config.Config
		kc   keychain.KeyChain
	)
	if conf, err = config.New(); err != nil {
		return nil, fmt.Errorf("cannot load mTLS config: %s", err)
	}
	if kc, err = keychain.Load(&conf.Node.MTLSConfig); err != nil {
		return nil, fmt.Errorf("cannot load keychain: %s", err)
	}
	audit.UseKeyChain(kc)

	return newCtx, nil
}

func openDB(c *cli.Context) (err error) {
	if conf, err = config.New(); err != nil {
		return cli.Exit(err, 1)
	}

	if db, err = store.Open(conf.DatabaseURL); err != nil {
		return cli.Exit(err, 1)
	}

	return nil
}

func closeDB(c *cli.Context) error {
	if db != nil {
		if err := db.Close(); err != nil {
			return cli.Exit(err, 1)
		}
	}
	return nil
}

func connectSqlite3(path string) (dbs *sqlite.Store, tx *sqlite.Tx, err error) {
	var db store.Store
	if db, err = store.Open("sqlite3:///" + path); err != nil {
		return nil, nil, err
	}

	dbs = db.(*sqlite.Store)
	if tx, err = dbs.BeginTx(context.Background(), nil); err != nil {
		return nil, nil, err
	}

	return dbs, tx, nil
}

func sqlite3Tables(tx *sqlite.Tx) (tables []string, err error) {
	tables = make([]string, 0)

	var rows *sql.Rows
	if rows, err = tx.Query("SELECT name FROM sqlite_master WHERE type='table'"); err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var table string
		if err = rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}
	return tables, nil
}

func valueForColumn(table, column string) interface{} {
	if column == "created" || column == "modified" {
		return &time.Time{}
	}

	if column == "id" {
		if table == "transactions" {
			return &uuid.UUID{}
		}
		return &ulid.ULID{}
	}

	switch table {
	case "accounts":
		switch column {
		case "customer_id", "first_name", "last_name", "travel_address":
			return &sql.NullString{}
		case "ivms101":
			var data []byte
			return &data
		}
	case "crypto_addresses":
		switch column {
		case "account_id":
			return &ulid.ULID{}
		case "crypto_address", "network":
			var s string
			return &s
		case "asset_type", "tag", "travel_address":
			return &sql.NullString{}
		}
	case "users":
		switch column {
		case "name":
			return &sql.NullString{}
		case "email", "password":
			var s string
			return &s
		case "role_id":
			var i int64
			return &i
		case "last_login":
			return &sql.NullTime{}
		}
	case "api_keys":
		switch column {
		case "description":
			return &sql.NullString{}
		case "client_id", "secret":
			var s string
			return &s
		case "last_seen":
			return &sql.NullTime{}
		}
	case "api_key_permissions":
		switch column {
		case "api_key_id":
			var s string
			return &s
		case "permission_id":
			var i int64
			return &i
		}
	case "counterparties":
		switch column {
		case "source", "protocol", "common_name", "endpoint", "name":
			var s string
			return &s
		case "directory_id", "registered_directory", "website", "country", "business_category":
			return &sql.NullString{}
		case "vasp_categories", "ivms101":
			var data []byte
			return &data
		case "verified_on":
			return &sql.NullTime{}
		}
	case "transactions":
		switch column {
		case "source", "status", "counterparty", "virtual_asset":
			var s string
			return &s
		case "counterparty_id":
			return &ulid.NullULID{}
		case "originator", "originator_address", "beneficiary", "beneficiary_address":
			return &sql.NullString{}
		case "amount":
			var f float64
			return &f
		case "last_update":
			return &sql.NullTime{}
		}
	}

	panic(fmt.Errorf("unknown type for %s.%s", table, column))
}
