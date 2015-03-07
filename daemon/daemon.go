package daemon

import (
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"os"
	"os/signal"
	"time"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/codegangsta/cli"
	"github.com/socketplane/socketplane/config"
)

type Daemon struct {
	Configuration   *Configuration
	Connections     map[string]*Connection
	cC              chan *ConnectionContext
	bindChan        chan *ClusterContext
	clusterListener string
	serialChan      chan bool
	bootstrapNode   bool
}

func NewDaemon() *Daemon {
	return &Daemon{
		&Configuration{},
		map[string]*Connection{},
		make(chan *ConnectionContext),
		make(chan *ClusterContext),
		"",
		make(chan bool),
		false,
	}
}

const (
	ClusterBind = iota
	ClusterJoin
	ClusterLeave
)

type ClusterContext struct {
	Param  string
	Action int
}

func Initialize() {
	OvsInit()
}

func isDebugEnabled(ctx *cli.Context) bool {
	if ctx.App != nil && !ctx.IsSet(config.CFG_DEBUG) {
		return config.Daemon.Debug
	}
	return ctx.Bool(config.CFG_DEBUG)
}

func isBootstrapNode(ctx *cli.Context) bool {
	if ctx.App != nil && !ctx.IsSet(config.CFG_BOOTSTRAP) {
		return config.Daemon.Bootstrap
	}
	return ctx.Bool(config.CFG_BOOTSTRAP)
}

func (d *Daemon) Run(ctx *cli.Context) {
	if isDebugEnabled(ctx) {
		log.SetLevel(log.DebugLevel)
	}
	d.bootstrapNode = isBootstrapNode(ctx)

	if err := os.Mkdir("/var/run/netns", 0777); err != nil {
		fmt.Println("mkdir /var/run/netns failed", err)
	}

	go ServeAPI(d)
	go func() {
		var bindInterface string
		if ctx.String("iface") != "auto" {
			bindInterface = ctx.String("iface")
		} else {
			intf := d.identifyInterfaceToBind()
			if intf != nil {
				bindInterface = intf.Name
			}
		}
		if bindInterface != "" {
			log.Printf("Binding to %s", bindInterface)
			d.clusterListener = bindInterface
		} else {
			log.Errorf("Unable to identify any Interface to Bind to. Going with Defaults")
		}
		InitDatastore(bindInterface, d.bootstrapNode)
		Bonjour(bindInterface)
		if !d.bootstrapNode {
			d.serialChan <- true
		}
	}()

	go ClusterRPCHandler(d)

	go func() {
		if !d.bootstrapNode {
			log.Printf("Non-Bootstrap node waiting on peer discovery")
			<-d.serialChan
			log.Printf("Non-Bootstrap node admitted into cluster")
		}
		err := CreateBridge()
		if err != nil {
			log.Error(err.Error)
		}
		d.populateConnections()
		_, err = CreateDefaultNetwork()
		if err != nil {
			log.Error(err.Error)
		}
	}()

	go ConnectionRPCHandler(d)

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			os.Exit(0)
		}
	}()
	select {}
}

func (d *Daemon) ConfigureClusterListenerPort(listen string) error {
	iface, err := net.InterfaceByName(listen)
	if err != nil {
		log.Debugf("Could not find interface %s", listen)
		return err
	}
	if iface.Flags&net.FlagUp == 0 {
		log.Debugf("%s is down", listen)
		return errors.New("Interface is down")
	}
	log.Debugf("Requesting to bind to %s", listen)
	context := &ClusterContext{listen, ClusterBind}
	d.bindChan <- context
	return nil
}

func (d *Daemon) JoinCluster(address string) error {
	if addr := net.ParseIP(address); addr == nil {
		return errors.New("Invalid IP address")
	}
	log.Debugf("Requesting to join cluster %s", address)
	context := &ClusterContext{address, ClusterJoin}
	d.bindChan <- context
	return nil
}

func ClusterRPCHandler(d *Daemon) {
	for {
		context := <-d.bindChan
		switch context.Action {
		case ClusterBind:
			bindInterface := context.Param
			err := clusterBindRPC(d, bindInterface)
			if err != nil {
				break
			}
		case ClusterJoin:
			joinAddress := context.Param
			err := clusterJoinRPC(d, joinAddress)
			if err != nil {
				break
			}
		case ClusterLeave:
			if err := LeaveDatastore(); err != nil {
				log.Errorf("Error leaving cluster. %s", err.Error())
				break
			}
		}
	}
}

func clusterBindRPC(d *Daemon, bindInterface string) error {
	if bindInterface == d.clusterListener {
		log.Debug("Bind Interface is the same as currently bound interface")
		return errors.New("Bind Interface is the same as currently bound interface")
	}
	once := true
	if d.clusterListener != "" {
		log.Debug("Cluster is already bound on another interface. Leaving...")
		once = false
		LeaveDatastore()
		time.Sleep(time.Second * 5)
	}
	log.Debugf("Setting new cluster listener to %s", bindInterface)
	d.clusterListener = bindInterface
	InitDatastore(d.clusterListener, d.bootstrapNode)

	if !d.bootstrapNode && once {
		d.serialChan <- true
	}
	return nil
}

func clusterJoinRPC(d *Daemon, joinAddress string) error {
	bindInterface, err := GetIfaceForRoute(joinAddress)
	if err != nil {
		return err
	}
	if d.clusterListener != bindInterface {
		log.Debug("Cluster is already bound on another interface. Leaving...")
		LeaveDatastore()
		time.Sleep(time.Second * 10)
		log.Debugf("Setting new cluster listener to %s", bindInterface)
		d.bootstrapNode = false
		d.clusterListener = bindInterface
		InitDatastore(d.clusterListener, d.bootstrapNode)
	}
	if err = JoinDatastore(joinAddress); err != nil {
		log.Errorf("Could not join cluster %s. %s", joinAddress, err.Error())
	}
	return nil
}

func (d *Daemon) identifyInterfaceToBind() *net.Interface {
	// If the user isnt binding an interface using --iface option and let the daemon to
	// identify the interface, the daemon will try its best to identify the best interface
	// for the job.
	// In a few auto-install / zerotouch config scenarios, eligible interfaces may
	// be identified after the socketplane daemon is up and running.

	for {
		var intf *net.Interface
		if d.clusterListener != "" {
			intf, _ = net.InterfaceByName(d.clusterListener)
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
	for key, val := range ContextCache {
		connection := &Connection{}
		err := json.Unmarshal([]byte(val), connection)
		if err == nil {
			d.Connections[key] = connection
		}
	}
}
