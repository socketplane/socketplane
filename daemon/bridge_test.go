package daemon

import (
	"bytes"
	"net"
	"testing"
)

func TestGetAvailableGwAddress(t *testing.T) {
	addr, err := GetAvailableGwAddress("")
	if err != nil {
		t.Fatal("Could not get Gateway address:", err)
	}
	if addr == "" {
		t.Fatal("Address is nil")
	}

	addr, err = GetAvailableGwAddress("1.2.3.1/24")
	if err != nil {
		t.Fatal("Could not get Gateway address:", err)
	}
	if addr == "" {
		t.Fatal("Address is nil")
	}
}

func TestGetAvailableGwAddressError(t *testing.T) {
	_, err := GetAvailableGwAddress("socketplane")
	if err == nil {
		t.Fatal("socketplane is not a valid IP Address")
	}
}

func TestGetAvailableSubnet(t *testing.T) {
	subnet, err := GetAvailableSubnet()
	if err != nil {
		t.Fatal("Error finding available subnet:", err)
	}
	if subnet == nil {
		t.Fatal("Subnet should not be nil")
	}
}

func TestCreateBridge(t *testing.T) {
	err := CreateBridge()
	if err != nil {
		t.Fatal("Error creating bridge:", err)
	}
}

func TestAddPeer(t *testing.T) {
	err := AddPeer("1.1.1.1")
	if err != nil {
		t.Fatal("Could not add peer:", err)
	}

	exists, err := portExists(ovs, "vxlan-1.1.1.1")
	if err != nil {
		t.Fatal("Error finding port:", err)
	}

	if !exists {
		t.Fatal("Port does not exist")
	}
}

func TestDeletePeer(t *testing.T) {
	err := DeletePeer("1.1.1.1")
	if err != nil {
		t.Fatal("Could not delete peer")
	}

	exists, err := portExists(ovs, "vxlan-1.1.1.1")
	if err != nil {
		t.Fatal("Error finding port:", err)
	}

	if exists {
		t.Fatal("Port has not been deleted")
	}
}

func TestGenerateRandomName(t *testing.T) {
	results := make(map[string]bool)
	for i := 0; i <= 100; i++ {
		result, err := GenerateRandomName("foo", 12)
		if err != nil {
			t.Fatal(err)
		}
		if results[result] != false {
			t.Fatal("generated name not unique")
		}
		results[result] = true
	}
}

func TestGenerateMacAddress(t *testing.T) {
	ip := net.ParseIP("1.1.1.1")
	mac := generateMacAddr(ip)
	if mac[0] != 0x02 {
		t.Fatal("first byte should be 0x02")
	}
	if mac[1] != 0x42 {
		t.Fatal("second byte should be 0x42")
	}
	if !bytes.Equal(mac[2:], ip.To4()) {
		t.Fatal("remaning bytes should be ipv4 address")
	}
}
