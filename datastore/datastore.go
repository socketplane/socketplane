package datastore

import "github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"

const dataDir = "/tmp/socketplane"

func Init(bindInterface string, bootstrap bool) error {
	return ecc.Start(true, bootstrap, bindInterface, dataDir)
}

func Join(address string) error {
	return ecc.Join(address)
}

func Leave() error {
	return ecc.Leave()
}
