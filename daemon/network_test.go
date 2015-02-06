package daemon

import (
	"fmt"
	"net"
	"os"
	"testing"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
)

var subnetArray []*net.IPNet

func TestInit(t *testing.T) {
	if os.Getuid() != 0 {
		msg := "Skipped test because it requires root privileges."
		log.Printf(msg)
		t.Skip(msg)
	}
	_, ipNet1, _ := net.ParseCIDR("192.168.1.0/24")
	_, ipNet2, _ := net.ParseCIDR("192.168.2.0/24")
	_, ipNet3, _ := net.ParseCIDR("192.168.3.0/24")
	_, ipNet4, _ := net.ParseCIDR("192.168.4.0/24")
	_, ipNet5, _ := net.ParseCIDR("192.168.5.0/24")

	subnetArray = []*net.IPNet{ipNet1, ipNet2, ipNet3, ipNet4, ipNet5}
}

func TestGetEmptyNetworks(t *testing.T) {
	networks, _ := GetNetworks()
	if networks == nil {
		t.Error("GetNetworks must return an empty array when networks are not created ")
	}
}

func TestNetworkCreate(t *testing.T) {
	if os.Getuid() != 0 {
		msg := "Skipped test because it requires root privileges."
		log.Printf(msg)
		t.Skip(msg)
	}
	for i := 0; i < len(subnetArray); i++ {
		network, err := CreateNetwork(fmt.Sprintf("Network-%d", i+1), subnetArray[i])
		if err != nil {
			t.Error("Error Creating network ", err)
		}
		fmt.Println("Network Created Successfully", network)
	}
}

func TestGetNetworks(t *testing.T) {
	networks, _ := GetNetworks()
	if networks == nil || len(networks) < len(subnetArray) {
		t.Error("GetNetworks must return an empty array when networks are not created ")
	}
}

func TestGetNetwork(t *testing.T) {
	if os.Getuid() != 0 {
		msg := "Skipped test because it requires root privileges."
		log.Printf(msg)
		t.Skip(msg)
	}
	for i := 0; i < len(subnetArray); i++ {
		network, _ := GetNetwork(fmt.Sprintf("Network-%d", i+1))
		if network == nil {
			t.Error("Error GetNetwork")
		} else if network.Subnet != subnetArray[i].String() {
			t.Error("Network mismatch")
		}
		fmt.Println("GetNetwork : ", network)
	}
}

func TestCleanup(t *testing.T) {
	if os.Getuid() != 0 {
		msg := "Skipped test because it requires root privileges."
		log.Printf(msg)
		t.Skip(msg)
	}
	for i := 0; i < len(subnetArray); i++ {
		err := DeleteNetwork(fmt.Sprintf("Network-%d", i+1))
		if err != nil {
			t.Error("Error Deleting Network", err)
		}
	}
}
