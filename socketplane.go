package main

import (
	"os"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/socketplane/socketplane/daemon"
)

func main() {
	d := daemon.NewDaemon()

	app := cli.NewApp()
	app.Name = "socketplane"
	app.Usage = "linux container networking"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "iface",
			Value: "auto",
			Usage: "Name of the interface to bind to. The default is to auto select",
		},
		cli.BoolFlag{
			Name:  "bootstrap",
			Usage: "Set --bootstrap for the first socketplane instance being started",
		},
	}
	app.Action = d.Run
	app.Run(os.Args)
}
