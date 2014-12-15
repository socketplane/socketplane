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
		ServiceName:   DOCKER_CLUSTER_SERVICE,
		ServiceDomain: DOCKER_CLUSTER_DOMAIN,
		ServicePort:   9999,
		InterfaceName: intfName,
		BindToIntf:    true,
		Notify:        notify{},
	}
	b.Start()

	select {}
}

type notify struct{}

func (n notify) NewMember(addr net.IP) {
	fmt.Println("New Member Added : ", addr)
}
func (n notify) RemoveMember(addr net.IP) {
	fmt.Println("Member Left : ", addr)
}
