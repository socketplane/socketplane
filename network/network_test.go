package network

import (
	"fmt"
	"net"
	"testing"

	"github.com/socketplane/socketplane/datastore"
)

var subnetArray []*net.IPNet

func TestInit(t *testing.T) {
	err := datastore.Init("eth1", true)
	if err != nil {
		t.Error("Error starting Consul ", err)
	}
	_, ipNet1, _ := net.ParseCIDR("192.168.1.0/24")
	_, ipNet2, _ := net.ParseCIDR("192.168.2.0/24")
	_, ipNet3, _ := net.ParseCIDR("192.168.3.0/24")
	_, ipNet4, _ := net.ParseCIDR("192.168.4.0/24")
	_, ipNet5, _ := net.ParseCIDR("192.168.5.0/24")

	subnetArray = []*net.IPNet{ipNet1, ipNet2, ipNet3, ipNet4, ipNet5}
}

func TestNetworkCreate(t *testing.T) {
	for i := 0; i < len(subnetArray); i++ {
		network, err := CreateNetwork(fmt.Sprintf("Network-%d", i+1), subnetArray[i])
		if err != nil {
			t.Error("Error Creating network ", err)
		}
		fmt.Println("Network Created Successfully", network)
	}
}

func TestGetNetwork(t *testing.T) {
	for i := 0; i < 5; i++ {
		network := GetNetwork(fmt.Sprintf("Network-%d", i+1))
		if network == nil {
			t.Error("Error GetNetwork")
		} else if network.Subnet.String() != subnetArray[i].String() {
			t.Error("Network mismatch")
		}
		fmt.Println("GetNetwork : ", network)
	}
}

func TestCleanup(t *testing.T) {
	for i := 0; i < 5; i++ {
		err := DeleteNetwork(fmt.Sprintf("Network-%d", i+1))
		if err != nil {
			t.Error("Error Deleting Network", err)
		}
	}
	datastore.Leave()
}
