package main

import (
	"fmt"
	"net"
	"os"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/bonjour"
)

const DOCKER_CLUSTER_SERVICE = "_foobar._service"
const DOCKER_CLUSTER_DOMAIN = "local"

func main() {
	var intfName = ""
	if len(os.Args) > 1 {
		intfName = os.Args[1]
	}
	b := bonjour.Bonjour{
		ServiceName:     DOCKER_CLUSTER_SERVICE,
		ServiceDomain:   DOCKER_CLUSTER_DOMAIN,
		ServicePort:     9999,
		InterfaceName:   intfName,
		OnMemberHello:   newMember,
		OnMemberGoodBye: removeMember,
	}
	b.Start()

	select {}
}

func newMember(addr net.IP) {
	fmt.Println("New Member Added : ", addr)
}
func removeMember(addr net.IP) {
	fmt.Println("Member Left : ", addr)
}
