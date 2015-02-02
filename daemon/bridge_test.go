package daemon

import "testing"

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
