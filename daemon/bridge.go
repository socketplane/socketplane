package daemon

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/libovsdb"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/vishvananda/netns"
	"github.com/socketplane/socketplane/ipam"
)

// Gateway addresses are from docker/daemon/networkdriver/bridge/driver.go to reflect similar behaviour
// between this temporary wrapper solution to the native Network integration

var gatewayAddrs = []string{
	// Here we don't follow the convention of using the 1st IP of the range for the gateway.
	// This is to use the same gateway IPs as the /24 ranges, which predate the /16 ranges.
	// In theory this shouldn't matter - in practice there's bound to be a few scripts relying
	// on the internal addressing or other stupid things like that.
	// They shouldn't, but hey, let's not break them unless we really have to.
	"10.1.42.1/16",
	"10.42.42.1/16",
	"172.16.42.1/24",
	"172.16.43.1/24",
	"172.16.44.1/24",
	"10.0.42.1/24",
	"10.0.43.1/24",
	"172.17.42.1/16", // Don't use 172.16.0.0/16, it conflicts with EC2 DNS 172.16.0.23
	"10.0.42.1/16",   // Don't even try using the entire /8, that's too intrusive
	"192.168.42.1/24",
	"192.168.43.1/24",
	"192.168.44.1/24",
}

// Setting a mtu value to 1440 temporarily to resolve #71
const mtu = 1440
const defaultBridgeName = "docker0-ovs"

type Bridge struct {
	Name string
	//	IP     net.IP
	//	Subnet *net.IPNet
}

var OvsBridge Bridge = Bridge{Name: defaultBridgeName}

var ovs *libovsdb.OvsdbClient
var ContextCache map[string]string

func OvsInit() {
	var err error
	ovs, err = ovs_connect()
	if err != nil {
		log.Error("Error connecting OVS ", err)
	} else {
		ovs.Register(notifier{})
	}
	ContextCache = make(map[string]string)
	populateContextCache()
}

func GetAvailableGwAddress(bridgeIP string) (gwaddr string, err error) {
	if len(bridgeIP) != 0 {
		_, _, err = net.ParseCIDR(bridgeIP)
		if err != nil {
			return
		}
		gwaddr = bridgeIP
	} else {
		for _, addr := range gatewayAddrs {
			_, dockerNetwork, err := net.ParseCIDR(addr)
			if err != nil {
				return "", err
			}
			if err = CheckRouteOverlaps(dockerNetwork); err != nil {
				continue
			}
			gwaddr = addr
			break
		}
	}
	if gwaddr == "" {
		return "", errors.New("No available gateway addresses")
	}
	return gwaddr, nil
}

func GetAvailableSubnet() (subnet *net.IPNet, err error) {
	for _, addr := range gatewayAddrs {
		_, dockerNetwork, err := net.ParseCIDR(addr)
		if err != nil {
			return &net.IPNet{}, err
		}
		if err = CheckRouteOverlaps(dockerNetwork); err == nil {
			return dockerNetwork, nil
		}
	}

	return &net.IPNet{}, errors.New("No available GW address")
}

func CreateBridge() error {
	if ovs == nil {
		return errors.New("OVS not connected")
	}
	// If the bridge has been created, a port with the same name should exist
	exists, err := portExists(ovs, OvsBridge.Name)
	if err != nil {
		return err
	}
	if !exists {
		if err := createBridgeIface(OvsBridge.Name); err != nil {
			return err
		}
		exists, err = portExists(ovs, OvsBridge.Name)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("Error creating Bridge")
		}
	}
	return nil
}

func createBridgeIface(name string) error {
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
	deletePort(ovs, OvsBridge.Name, "vxlan-"+peerIp)
	return nil
}

type OvsConnection struct {
	Name    string `json:"name"`
	Ip      string `json:"ip"`
	Subnet  string `json:"subnet"`
	Mac     string `json:"mac"`
	Gateway string `json:"gateway"`
}

const (
	ConnectionAdd    = iota
	ConnectionUpdate = iota
	ConnectionDelete = iota
)

type ConnectionContext struct {
	Action     int
	Connection *Connection
	Result     chan *Connection
}

func ConnectionRPCHandler(d *Daemon) {
	for {
		c := <-d.cC

		switch c.Action {
		case ConnectionAdd:
			pid, _ := strconv.Atoi(c.Connection.ContainerPID)
			connDetails, _ := AddConnection(pid, c.Connection.Network)
			c.Connection.OvsPortID = connDetails.Name
			c.Connection.ConnectionDetails = connDetails
			d.Connections[c.Connection.ContainerID] = c.Connection
			// ToDo: We should deprecate this when we have a proper CLI
			c.Result <- c.Connection
		case ConnectionUpdate:
			// noop
		case ConnectionDelete:
			DeleteConnection(c.Connection.ConnectionDetails)
			delete(d.Connections, c.Connection.ContainerID)
			c.Result <- c.Connection
		}
	}
}

func AddConnection(nspid int, networkName string) (ovsConnection OvsConnection, err error) {
	var (
		bridge = OvsBridge.Name
		prefix = "ovs"
	)
	ovsConnection = OvsConnection{}
	err = nil

	if bridge == "" {
		err = fmt.Errorf("bridge is not available")
		return
	}

	if networkName == "" {
		networkName = DefaultNetworkName
	}

	bridgeNetwork, err := GetNetwork(networkName)
	if err != nil {
		return ovsConnection, err
	}

	portName, err := createOvsInternalPort(prefix, bridge, bridgeNetwork.Vlan)
	if err != nil {
		return
	}
	// Add a dummy sleep to make sure the interface is seen by the subsequent calls.
	time.Sleep(time.Second * 1)

	_, subnet, _ := net.ParseCIDR(bridgeNetwork.Subnet)

	ip := ipam.Request(*subnet)
	mac := generateMacAddr(ip).String()

	subnetString := subnet.String()
	subnetPrefix := subnetString[len(subnetString)-3 : len(subnetString)]

	ovsConnection = OvsConnection{portName, ip.String(), subnetPrefix, mac, bridgeNetwork.Gateway}

	if err = SetMtu(portName, mtu); err != nil {
		return
	}
	if err = InterfaceUp(portName); err != nil {
		return
	}

	if err = os.Symlink(filepath.Join(os.Getenv("PROCFS"), strconv.Itoa(nspid), "ns/net"),
		filepath.Join("/var/run/netns", strconv.Itoa(nspid))); err != nil {
		return
	}

	// Lock the OS Thread so we don't accidentally switch namespaces
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	origns, err := netns.Get()
	if err != nil {
		return
	}
	defer origns.Close()

	targetns, err := netns.GetFromName(strconv.Itoa(nspid))
	if err != nil {
		return
	}
	defer targetns.Close()

	if err = SetInterfaceInNamespaceFd(portName, uintptr(int(targetns))); err != nil {
		return
	}

	if err = netns.Set(targetns); err != nil {
		return
	}
	defer netns.Set(origns)

	if err = InterfaceDown(portName); err != nil {
		return
	}

	/* TODO : Find a way to change the interface name to defaultDevice (eth0).
	   Currently using the Randomly created OVS port as is.
	   refer to veth.go where one end of the veth pair is renamed to eth0
	*/
	if err = ChangeInterfaceName(portName, portName); err != nil {
		return
	}

	if err = SetInterfaceIp(portName, ip.String()+subnetPrefix); err != nil {
		return
	}

	if err = SetInterfaceMac(portName, generateMacAddr(ip).String()); err != nil {
		return
	}

	if err = InterfaceUp(portName); err != nil {
		return
	}

	if err = SetDefaultGateway(bridgeNetwork.Gateway, portName); err != nil {
		return
	}

	return ovsConnection, nil
}

func UpdateConnectionContext(ovsPort string, key string, context string) error {
	return UpdatePortContext(ovs, ovsPort, key, context)
}

func populateContextCache() {
	if ovs == nil {
		return
	}
	tableCache := GetTableCache("Interface")
	for _, row := range tableCache {
		config, ok := row.Fields["other_config"]
		ovsMap := config.(libovsdb.OvsMap)
		other_config := map[interface{}]interface{}(ovsMap.GoMap)
		if ok {
			container_id, ok := other_config[CONTEXT_KEY]
			if ok {
				ContextCache[container_id.(string)] = other_config[CONTEXT_VALUE].(string)
			}
		}
	}
}

func DeleteConnection(connection OvsConnection) error {
	if ovs == nil {
		return errors.New("OVS not connected")
	}
	deletePort(ovs, OvsBridge.Name, connection.Name)
	ip := net.ParseIP(connection.Ip)
	_, subnet, _ := net.ParseCIDR(connection.Ip + connection.Subnet)
	ipam.Release(ip, *subnet)
	return nil
}

// createOvsInternalPort will generate a random name for the
// the port and ensure that it has been created
func createOvsInternalPort(prefix string, bridge string, tag uint) (port string, err error) {
	if port, err = GenerateRandomName(prefix, 7); err != nil {
		return
	}

	if ovs == nil {
		err = errors.New("OVS not connected")
		return
	}

	AddInternalPort(ovs, bridge, port, tag)
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

func setupIPTables(bridgeName string, bridgeIP string) error {
	/*
		# Enable IP Masquerade on all ifaces that are not docker-ovs0
		iptables -t nat -A POSTROUTING -s 10.1.42.1/16 ! -o %bridgeName -j MASQUERADE

		# Enable outgoing connections on all interfaces
		iptables -A FORWARD -i %bridgeName ! -o %bridgeName -j ACCEPT

		# Enable incoming connections for established sessions
		iptables -A FORWARD -o %bridgeName -m conntrack --ctstate RELATED,ESTABLISHED -j ACCEPT
	*/

	log.Debug("Setting up iptables")
	natArgs := []string{"-t", "nat", "-A", "POSTROUTING", "-s", bridgeIP, "!", "-o", bridgeName, "-j", "MASQUERADE"}
	output, err := installRule(natArgs...)
	if err != nil {
		log.Debugf("Unable to enable network bridge NAT: %s", err)
		return fmt.Errorf("Unable to enable network bridge NAT: %s", err)
	}
	if len(output) != 0 {
		log.Debugf("Error enabling network bridge NAT: %s", err)
		return fmt.Errorf("Error enabling network bridge NAT: %s", output)
	}

	outboundArgs := []string{"-A", "FORWARD", "-i", bridgeName, "!", "-o", bridgeName, "-j", "ACCEPT"}
	output, err = installRule(outboundArgs...)
	if err != nil {
		log.Debugf("Unable to enable network outbound forwarding: %s", err)
		return fmt.Errorf("Unable to enable network outbound forwarding: %s", err)
	}
	if len(output) != 0 {
		log.Debugf("Error enabling network outbound forwarding: %s", output)
		return fmt.Errorf("Error enabling network outbound forwarding: %s", output)
	}

	inboundArgs := []string{"-A", "FORWARD", "-o", bridgeName, "-m", "conntrack", "--ctstate", "RELATED,ESTABLISHED", "-j", "ACCEPT"}
	output, err = installRule(inboundArgs...)
	if err != nil {
		log.Debugf("Unable to enable network inbound forwarding: %s", err)
		return fmt.Errorf("Unable to enable network inbound forwarding: %s", err)
	}
	if len(output) != 0 {
		log.Debugf("Error enabling network inbound forwarding: %s")
		return fmt.Errorf("Error enabling network inbound forwarding: %s", output)
	}
	return nil
}

func installRule(args ...string) ([]byte, error) {
	path, err := exec.LookPath("iptables")
	if err != nil {
		return nil, errors.New("iptables not found")
	}

	output, err := exec.Command(path, args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("iptables failed: iptables %v: %s (%s)", strings.Join(args, " "), output, err)
	}

	return output, err
}

type notifier struct {
}

func (n notifier) Disconnected(ovsClient *libovsdb.OvsdbClient) {
	log.Error("OVS Disconnected. Retrying...")
	ovs = nil
	go OvsInit()
}

func (n notifier) Update(context interface{}, tableUpdates libovsdb.TableUpdates) {
}
func (n notifier) Locked([]interface{}) {
}
func (n notifier) Stolen([]interface{}) {
}
func (n notifier) Echo([]interface{}) {
}
