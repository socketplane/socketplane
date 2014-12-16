package ovs

import (
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/docker/libcontainer/netlink"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/libovsdb"
)

// shameless copy from docker/daemon/networkdriver/bridge/driver.go to reflect same behaviour
// between this temporary wrapper solution to the native Network integration

var addrs = []string{
	// Here we don't follow the convention of using the 1st IP of the range for the gateway.
	// This is to use the same gateway IPs as the /24 ranges, which predate the /16 ranges.
	// In theory this shouldn't matter - in practice there's bound to be a few scripts relying
	// on the internal addressing or other stupid things like that.
	// They shouldn't, but hey, let's not break them unless we really have to.
	"172.17.42.1/16", // Don't use 172.16.0.0/16, it conflicts with EC2 DNS 172.16.0.23
	"10.0.42.1/16",   // Don't even try using the entire /8, that's too intrusive
	"10.1.42.1/16",
	"10.42.42.1/16",
	"172.16.42.1/24",
	"172.16.43.1/24",
	"172.16.44.1/24",
	"10.0.42.1/24",
	"10.0.43.1/24",
	"192.168.42.1/24",
	"192.168.43.1/24",
	"192.168.44.1/24",
}

const defaultBridgeName = "docker0-ovs"

var ovsBridgeName string = defaultBridgeName

var ovs *libovsdb.OvsdbClient

func init() {
	var err error
	ovs, err = ovs_connect()
	if err != nil {
		log.Println("Error connecting OVS ", err)
	}
}

func SetBridgeName(name string) {
	ovsBridgeName = name
}

func CreateBridge(bridgeIP string) error {
	var ifaceAddr string
	if len(bridgeIP) != 0 {
		_, _, err := net.ParseCIDR(bridgeIP)
		if err != nil {
			return err
		}
		ifaceAddr = bridgeIP
	} else {
		for _, addr := range addrs {
			_, dockerNetwork, err := net.ParseCIDR(addr)
			if err != nil {
				return err
			}
			if err := CheckRouteOverlaps(dockerNetwork); err == nil {
				ifaceAddr = addr
				break
			} else {
				log.Printf("%s %s", addr, err)
			}
		}
	}

	if ifaceAddr == "" {
		return fmt.Errorf("Could not find a free IP address range for interface '%s'. Please configure its address manually and run 'docker -b %s'", ovsBridgeName, ovsBridgeName)
	}
	fmt.Printf("Creating bridge %s with network %s", ovsBridgeName, ifaceAddr)

	if err := createBridgeIface(ovsBridgeName); err != nil {
		return err
	}
	iface, err := net.InterfaceByName(ovsBridgeName)
	if err != nil {
		return err
	}

	ipAddr, ipNet, err := net.ParseCIDR(ifaceAddr)
	if err != nil {
		return err
	}

	if netlink.NetworkLinkAddIp(iface, ipAddr, ipNet); err != nil {
		return fmt.Errorf("Unable to add private network: %s", err)
	}
	if err := netlink.NetworkLinkUp(iface); err != nil {
		return fmt.Errorf("Unable to start network bridge: %s", err)
	}
	return nil
}

func createBridgeIface(name string) error {
	if ovs == nil {
		return errors.New("OVS not connected")
	}
	// TODO : Error handling for CreateOVSBridge.
	CreateOVSBridge(ovs, name)
	// TODO : Lame. Remove the sleep. This is required now to keep netlink happy
	// in the next step to find the created interface.
	time.Sleep(time.Second * 1)
	return nil
}

func AddPeer(peerIp string) error {
	if ovs == nil {
		return errors.New("OVS not connected")
	}
	addVxlanPort(ovs, ovsBridgeName, "vxlan-"+peerIp, peerIp)
	return nil
}

func DeletePeer(peerIp string) error {
	if ovs == nil {
		return errors.New("OVS not connected")
	}
	deleteVxlanPort(ovs, ovsBridgeName, "vxlan-"+peerIp)
	return nil
}
