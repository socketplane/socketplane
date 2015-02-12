package main

import (
	"log"
	"os"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/socketplane/socketplane/config"
	"github.com/socketplane/socketplane/daemon"
)

func main() {
	app := cli.NewApp()
	app.Name = "socketplane"
	app.Usage = "linux container networking"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  config.CFG_FILE,
			Value: "/etc/socketplane/socketplane.toml",
			Usage: "Name of the configuration file in TOML format",
		},
		cli.StringFlag{
			Name:  config.CFG_IFACE,
			Value: "auto",
			Usage: "Name of the interface to bind to. The default is to auto select",
		},
		cli.BoolFlag{
			Name:  config.CFG_BOOTSTRAP,
			Usage: "Set --bootstrap for the first socketplane instance being started",
		},
		cli.BoolFlag{
			Name:  config.CFG_DEBUG,
			Usage: "Provide debug level logging",
		},
	}
	app.Action = Run
	app.Run(os.Args)
}

func Run(ctx *cli.Context) {
	configFilename := ctx.String(config.CFG_FILE)
	err := config.Parse(configFilename)
	if err != nil {
		log.Fatal("Unable to parse configuration file " + configFilename)
		os.Exit(1)
	}
	d := daemon.NewDaemon()
	d.Run(ctx)

}
