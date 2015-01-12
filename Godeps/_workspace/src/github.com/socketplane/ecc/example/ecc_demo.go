package main

import (
	"fmt"
	"os"
	"os/signal"
	"time"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/ecc"
)

const dataDir = "/tmp/ecc"

type eccListener struct {
}

func (e eccListener) NotifyNodeUpdate(nType ecc.NotifyUpdateType, nodeName string) {
	fmt.Println("CLIENT UPDATE :", nType, nodeName)
}
func (e eccListener) NotifyKeyUpdate(nType ecc.NotifyUpdateType, key string, data []byte) {
	fmt.Println("KEY UPDATE :", nType, key, data)
}
func (e eccListener) NotifyStoreUpdate(nType ecc.NotifyUpdateType, store string, data map[string][]byte) {
}
func main() {
	ecc.Start(true, true, "eth1", dataDir)
	listener := eccListener{}
	go ecc.RegisterForNodeUpdates(listener)
	go ecc.RegisterForKeyUpdates("network", "web", listener)
	keyUpdates("web")
	ecc.Delete("network", "web")
	go ecc.RegisterForKeyUpdates("network", "db", listener)
	keyUpdates("db")
	keyUpdates("web")
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

//Random Key updates
func keyUpdates(key string) {
	valArray, _, _ := ecc.Get("network", key)
	currVal := make([]byte, len(valArray))
	copy(currVal, valArray)
	valArray = []byte("value1")
	ecc.Put("network", key, valArray, currVal)
	time.Sleep(time.Second * 2)
	updArray := []byte("value2")
	ecc.Put("network", key, updArray, valArray)
	time.Sleep(time.Second * 2)
}
