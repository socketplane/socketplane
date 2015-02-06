package daemon

import (
	"bytes"
	"net"
	"runtime"
	"testing"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/vishvananda/netlink"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/vishvananda/netns"
)

const testIface = "foo0"

type tearDown func()

// Each test that calls setUp runs in it's own netns
func setUp(t *testing.T) tearDown {
	// lock thread since the namespace is thread local
	runtime.LockOSThread()
	var err error
	ns, err := netns.New()
	if err != nil {
		t.Fatal("Failed to create newns", ns)
	}

	netlink.LinkAdd(&netlink.Dummy{netlink.LinkAttrs{Name: testIface}})

	return func() {
		ns.Close()
		runtime.UnlockOSThread()
	}
}

func TestCheckRouteOverlaps(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	_, bad, _ := net.ParseCIDR("10.254.44.1/24")
	_, good, _ := net.ParseCIDR("1.1.1.1/24")

	var err error

	if err = SetInterfaceIp(testIface, "10.254.44.1/24"); err != nil {
		t.Fatal(err)
	}
	if err = InterfaceUp(testIface); err != nil {
		t.Fatal(err)
	}

	err = CheckRouteOverlaps(bad)
	if err != ErrNetworkOverlaps {
		t.Fatal("network should be overlapping")
	}

	err = CheckRouteOverlaps(good)
	if err != nil {
		t.Fatal("There should be no errors here")
	}
}

func TestNetworkOverlaps(t *testing.T) {
	_, netA, _ := net.ParseCIDR("10.1.0.0/22")
	_, netB, _ := net.ParseCIDR("10.1.1.0/24")
	_, netC, _ := net.ParseCIDR("10.10.10.0/24")

	if !NetworkOverlaps(netA, netB) {
		t.Fatal("netA and netB overlap")
	}
	if NetworkOverlaps(netA, netC) {
		t.Fatal("netB and netC do not overlap")
	}
}

func TestNetworkRange(t *testing.T) {
	_, netA, _ := net.ParseCIDR("10.1.0.0/22")
	expectedA := net.ParseIP("10.1.0.0")
	expectedB := net.ParseIP("10.1.3.255")
	addrA, addrB := NetworkRange(netA)

	if bytes.Equal(addrA, expectedA) {
		t.Fatalf("got: %v, expected, %v\n", addrA, expectedA)
	}
	if bytes.Equal(addrB, expectedB) {
		t.Fatalf("got: %v, expected, %v\n", addrB, expectedB)
	}

}

func TestNetworkSize(t *testing.T) {
	mask := net.IPv4Mask(255, 255, 255, 0)
	result := NetworkSize(mask)
	expected := int32(256)
	if result != expected {
		t.Fatalf("got %v, expected %v\n", result, expected)
	}
}

func TestGetDefaultRouteIface(t *testing.T) {
	result, err := GetDefaultRouteIface()
	if err != nil {
		t.Fatal(err)
	}
	if result <= 0 {
		t.Fatalf("link index should be > 0")
	}
}

func TestInterfaceUpDown(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	err := InterfaceUp(testIface)
	if err != nil {
		t.Fatal(err)
	}

	err = InterfaceDown(testIface)
	if err != nil {
		t.Fatal(err)
	}
}

func TestChangeInterfaceName(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	netlink.LinkAdd(&netlink.Dummy{netlink.LinkAttrs{Name: "foo1"}})
	err := ChangeInterfaceName("foo1", "eth42")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetInterfaceInNamespacePid(t *testing.T) {
	t.Skip("write the test")

}

func TestSetInterfaceInNamespaceFd(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	newns, err := netns.New()
	if err != nil {
		t.Fatal("Failed to create newns")
	}
	defer newns.Close()

	netlink.LinkAdd(&netlink.Dummy{netlink.LinkAttrs{Name: "foo2"}})

	err = SetInterfaceInNamespaceFd("foo2", uintptr(newns))
	if err != nil {
		t.Fatal(err)
	}

	_, err = netlink.LinkByName("bar")
	if err == nil {
		t.Fatal("Link foo2 is still in newns")
	}
}

func TestSetInterfaceMac(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	err := SetInterfaceMac("doesnotexist0", "notamacaddress")
	if err == nil {
		t.Fatal("this interface should not exist")
	}
	err = SetInterfaceMac(testIface, "notamacaddress")
	if err == nil {
		t.Fatal("not a valid mac address")
	}
	err = SetInterfaceMac(testIface, "00:01:22:33:44:ab")
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetGetInterfaceIp(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	addr, err := GetIfaceAddr("doesnotexist0")
	if err == nil {
		t.Fatalf("this interface should not exist")
	}
	if addr != nil {
		t.Fatal("no address should be returned")
	}

	addr, err = GetIfaceAddr(testIface)
	if err == nil {
		t.Fatalf("this interface has no address")
	}
	if addr != nil {
		t.Fatalf("no address should be returned")
	}

	err = SetInterfaceIp("doesnotexist0", "notanipaddress")
	if err == nil {
		t.Fatal("this interface should not exist")
	}
	err = SetInterfaceIp(testIface, "notanipaddress")
	if err == nil {
		t.Fatal("not a valid ip address")
	}
	err = SetInterfaceIp(testIface, "2.2.2.1/24")
	if err != nil {
		t.Fatal(err)
	}

	addr, err = GetIfaceAddr(testIface)
	if err != nil {
		t.Fatal(err)
	}
	if addr == nil {
		t.Fatal("address is nil")
	}

	err = SetInterfaceIp(testIface, "172.88.21.1/24")
	if err != nil {
		t.Fatal(err)
	}

	addr, err = GetIfaceAddr(testIface)
	if err != nil {
		t.Fatal(err)
	}
	if addr == nil {
		t.Fatal("address is nil")
	}

}

func TestSetMtu(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	err := SetMtu("doesnotexist0", 1400)
	if err == nil {
		t.Fatal("this interface should not exist")
	}
	err = SetMtu(testIface, 1400)
	if err != nil {
		t.Fatal(err)
	}
}

func TestSetDefaultGateway(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	err := SetDefaultGateway("badip", "doesnotexist0")
	if err == nil {
		t.Fatalf("this interface should not exist")
	}
	err = SetDefaultGateway("badip", testIface)
	if err == nil {
		t.Fatalf("this ip address is inavlid")
	}

	err = SetInterfaceIp(testIface, "2.2.2.2/24")
	if err != nil {
		t.Fatal("could not set interface ip")
	}
	err = InterfaceUp(testIface)
	if err != nil {
		t.Fatal("could not bring interface up")
	}

	err = SetDefaultGateway("2.2.2.254", testIface)
	if err != nil {
		t.Fatal(err)
	}

}

func TestGetIfaceForRoute(t *testing.T) {
	teardown := setUp(t)
	defer teardown()

	i, err := GetIfaceForRoute("invalidip")
	if err == nil {
		t.Fatal("invalid ip address")
	}

	i, err = GetIfaceForRoute("2.2.2.2")
	if err == nil {
		t.Fatal("no route present for for 2.2.2.2")
	}

	if err = SetInterfaceIp(testIface, "2.2.2.2/24"); err != nil {
		t.Fatal("could not set interface ip")
	}

	if err = InterfaceUp(testIface); err != nil {
		t.Fatal("can't bring iface up")
	}

	i, err = GetIfaceForRoute("2.2.2.2")
	if err != nil {
		t.Fatal(err)
	}

	if i <= 0 {
		t.Fatal("ifindex should be > 0")
	}
}
