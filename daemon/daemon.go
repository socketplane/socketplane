package daemon

import (
	"os"
	"os/signal"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/socketplane/socketplane/datastore"
	"github.com/socketplane/socketplane/ovs"
)

type Daemon struct {
	Configuration *Configuration
	Connections   map[string]*Connection
}

func NewDaemon() *Daemon {
	return &Daemon{
		&Configuration{},
		map[string]*Connection{},
	}
}

func (d *Daemon) Run(ctx *cli.Context) {
	var bindInterface string
	if ctx.String("iface") != "auto" {
		bindInterface = ctx.String("iface")
	} else {
		intf := InterfaceToBind()
		if intf != nil {
			bindInterface = intf.Name
		}
	}
	go ServeAPI(d)
	go ovs.CreateBridge("")
	go Bonjour(bindInterface)
	go datastore.Init(bindInterface, ctx.Bool("bootstrap"))
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			os.Exit(0)
		}
	}()
	select {}
}
