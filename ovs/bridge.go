package ovs

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/docker/libcontainer/netlink"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/libovsdb"
	"github.com/socketplane/socketplane/ipam"
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

const mtu = 1514
const defaultBridgeName = "docker0-ovs"

type Bridge struct {
	Name   string
	IP     net.IP
	Subnet *net.IPNet
}

var OvsBridge Bridge = Bridge{Name: defaultBridgeName}

var ovs *libovsdb.OvsdbClient

func init() {
	var err error
	ovs, err = ovs_connect()
	if err != nil {
		log.Println("Error connecting OVS ", err)
	}
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
		return fmt.Errorf("Could not find a free IP address range for interface '%s'. Please configure its address manually and run 'docker -b %s'", OvsBridge.Name, OvsBridge.Name)
	}

	if err := createBridgeIface(OvsBridge.Name); err != nil {
		return err
	}
	iface, err := net.InterfaceByName(OvsBridge.Name)
	if err != nil {
		return err
	}

	ipAddr, ipNet, err := net.ParseCIDR(ifaceAddr)
	if err != nil {
		return err
	}

	OvsBridge.IP = ipAddr
	OvsBridge.Subnet = ipNet

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
	addVxlanPort(ovs, OvsBridge.Name, "vxlan-"+peerIp, peerIp)
	return nil
}

func DeletePeer(peerIp string) error {
	if ovs == nil {
		return errors.New("OVS not connected")
	}
	deleteVxlanPort(ovs, OvsBridge.Name, "vxlan-"+peerIp)
	return nil
}

func AddConnection(nspid int) (string, error) {
	var (
		bridge = OvsBridge.Name
		prefix = "ovs"
	)
	if bridge == "" {
		return "", fmt.Errorf("bridge is not available")
	}
	portName, err := createOvsInternalPort(prefix, bridge)
	if err != nil {
		return "", err
	}
	// Add a dummy sleep to make sure the interface is seen by the subsequent calls.
	time.Sleep(time.Second * 1)
	if err := SetMtu(portName, mtu); err != nil {
		return "", err
	}
	if err := InterfaceUp(portName); err != nil {
		return "", err
	}
	if err := SetInterfaceInNamespacePid(portName, nspid); err != nil {
		return "", err
	}

	if err := InterfaceDown(portName); err != nil {
		return "", fmt.Errorf("interface down %s %s", portName, err)
	}
	// TODO : Find a way to change the interface name to defaultDevice (eth0).
	// Currently using the Randomly created OVS port as is.
	// refer to veth.go where one end of the veth pair is renamed to eth0
	if err := ChangeInterfaceName(portName, portName); err != nil {
		return "", fmt.Errorf("change %s to %s %s", portName, portName, err)
	}

	ip := ipam.Request(*OvsBridge.Subnet)
	if err := SetInterfaceIp(portName, ip.String()); err != nil {
		return "", fmt.Errorf("set %s ip %s", portName, err)
	}
	if err := SetInterfaceMac(portName, generateMacAddr(ip).String()); err != nil {
		return "", fmt.Errorf("set %s mac %s", portName, err)
	}

	if err := InterfaceUp(portName); err != nil {
		return "", fmt.Errorf("%s up %s", portName, err)
	}
	if err := SetDefaultGateway(OvsBridge.IP.String(), portName); err != nil {
		return "", fmt.Errorf("set gateway to %s on device %s failed with %s", OvsBridge.IP, portName, err)
	}
	return portName, nil
}

// createOvsInternalPort will generate a random name for the
// the port and ensure that it has been created
func createOvsInternalPort(prefix string, bridge string) (port string, err error) {
	if port, err = GenerateRandomName(prefix, 7); err != nil {
		return
	}

	if ovs == nil {
		err = errors.New("OVS not connected")
		return
	}

	AddInternalPort(ovs, bridge, port)
	return
}

// GenerateRandomName returns a new name joined with a prefix.  This size
// specified is used to truncate the randomly generated value
func GenerateRandomName(prefix string, size int) (string, error) {
	id := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, id); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(id)[:size], nil
}

func generateMacAddr(ip net.IP) net.HardwareAddr {
	hw := make(net.HardwareAddr, 6)

	// The first byte of the MAC address has to comply with these rules:
	// 1. Unicast: Set the least-significant bit to 0.
	// 2. Address is locally administered: Set the second-least-significant bit (U/L) to 1.
	// 3. As "small" as possible: The veth address has to be "smaller" than the bridge address.
	hw[0] = 0x02

	// The first 24 bits of the MAC represent the Organizationally Unique Identifier (OUI).
	// Since this address is locally administered, we can do whatever we want as long as
	// it doesn't conflict with other addresses.
	hw[1] = 0x42

	// Insert the IP address into the last 32 bits of the MAC address.
	// This is a simple way to guarantee the address will be consistent and unique.
	copy(hw[2:], ip.To4())

	return hw
}
