package datastore

import (
	"errors"
    "os"
    "os/signal"
    "time"
	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
	"github.com/socketplane/socketplane/ovs"
	"net"
)

const (
    dataDir = "/tmp/socketplane"
    CLUSTER_PREFIX = "cluster"
)

type ClusterNode struct{}

type eccListener struct{}

func Init(bindInterface string, bootstrap bool) error {
	return ecc.Start(bootstrap, bootstrap, bindInterface, dataDir)
}

func Join(address string) error {
	return ecc.Join(address)
}

func Leave() error {
	return ecc.Leave()
}

func (e eccListener) NotifyNodeUpdate(nType ecc.NotifyUpdateType, nodeName string) {
	log.Debug("CLIENT UPDATE :", nType, nodeName)
}

func (e eccListener) NotifyKeyUpdate(nType ecc.NotifyUpdateType, key string, data []byte) {
}

func (e eccListener) NotifyStoreUpdate(nType ecc.NotifyUpdateType, store string, data map[string][]byte) {
}

func WatchNewNodes(iface string, dirPrefix string) {
	ecc.Start(true, true, iface, dirPrefix)
	go ecc.RegisterForNodeUpdates(eccListener{})
	// Ctrl+C handling
	handler := make(chan os.Signal, 1)
	signal.Notify(handler, os.Interrupt)
	for sig := range handler {
		if sig == os.Interrupt {
			time.Sleep(1e9)
			break
		}
	}
}

func (n ClusterNode) NewClusterNode(addr net.IP) error {
	log.Info("New Member Added : ", addr)
	Join(addr.String())
	// add the nodes prefix to the k/v store
	addNodePrefix(parseAddrStr(addr))
	// add the local node tunnels
	err := ovs.AddPeer(addr.String())
	if err != nil {
		return errors.New("Failed to adding new node")
	}
	log.Info("Added cluster member : ", addr)
	return nil
}

func addNodePrefix(nodeAddr string) error {
	eccerr := ecc.Put(CLUSTER_PREFIX, nodeAddr, nil, nil)
	if eccerr == ecc.OUTDATED {
		return addNodePrefix(nodeAddr)
	}
	return nil
}

// Added to remove orphaned hosts
func (n ClusterNode) RemoveClusterNode(addr net.IP) error {
	log.Info("Member Left : ", addr)
	// remove the nodes prefix to the k/v store
	delNodePrefix(parseAddrStr(addr))
	// remove the local node tunnels
	err := ovs.DeletePeer(addr.String())
	if err != nil {
		return errors.New("Failed to adding new node")
	}
	log.Info("Deleted cluster member : ", addr)
	return nil
}

// Added to remove orphaned hosts
func delNodePrefix(nodeAddr string) error {
	if ClusterNodeExists(nodeAddr) != true {
		return errors.New("Error deleting node")
	}
	eccerror := ecc.Delete(CLUSTER_PREFIX, nodeAddr)
	if eccerror == ecc.OK {
		delNodePrefix(nodeAddr)
		return nil
	}
	return errors.New("Error deleting node")
}

// check if a prefix exists
func ClusterNodeExists(nodeAddr string) bool {
	_, _, ok := ecc.Get(CLUSTER_PREFIX, nodeAddr)
	if ok {
		return true
	}
	return false
}

func parseAddrStr(ipStr net.IP) string {
	ipAddr, _ := net.ResolveIPAddr("ip", ipStr.String())
	return ipAddr.String()
}
