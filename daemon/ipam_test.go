package daemon

import (
	"net"
	"testing"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
)

func TestIPAMInit(t *testing.T) {
	err := InitDatastore("", true)
	if err != nil {
		t.Log("Error starting Consul . Not failing ", err)
	}
}

func TestIpRelease(t *testing.T) {
	count := 25
	_, ipNet, _ := net.ParseCIDR("192.170.0.0/24")
	for i := 1; i < count; i++ {
		address := IPAMRequest(*ipNet)
		address = address.To4()
		if i%256 != int(address[3]) || i/256 != int(address[2]) {
			t.Error(address.String())
		}
	}

	IPAMRelease(net.ParseIP("192.170.0.1"), *ipNet)
	IPAMRelease(net.ParseIP("192.170.0.11"), *ipNet)

	address := IPAMRequest(*ipNet).To4()
	if int(address[3]) != 1 {
		t.Error(address.String())
	}
	address = IPAMRequest(*ipNet).To4()
	if int(address[3]) != 11 {
		t.Error(address.String())
	}
}

func TestIpReleasePartialMask(t *testing.T) {
	count := 25
	_, ipNet, _ := net.ParseCIDR("192.170.32.0/20")
	for i := 1; i < count; i++ {
		address := IPAMRequest(*ipNet)
		address = address.To4()
		if i%256 != int(address[3]) || 32+i/256 != int(address[2]) {
			t.Error(address.String())
		}
	}

	IPAMRelease(net.ParseIP("192.170.32.3"), *ipNet)
	IPAMRelease(net.ParseIP("192.170.32.14"), *ipNet)

	address := IPAMRequest(*ipNet).To4()
	if int(address[3]) != 3 {
		t.Error(address.String())
	}
	address = IPAMRequest(*ipNet).To4()
	if int(address[3]) != 14 {
		t.Error(address.String())
	}
}

func TestGetIpFullMask(t *testing.T) {
	count := 300
	for i := 1; i < count; i++ {
		_, ipNet, _ := net.ParseCIDR("192.168.0.0/16")
		address := IPAMRequest(*ipNet)
		address = address.To4()
		if i%256 != int(address[3]) || i/256 != int(address[2]) {
			t.Error(address.String())
		}
	}
}

func TestGetIpPartialMask(t *testing.T) {
	count := 300
	for i := 1; i < count; i++ {
		_, ipNet, _ := net.ParseCIDR("192.169.32.0/20")
		address := IPAMRequest(*ipNet)
		address = address.To4()
		if i%256 != int(address[3]) || 32+i/256 != int(address[2]) {
			t.Error(address.String())
		}
	}
}

func TestIPAMCleanup(t *testing.T) {
	ecc.Delete(dataStore, "192.170.0.0/24")
	ecc.Delete(dataStore, "192.170.32.0/20")
	ecc.Delete(dataStore, "192.167.1.0/24")
	ecc.Delete(dataStore, "192.168.0.0/16")
	ecc.Delete(dataStore, "192.169.32.0/20")
	LeaveDatastore()
}
