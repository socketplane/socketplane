package daemon

import (
	"encoding/binary"
	"errors"
	"fmt"

	"net"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/vishvananda/netlink"
)

var (
	ErrNoDefaultRoute                 = errors.New("no default route")
	ErrNetworkOverlapsWithNameservers = errors.New("requested network overlaps with nameserver")
	ErrNetworkOverlaps                = errors.New("requested network overlaps with existing network")
)

func CheckRouteOverlaps(toCheck *net.IPNet) error {
	networks, err := netlink.RouteList(nil, netlink.FAMILY_V4)
	if err != nil {
		return err
	}

	for _, network := range networks {
		if network.Dst != nil && NetworkOverlaps(toCheck, network.Dst) {
			return ErrNetworkOverlaps
		}
	}
	return nil
}

// Detects overlap between one IPNet and another
func NetworkOverlaps(netX *net.IPNet, netY *net.IPNet) bool {
	if firstIP, _ := NetworkRange(netX); netY.Contains(firstIP) {
		return true
	}
	if firstIP, _ := NetworkRange(netY); netX.Contains(firstIP) {
		return true
	}
	return false
}

// Calculates the first and last IP addresses in an IPNet
func NetworkRange(network *net.IPNet) (net.IP, net.IP) {
	var (
		netIP   = network.IP.To4()
		firstIP = netIP.Mask(network.Mask)
		lastIP  = net.IPv4(0, 0, 0, 0).To4()
	)

	for i := 0; i < len(lastIP); i++ {
		lastIP[i] = netIP[i] | ^network.Mask[i]
	}
	return firstIP, lastIP
}

// Given a netmask, calculates the number of available hosts
func NetworkSize(mask net.IPMask) int32 {
	m := net.IPv4Mask(0, 0, 0, 0)
	for i := 0; i < net.IPv4len; i++ {
		m[i] = ^mask[i]
	}
	return int32(binary.BigEndian.Uint32(m)) + 1
}

// Return the IPv4 address of a network interface
func GetIfaceAddr(name string) (*net.IPNet, error) {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return nil, err
	}

	addrs, err := netlink.AddrList(iface, netlink.FAMILY_V4)
	if err != nil {
		return nil, err
	}

	if len(addrs) == 0 {
		return nil, fmt.Errorf("Interface %v has no IP addresses", name)
	}

	if len(addrs) > 1 {
		log.Info("Interface %v has more than 1 IPv4 address. Defaulting to using %v\n", name, addrs[0].IP)
	}

	return addrs[0].IPNet, nil
}

func GetDefaultRouteIface() (int, error) {
	defaultRt := net.ParseIP("0.0.0.0")
	rs, err := netlink.RouteGet(defaultRt)
	if err != nil {
		return -1, fmt.Errorf("unable to get default route: %v", err)
	}
	if len(rs) > 0 {
		return rs[0].LinkIndex, nil
	}
	return -1, ErrNoDefaultRoute
}

func InterfaceUp(name string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetUp(iface)
}

func InterfaceDown(name string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetDown(iface)
}

func ChangeInterfaceName(old, newName string) error {
	iface, err := netlink.LinkByName(old)
	if err != nil {
		return err
	}
	return netlink.LinkSetName(iface, newName)
}

func SetInterfaceInNamespacePid(name string, nsPid int) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetNsPid(iface, nsPid)
}

func SetInterfaceInNamespaceFd(name string, fd uintptr) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetNsFd(iface, int(fd))
}

func SetDefaultGateway(ip, ifaceName string) error {
	iface, err := netlink.LinkByName(ifaceName)
	if err != nil {
		return err
	}
	gw := net.ParseIP(ip)
	if gw == nil {
		return errors.New("Invalid gateway address")
	}

	_, dst, err := net.ParseCIDR("0.0.0.0/0")
	if err != nil {
		return err
	}
	defaultRoute := &netlink.Route{
		LinkIndex: iface.Attrs().Index,
		Dst:       dst,
		Gw:        gw,
	}
	return netlink.RouteAdd(defaultRoute)
}

func SetInterfaceMac(name string, macaddr string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	hwaddr, err := net.ParseMAC(macaddr)
	if err != nil {
		return err
	}
	return netlink.LinkSetHardwareAddr(iface, hwaddr)
}

func SetInterfaceIp(name string, rawIp string) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}

	ipNet, err := netlink.ParseIPNet(rawIp)
	if err != nil {
		return err
	}
	addr := &netlink.Addr{ipNet, ""}
	return netlink.AddrAdd(iface, addr)
}

func SetMtu(name string, mtu int) error {
	iface, err := netlink.LinkByName(name)
	if err != nil {
		return err
	}
	return netlink.LinkSetMTU(iface, mtu)
}

func GetIfaceForRoute(address string) (string, error) {
	addr := net.ParseIP(address)
	if addr == nil {
		return "", errors.New("invalid address")
	}
	routes, err := netlink.RouteGet(addr)
	if err != nil {
		return "", err
	}
	if len(routes) <= 0 {
		return "", errors.New("no route to destination")
	}
	link, err := netlink.LinkByIndex(routes[0].LinkIndex)
	if err != nil {
		return "", err
	}
	return link.Attrs().Name, nil
}
