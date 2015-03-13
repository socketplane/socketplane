package daemon

import (
	"encoding/json"
	"errors"
	"net"
	"time"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
)

const networkStore = "network"
const vlanStore = "vlan"
const DefaultNetworkName = "default"

const vlanCount = 4096

type Network struct {
	ID      string `json:"id"`
	Subnet  string `json:"subnet"`
	Gateway string `json:"gateway"`
	Vlan    uint   `json:"vlan"`
}

func GetNetworks() ([]Network, error) {
	netByteArray, _, ok := ecc.GetAll(networkStore)
	networks := make([]Network, 0)
	if ok {
		for _, byteArray := range netByteArray {
			network := Network{}
			err := json.Unmarshal(byteArray, &network)
			if err != nil {
				return nil, err
			}
			networks = append(networks, network)
		}
	}
	return networks, nil
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
		log.Debugf("Network '%s' found", id)
		return network, nil
	}

	vlan, err := allocateVlan()
	if err != nil {
		log.Debugf("Unable to allocate VLAN for Network '%s'. Error: %v", id, err.Error())
		return nil, err
	}

	var gateway net.IP

	addr, err := GetIfaceAddr(id)
	if err != nil {
		log.Debugf("Interface with name %s does not exist. Creating it.", id)
		if ovs == nil {
			return nil, errors.New("OVS not connected")
		}
		// Interface does not exist, use the generated subnet
		gateway = IPAMRequest(*subnet)
		network = &Network{id, subnet.String(), gateway.String(), vlan}
		if err = AddInternalPort(ovs, defaultBridgeName, network.ID, vlan); err != nil {
			return network, err
		}
		// TODO : Lame. Remove the sleep. This is required now to keep netlink happy
		// in the next step to find the created interface.
		time.Sleep(time.Second * 1)

		gatewayNet := &net.IPNet{gateway, subnet.Mask}

		log.Debugf("Setting address %s on %s", gatewayNet.String(), network.ID)

		if err = SetMtu(network.ID, mtu); err != nil {
			return network, err
		}
		if err = SetInterfaceIp(network.ID, gatewayNet.String()); err != nil {
			return network, err
		}
		if err = InterfaceUp(network.ID); err != nil {
			return network, err
		}
	} else {
		log.Debugf("Interface with name %s already exists", id)
		ifaceAddr := addr.String()
		gateway, subnet, err = net.ParseCIDR(ifaceAddr)
		if err != nil {
			return nil, err
		}
		network = &Network{id, subnet.String(), gateway.String(), vlan}
	}

	data, err := json.Marshal(network)
	if err != nil {
		return nil, err
	}

	eccerr := ecc.Put(networkStore, id, data, nil)
	if eccerr == ecc.OUTDATED {
		releaseVlan(vlan)
		IPAMRelease(gateway, *subnet)
		return CreateNetwork(id, subnet)
	}

	if err = setupIPTables(network.ID, network.Subnet); err != nil {
		return network, err
	}

	return network, nil
}

func DeleteNetwork(id string) error {
	network, err := GetNetwork(id)
	if err != nil {
		return err
	}
	if ovs == nil {
		return errors.New("OVS not connected")
	}
	eccerror := ecc.Delete(networkStore, id)
	if eccerror != ecc.OK {
		return errors.New("Error deleting network")
	}
	releaseVlan(network.Vlan)
	deletePort(ovs, defaultBridgeName, id)
	return nil
}

func CreateDefaultNetwork() (*Network, error) {
	subnet, err := GetAvailableSubnet()
	if err != nil {
		return &Network{}, err
	}
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
