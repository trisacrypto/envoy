package main

import (
	"os"
	"self-hosted-node/pkg"
	"self-hosted-node/pkg/config"
	"self-hosted-node/pkg/node"
	"text/tabwriter"

	"github.com/joho/godotenv"
	confire "github.com/rotationalio/confire/usage"
	"github.com/urfave/cli/v2"
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
	}

	app.Run(os.Args)
}

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
