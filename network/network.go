package network

import (
	"errors"
	"log"
	"net"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
)

const dataStore = "network"
const DefaultNetworkName = "default"

const vlanCount = 4096

type Network struct {
	ID     string     `json:"id"`
	Subnet *net.IPNet `json:"subnet"`
	Vlan   uint       `json:"vlan"`
}

var vlanArray []byte
var networkMap map[string]*Network

func init() {
	vlanArray = make([]byte, vlanCount/8)
	networkMap = make(map[string]*Network)
}

func CreateNetwork(id string, subnet *net.IPNet) (*Network, error) {
	storeSubnet, _, ok := ecc.Get(dataStore, id)
	if ok {
		if subnet.String() != string(storeSubnet[:]) {
			return nil, errors.New("Network mismatch")
		}
		return networkFromLocalCache(id, subnet), nil
	}
	eccerr := ecc.Put(dataStore, id, []byte(subnet.String()), nil)
	if eccerr == ecc.OUTDATED {
		return CreateNetwork(id, subnet)
	}
	return networkFromLocalCache(id, subnet), nil
}

func DeleteNetwork(id string) error {
	eccerror := ecc.Delete(dataStore, id)
	if eccerror == ecc.OK {
		removeNetworkFromLocalCache(id)
		return nil
	}
	return errors.New("Error deleting network")
}

func GetNetwork(id string) *Network {
	return networkMap[id]
}

func CreateDefaultNetwork(subnet *net.IPNet) (*Network, error) {
	return CreateNetwork(DefaultNetworkName, subnet)
}

func GetDefaultNetwork() *Network {
	return GetNetwork(DefaultNetworkName)
}

func networkFromLocalCache(id string, subnet *net.IPNet) *Network {
	if network, ok := networkMap[id]; ok {
		log.Println("Network from local cache", *network)
		return network
	}
	vlan := testAndSetBit(vlanArray)
	if vlan >= vlanCount {
		log.Println("No more vlan for ", id)
		return nil
	}
	network := Network{id, subnet, vlan}
	networkMap[id] = &network
	log.Println("Network created in local cache", network)
	return networkMap[id]
}

func removeNetworkFromLocalCache(id string) {
	if _, ok := networkMap[id]; !ok {
		return
	}
	clearBit(vlanArray, networkMap[id].Vlan)
	delete(networkMap, id)
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
