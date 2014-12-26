package datastore

import (
	"errors"
	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
	"github.com/socketplane/socketplane/ovs"
	"net"
)

const dataDir = "/tmp/socketplane"

func Init(bindInterface string, bootstrap bool) error {
	return ecc.Start(bootstrap, bootstrap, bindInterface, dataDir)
}

func Join(address string) error {
	return ecc.Join(address)
}

func Leave() error {
	return ecc.Leave()
}

type EndPoint struct{}

func (n EndPoint) NewEndpoint(addr net.IP) error {
	log.Info("New Member Added : ", addr)
	Join(addr.String())
	err := ovs.AddPeer(addr.String())
	if err != nil {
		return errors.New("Failed to add new peer")
	}
	log.Info("Added Member : ", addr)
	return nil
}

func (n EndPoint) RemoveEndpoint(addr net.IP) error {
	log.Info("Member Left : ", addr)
	err := ovs.DeletePeer(addr.String())
	if err != nil {
		return errors.New("Failed to add new peer")
	}
	log.Info("Deleted Member : ", addr)
	return nil
}
