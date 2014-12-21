package network

import (
	"encoding/json"
	"errors"
	"net"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
)

const networkStore = "network"
const vlanStore = "vlan"
const DefaultNetworkName = "default"

const vlanCount = 4096

type Network struct {
	ID     string `json:"id"`
	Subnet string `json:"subnet"`
	Vlan   uint   `json:"vlan"`
}

func GetNetwork(id string) (*Network, error) {
	netByteArray, _, ok := ecc.Get(networkStore, id)
	if ok {
		network := &Network{}
		err := json.Unmarshal(netByteArray, network)
		if err != nil {
			return nil, err
		}
		return network, nil
	}
	return nil, errors.New("Network unavailable")
}

func CreateNetwork(id string, subnet *net.IPNet) (*Network, error) {
	network, err := GetNetwork(id)
	if err == nil {
		return network, nil
	}
	vlan, err := allocateVlan()
	if err != nil {
		return nil, err
	}
	network = &Network{id, subnet.String(), vlan}
	data, err := json.Marshal(network)
	if err != nil {
		return nil, err
	}
	eccerr := ecc.Put(networkStore, id, data, nil)
	if eccerr == ecc.OUTDATED {
		releaseVlan(vlan)
		return CreateNetwork(id, subnet)
	}
	return network, nil
}

func DeleteNetwork(id string) error {
	network, err := GetNetwork(id)
	if err != nil {
		return err
	}
	eccerror := ecc.Delete(networkStore, id)
	if eccerror == ecc.OK {
		releaseVlan(network.Vlan)
		return nil
	}
	return errors.New("Error deleting network")
}

func CreateDefaultNetwork(subnet *net.IPNet) (*Network, error) {
	return CreateNetwork(DefaultNetworkName, subnet)
}

func GetDefaultNetwork() (*Network, error) {
	return GetNetwork(DefaultNetworkName)
}

func allocateVlan() (uint, error) {
	vlanArray, _, ok := ecc.Get(vlanStore, "vlan")
	currVal := make([]byte, vlanCount/8)
	copy(currVal, vlanArray)
	if !ok {
		vlanArray = make([]byte, vlanCount/8)
	}
	vlan := testAndSetBit(vlanArray)
	if vlan >= vlanCount {
		return vlanCount, errors.New("Vlan unavailable")
	}
	eccerr := ecc.Put(vlanStore, "vlan", vlanArray, currVal)
	if eccerr == ecc.OUTDATED {
		return allocateVlan()
	}
	return vlan, nil
}

func releaseVlan(vlan uint) {
	vlanArray, _, ok := ecc.Get(vlanStore, "vlan")
	currVal := make([]byte, vlanCount/8)
	copy(currVal, vlanArray)
	if !ok {
		vlanArray = make([]byte, vlanCount/8)
	}
	clearBit(vlanArray, vlan-1)
	eccerr := ecc.Put(vlanStore, "vlan", vlanArray, currVal)
	if eccerr == ecc.OUTDATED {
		releaseVlan(vlan)
	}
}
func setBit(a []byte, k uint) {
	a[k/8] |= 1 << (k % 8)
}

func clearBit(a []byte, k uint) {
	a[k/8] &= ^(1 << (k % 8))
}

func testBit(a []byte, k uint) bool {
	return ((a[k/8] & (1 << (k % 8))) != 0)
}

func testAndSetBit(a []byte) uint {
	var i uint
	for i = uint(0); i < uint(len(a)*8); i++ {
		if !testBit(a, i) {
			setBit(a, i)
			return i + 1
		}
	}
	return i
}
