package daemon

import (
	"encoding/json"
	"errors"
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
	bootstrapNode := ctx.Bool("bootstrap")
	serialChan := make(chan bool)
	bindChan = make(chan string)

	var bindInterface string
	go ServeAPI(d)
	go func() {
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
		Bonjour(bindInterface)
		if clusterListener == "" {
			bindChan <- bindInterface
		}
	}()

	go func() {
		for {
			bindInterface = <-bindChan
			if bindInterface == clusterListener {
				continue
			}
			once := true
			if clusterListener != "" {
				once = false
				datastore.Leave()
				time.Sleep(time.Second * 5)
			}
			clusterListener = bindInterface
			datastore.Init(clusterListener, bootstrapNode)
			if !bootstrapNode && once {
				serialChan <- true
			}
		}
	}()

	go func() {
		if !bootstrapNode {
			log.Printf("Non-Bootstrap node waiting on peer discovery")
			<-serialChan
			log.Printf("Non-Bootstrap node admitted into cluster")
		}
		err := ovs.CreateBridge()
		if err != nil {
			log.Error(err.Error)
		}
		d.populateConnections()
		_, err = ovs.CreateDefaultNetwork()
		if err != nil {
			log.Error(err.Error)
		}
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

var bindChan chan string
var clusterListener string

func ConfigureClusterListenerPort(listen string) error {
	iface, err := net.InterfaceByName(listen)
	if err != nil {
		return err
	}
	if iface.Flags&net.FlagUp == 0 {
		return errors.New("Interface is down")
	}
	bindChan <- listen
	return nil
}

func identifyInterfaceToBind() *net.Interface {
	// If the user isnt binding an interface using --iface option and let the daemon to
	// identify the interface, the daemon will try its best to identify the best interface
	// for the job.
	// In a few auto-install / zerotouch config scenarios, eligible interfaces may
	// be identified after the socketplane daemon is up and running.

	for {
		var intf *net.Interface
		if clusterListener != "" {
			intf, _ = net.InterfaceByName(clusterListener)
		} else {
			intf = InterfaceToBind()
		}
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
