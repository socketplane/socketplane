package daemon

import (
	"flag"
	"net"
	"testing"
	"time"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/codegangsta/cli"
)

func TestDaemonRun(t *testing.T) {
	bootstrap := true
	debug := true
	iface := "auto"

	set := flag.NewFlagSet("test", 0)
	set.Bool("debug", debug, "")
	set.String("iface", iface, "")
	set.Bool("bootstrap", bootstrap, "")
	c := cli.NewContext(nil, set, set)

	daemon := NewDaemon()
	go daemon.Run(c)

	// Wait for subsystems to come up
	time.Sleep(3 * time.Second)

	if daemon.bootstrapNode != bootstrap {
		t.Fatalf("bootstrap value is incorrect")
	}

	if log.GetLevel() != log.DebugLevel {
		t.Fatalf("incorrect log level")
	}

	_, err := net.InterfaceByName(defaultBridgeName)
	if err != nil {
		t.Fatal("bridge not created")
	}

	_, err = GetNetwork("default")
	if err != nil {
		t.Fatal("default network not created")
	}

}

func TestClusterBindRPC(t *testing.T) {
	d := NewDaemon()
	d.clusterListener = "foo1"
	err := clusterBindRPC(d, "foo1")
	if err == nil {
		t.Fatal("this should produce an error")
	}
	err = clusterBindRPC(d, "eth0")
	if err != nil {
		t.Fatal("this should work")
	}

	if d.clusterListener != "eth0" {
		t.Fatal("field not updated")
	}
	LeaveDatastore()
}

func TestClusterJoinRPC(t *testing.T) {
	d := NewDaemon()
	err := clusterJoinRPC(d, "foobar")
	if err == nil {
		t.Fatal("this should not work")
	}
	err = clusterJoinRPC(d, "10.0.0.1")
	if err != nil {
		t.Fatal("while this won't work, we shouldn't get errors")
	}
	if d.clusterListener != "eth0" {
		t.Fatal("field not updated")
	}
	LeaveDatastore()
}

//ToDo: @mavenugo to write this test
func TestPopulateConnection(t *testing.T) {
	t.Skip("")
}
