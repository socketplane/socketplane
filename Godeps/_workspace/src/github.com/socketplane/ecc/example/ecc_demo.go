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
}
func (e eccListener) NotifyStoreUpdate(nType ecc.NotifyUpdateType, store string, data map[string][]byte) {
}
func main() {
	ecc.Start(true, true, "eth1", dataDir)
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
