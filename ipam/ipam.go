package ipam

import (
	"errors"
	"log"
	"net"

	"github.com/socketplane/ecc"
)

// Naive, Quick and Dirty IPv4 IPAM solution using Consul Distributed KV store
// Key = subnet, Value = next available ip-address numerically
// Yes. Naive.

// TODO : Implement a more robust default IPAM

const dataDir = "/tmp/socketplane"
const dataStore = "ipam"

func Init(bindInterface string, bootstrap bool) error {
	return ecc.Start(true, bootstrap, bindInterface, dataDir)
}

func GetAnAddress(subnet string) (string, error) {
	address, _, err := net.ParseCIDR(subnet)
	address = address.To4()
	if err != nil || address == nil {
		log.Printf("%v is not an IPv4 address\n", address)
		return "", errors.New(subnet + "is not an IPv4 address")
	}
	addrVal, _, ok := ecc.Get(dataStore, subnet)
	currVal := addrVal
	if !ok {
		address[3] = 1
		addrVal = address.String()
	}
	address = net.ParseIP(string(addrVal[:]))
	if err != nil || address.To4() == nil {
		log.Printf("%s is not an IPv4 address : %v\n", address.String(), err)
		return "", errors.New("Not a valid Ipv4 address " + address.String())
	}
	address = address.To4()
	address[3] += 1
	addrVal = address.String()

	eccerr := ecc.Put(dataStore, subnet, addrVal, currVal)
	if eccerr != ecc.OK {
		return GetAnAddress(subnet)
	}
	address[3] -= 1
	return address.String(), nil
}

func Join(address string) error {
	return ecc.Join(address)
}

func Leave() error {
	return ecc.Leave()
}
