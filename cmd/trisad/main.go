package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"
	"time"

	"self-hosted-node/pkg"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/node"
	"self-hosted-node/pkg/store"
	"self-hosted-node/pkg/store/models"
	"self-hosted-node/pkg/web/auth"
	permiss "self-hosted-node/pkg/web/auth/permissions"

	"github.com/joho/godotenv"
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
	app.Name = "trisad"
	app.Usage = "serve and manage the TRISA self-hosted node"
	app.Version = pkg.Version()
	app.Flags = []cli.Flag{}
	app.Commands = []*cli.Command{
		{
			Name:     "serve",
			Usage:    "serve the TRISA node server configured from the environment",
			Action:   serve,
			Category: "server",
		},
		{
			Name:     "config",
			Usage:    "print TRISA node server configuration guide",
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
