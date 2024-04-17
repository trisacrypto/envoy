package main

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"database/sql"
	"encoding/pem"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/trisacrypto/envoy/pkg/node"
	"github.com/trisacrypto/envoy/pkg/store"
	"github.com/trisacrypto/envoy/pkg/store/models"
	"github.com/trisacrypto/envoy/pkg/web/auth"
	permiss "github.com/trisacrypto/envoy/pkg/web/auth/permissions"

	"github.com/trisacrypto/envoy/pkg/config"

	"github.com/trisacrypto/envoy/pkg"

	"github.com/joho/godotenv"
	"github.com/oklog/ulid/v2"
	confire "github.com/rotationalio/confire/usage"
	"github.com/urfave/cli/v2"
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
	app.Version = pkg.Version()
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

	password := auth.AlphaNumeric(12)
	if user.Password, err = auth.CreateDerivedKey(password); err != nil {
		return cli.Exit(err, 1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.CreateUser(ctx, user); err != nil {
		return cli.Exit(err, 1)
	}

	fmt.Printf("created user %s with role %s\npassword: %s\n", user.Email, c.String("role"), password)
	return nil
}

func createAPIKey(c *cli.Context) (err error) {
	if c.NArg() == 0 {
		return cli.Exit("specify permissions as arguments or \"all\" for all permissions", 1)
	}

	key := &models.APIKey{
		ClientID: auth.KeyID(),
	}

	secret := auth.Secret()
	if key.Secret, err = auth.CreateDerivedKey(secret); err != nil {
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

	if err = db.CreateAPIKey(ctx, key); err != nil {
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

//===========================================================================
// Helper Functions
//===========================================================================

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
