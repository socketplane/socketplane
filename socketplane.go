package main

import (
	"log"
	"net"
	"os"
	"os/signal"

	"github.com/socketplane/socketplane/ipam"

	"github.com/codegangsta/cli"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/bonjour"
)

const DOCKER_CLUSTER_SERVICE = "_docker._cluster"
const DOCKER_CLUSTER_SERVICE_PORT = 9999 //TODO : fix this
const DOCKER_CLUSTER_DOMAIN = "local"

func main() {
	app := cli.NewApp()
	app.Name = "socketplane"
	app.Usage = "linux container networking"
	app.Version = "0.1.0"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "iface",
			Value: "auto",
			Usage: "Name of the interface to bind to. The default is to auto select",
		},
	}

	app.Action = func(ctx *cli.Context) {

		var bindInterface string
		if ctx.String("iface") != "auto" {
			bindInterface = ctx.String("iface")
		}
		go Bonjour(bindInterface)
		ipam.Init(bindInterface, true)
		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		go func() {
			for _ = range c {
				os.Exit(0)
			}
		}()
		select {}
	}

	app.Run(os.Args)
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
