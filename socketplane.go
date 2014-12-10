package main

import (
	"fmt"
	"github.com/socketplane/socketplane/ipam"
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/bonjour"
)

const DOCKER_CLUSTER_SERVICE = "_docker._cluster"
const DOCKER_CLUSTER_SERVICE_PORT = 9999 //TODO : fix this
const DOCKER_CLUSTER_DOMAIN = "local"

func main() {
	go Bonjour("eth1")
	ipam.Init("", true)
	fmt.Println("HELLO SOCKETPLANE")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for _ = range c {
			os.Exit(0)
		}
	}()
	select {}
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
