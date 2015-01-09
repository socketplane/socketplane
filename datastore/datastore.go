package datastore

import (
	"os"

	log "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/Sirupsen/logrus"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
)

const dataDir = "/tmp/socketplane"

var listener eccListener

func Init(bindInterface string, bootstrap bool) error {
	err := ecc.Start(bootstrap, bootstrap, bindInterface, dataDir)
	if err == nil {
		go ecc.RegisterForNodeUpdates(listener)
	}
	return err
}

func Join(address string) error {
	return ecc.Join(address)
}

func Leave() error {
	if err := ecc.Leave(); err != nil {
		log.Error(err)
		return err
	}
	if err := os.RemoveAll(dataDir); err != nil {
		log.Errorf("Error deleting data directory %s", err)
		return err
	}
	return nil
}

type eccListener struct {
}

func (e eccListener) NotifyNodeUpdate(nType ecc.NotifyUpdateType, nodeAddress string) {
	if nType == ecc.NOTIFY_UPDATE_ADD {
		log.Infof("New Node joined the cluster : %s", nodeAddress)
		// TODO : Add code here to handle new cluster node case
	} else if nType == ecc.NOTIFY_UPDATE_DELETE {
		log.Infof("Node left the cluster : %s", nodeAddress)
		// TODO : Add code here to handle node leaving the cluster case
	}
}

func (e eccListener) NotifyKeyUpdate(nType ecc.NotifyUpdateType, key string, data []byte) {
}
func (e eccListener) NotifyStoreUpdate(nType ecc.NotifyUpdateType, store string, data map[string][]byte) {
}
