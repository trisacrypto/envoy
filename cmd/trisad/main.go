package main

import (
	"os"
	"self-hosted-node/pkg"

	"github.com/joho/godotenv"
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
	}

	app.Run(os.Args)
}

func serve(c *cli.Context) error {
	return cli.Exit("not implemented yet", 420)
}
