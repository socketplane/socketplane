package daemon

import (
	"encoding/json"
	"os"
	"os/signal"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/socketplane/socketplane/datastore"
	"github.com/socketplane/socketplane/network"
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
	if ctx.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	var bindInterface string
	if ctx.String("iface") != "auto" {
		bindInterface = ctx.String("iface")
	} else {
		intf := InterfaceToBind()
		if intf != nil {
			bindInterface = intf.Name
		}
	}
	log.Debugf("Binding to %s", bindInterface)
	go ServeAPI(d)
	go func() {
		ovs.CreateBridge("")
		d.populateConnections()
		datastore.Init(bindInterface, ctx.Bool("bootstrap"))
		network.CreateDefaultNetwork(ovs.OvsBridge.Subnet)
		Bonjour(bindInterface)
	}()
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			os.Exit(0)
		}
	}()
	select {}
}

func (d *Daemon) populateConnections() {
	for key, val := range ovs.ContextCache {
		connection := &Connection{}
		err := json.Unmarshal([]byte(val), connection)
		if err == nil {
			d.Connections[key] = connection
		}
	}
}
