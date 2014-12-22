package daemon

import (
	"encoding/json"
	"net"
	"os"
	"os/signal"
	"time"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
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
	if ctx.Bool("debug") {
		log.SetLevel(log.DebugLevel)
	}

	var bindInterface string
	if ctx.String("iface") != "auto" {
		bindInterface = ctx.String("iface")
	} else {
		intf := identifyInterfaceToBind()
		if intf != nil {
			bindInterface = intf.Name
		}
	}
	if bindInterface != "" {
		log.Printf("Binding to %s", bindInterface)
	} else {
		log.Errorf("Unable to identify any Interface to Bind to. Going with Defaults")
	}
	go ServeAPI(d)
	go func() {
		err := ovs.CreateBridge()
		if err != nil {
			log.Error(err.Error)
		}
		d.populateConnections()
		datastore.Init(bindInterface, ctx.Bool("bootstrap"))
		_, err = ovs.CreateDefaultNetwork()
		if err != nil {
			log.Error(err.Error)
		}
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

func identifyInterfaceToBind() *net.Interface {
	// If the user isnt binding an interface using --iface option and let the daemon to
	// identify the interface, the daemon will try its best to identify the best interface
	// for the job.
	// In a few auto-install / zerotouch config scenarios, eligible interfaces may
	// be identified after the socketplane daemon is up and running.

	for {
		intf := InterfaceToBind()
		if intf != nil {
			return intf
		}
		time.Sleep(time.Second * 5)
		log.Infof("Identifying interface to bind ... Use --iface option for static binding")
	}
	return nil
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
