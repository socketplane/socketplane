package ipam

import (
	"errors"
	"log"
	"net"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
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
	addrArray, _, ok := ecc.Get(dataStore, subnet)
	currVal := make([]byte, len(addrArray))
	copy(currVal, addrArray)
	if !ok {
		var err error
		address, _, err := net.ParseCIDR(subnet)
		address = address.To4()
		if err != nil || address == nil {
			log.Printf("%v is not an IPv4 address\n", address)
			return "", errors.New(subnet + "is not an IPv4 address")
		}
		address[3] = 1
		addrArray = []byte(address)
	}
	addrArray[3] += 1

	eccerr := ecc.Put(dataStore, subnet, addrArray, currVal)
	if eccerr != ecc.OK {
		return GetAnAddress(subnet)
	}
	addrArray[3] -= 1
	return net.IP(addrArray).String(), nil
}

func Join(address string) error {
	return ecc.Join(address)
}

func Leave() error {
	return ecc.Leave()
}
