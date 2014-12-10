package ipam

import (
	"log"
	"net"
	"testing"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
)

func TestInit(t *testing.T) {
	err := Init("eth1", true)
	if err != nil {
		t.Error("Error starting Consul ", err)
	}
}

func TestGetIp(t *testing.T) {
	for i := 1; i < 250; i++ {
		addressStr, err := GetAnAddress("192.168.1.0/24")
		if err != nil {
			log.Println(err)
			t.Fatal(err)
		}
		address := net.ParseIP(addressStr).To4()
		if i != int(address[3]) {
			t.Error(addressStr)
		}
	}
}

func TestCleanup(t *testing.T) {
	ecc.Delete(dataStore, "192.168.1.0/24")
	Leave()
}
