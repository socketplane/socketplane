package main

import (
	"fmt"
	"log"
	"net"

	"github.com/socketplane/bonjour"
)

const DOCKER_CLUSTER_SERVICE = "_docker._cluster"
const DOCKER_CLUSTER_SERVICE_PORT = 9999 //TODO : fix this
const DOCKER_CLUSTER_DOMAIN = "local"

func main() {
	go Bonjour("eth1")
	fmt.Println("HELLO SOCKETPLANE")
}

func Bonjour(intfName string) {
	b := bonjour.Bonjour{
		ServiceName:     DOCKER_CLUSTER_SERVICE,
		ServiceDomain:   DOCKER_CLUSTER_DOMAIN,
		ServicePort:     DOCKER_CLUSTER_SERVICE_PORT,
		InterfaceName:   intfName,
		OnMemberHello:   newMember,
		OnMemberGoodBye: removeMember,
	}
	b.Start()
}

func newMember(addr net.IP) {
	log.Println("New Member Added : ", addr)
}
func removeMember(addr net.IP) {
	log.Println("Member Left : ", addr)
}
