package ecc

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"reflect"
	"time"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/armon/consul-api"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/hashicorp/consul/command"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/hashicorp/consul/watch"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/mitchellh/cli"
)

// Embedded Consul Client
// Quick and Dirty way to embed Consul with any golang based application without the
// additional step of installing the Consul application in the host system

// Consul Agent related functions

var started bool
var OfflineSupport bool = true

func Start(serverMode bool, bootstrap bool, bindInterface string, dataDir string) error {
	watches = make(map[WatchType]watchData)
	bindAddress := ""
	if bindInterface != "" {
		intf, err := net.InterfaceByName(bindInterface)
		if err != nil {
			log.Printf("Error : %v", err)
			return err
		}
		addrs, err := intf.Addrs()
		if err == nil {
			for i := 0; i < len(addrs); i++ {
				addr := addrs[i].String()
				ip, _, _ := net.ParseCIDR(addr)
				if ip != nil && ip.To4() != nil {
					bindAddress = ip.To4().String()
				}
			}
		}
	}
	errCh := make(chan int)
	go startConsul(serverMode, bootstrap, bindAddress, dataDir, errCh)

	select {
	case <-errCh:
		return errors.New("Error starting Consul Agent")
	case <-time.After(time.Second * 5):
	}
	go populateKVStoreFromCache()
	return nil
}

func startConsul(serverMode bool, bootstrap bool, bindAddress string, dataDir string, eCh chan int) {
	args := []string{"agent", "-data-dir", dataDir}

	if serverMode {
		args = append(args, "-server")
	}

	if bootstrap {
		args = append(args, "-bootstrap")
	}

	if bindAddress != "" {
		args = append(args, "-bind")
		args = append(args, bindAddress)
	}

	ret := Execute(args...)

	if ret != 0 {
		eCh <- ret
	}
}

func HasStarted() bool {
	return started
}

func Join(address string) error {
	ret := Execute("join", address)

	if ret != 0 {
		log.Println("Error (%d) joining %s with Consul peers", ret, address)
		return errors.New("Error adding member")
	}
	return nil
}

func Leave() error {
	ret := Execute("leave")
	if ret != 0 {
		log.Println("Error Leaving Consul membership")
		return errors.New("Error leaving Consul cluster")
	}
	time.Sleep(time.Second * 3)
	return nil
}

// Execute function is borrowed from Consul's main.go
func Execute(args ...string) int {

	for _, arg := range args {
		if arg == "-v" || arg == "--version" {
			newArgs := make([]string, len(args)+1)
			newArgs[0] = "version"
			copy(newArgs[1:], args)
			args = newArgs
			break
		}
	}

	cli := &cli.CLI{
		Args:     args,
		Commands: Commands,
		HelpFunc: cli.BasicHelpFunc("consul"),
	}

	exitCode, err := cli.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing CLI: %s\n", err.Error())
		return 1
	}

	return exitCode
}

const CONSUL_BASE_URL = "http://localhost:8500/v1/"

func ConsulGet(url string) (string, bool) {
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error (%v) in Get for %s\n", err, url)
		return "", false
	}
	defer resp.Body.Close()
	log.Printf("Status of Get %s %d for %s", resp.Status, resp.StatusCode, url)
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var jsonBody []consulBody
		body, err := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(body, &jsonBody)
		existingValue, err := b64.StdEncoding.DecodeString(jsonBody[0].Value)
		if err != nil {
			return "", false
		}
		return string(existingValue[:]), true
	} else {
		return "", false
	}
}

// Consul KV Store related

const CONSUL_KV_BASE_URL = "http://localhost:8500/v1/kv/"

type consulBody struct {
	CreateIndex int    `json:"CreateIndex,omitempty"`
	ModifyIndex int    `json:"ModifyIndex,omitempty"`
	Key         string `json:"Key,omitempty"`
	Flags       int    `json:"Flags,omitempty"`
	Value       string `json:"Value,omitempty"`
}

func GetAll(store string) ([][]byte, []int, bool) {
	if OfflineSupport && !started {
		return getAllFromCache(store)
	}
	url := CONSUL_KV_BASE_URL + store + "?recurse"
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error (%v) in Get for %s\n", err, url)
		return nil, nil, false
	}
	defer resp.Body.Close()
	log.Printf("Status of Get %s %d for %s", resp.Status, resp.StatusCode, url)
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, nil, false
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var jsonBody []consulBody
		valueArr := make([][]byte, 0)
		indexArr := make([]int, 0)
		body, _ := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(body, &jsonBody)
		for _, body := range jsonBody {
			existingValue, _ := b64.StdEncoding.DecodeString(body.Value)
			valueArr = append(valueArr, existingValue)
			indexArr = append(indexArr, body.ModifyIndex)
		}
		return valueArr, indexArr, true
	} else {
		return nil, nil, false
	}
}

func Get(store string, key string) ([]byte, int, bool) {
	if OfflineSupport && !started {
		return getFromCache(store, key)
	}
	url := CONSUL_KV_BASE_URL + store + "/" + key
	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error (%v) in Get for %s\n", err, url)
		return nil, 0, false
	}
	defer resp.Body.Close()
	log.Printf("Status of Get %s %d for %s", resp.Status, resp.StatusCode, url)
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return nil, 0, false
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var jsonBody []consulBody
		body, err := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(body, &jsonBody)
		existingValue, err := b64.StdEncoding.DecodeString(jsonBody[0].Value)
		if err != nil {
			return nil, jsonBody[0].ModifyIndex, false
		}
		return existingValue, jsonBody[0].ModifyIndex, true
	} else {
		return nil, 0, false
	}
}

const (
	OK = iota
	OUTDATED
	ERROR
)

type eccerror int

func Put(store string, key string, value []byte, oldValue []byte) eccerror {
	if OfflineSupport && !started {
		return putInCache(store, key, value, oldValue)
	}
	existingValue, casIndex, ok := Get(store, key)
	if ok && !bytes.Equal(oldValue, existingValue) {
		return OUTDATED
	}
	url := fmt.Sprintf("%s%s/%s?cas=%d", CONSUL_KV_BASE_URL, store, key, casIndex)
	log.Printf("Updating KV pair for %s %s %s %d", url, key, value, casIndex)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(value))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error creating KV pair for ", url, err)
		return ERROR
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) == "false" {
		// Let the application retry based on return value
		// return Put(store, key, value, oldValue)
		return OUTDATED
	}
	return OK
}

func Delete(store string, key string) eccerror {
	if OfflineSupport && !started {
		return deleteFromCache(store, key)
	}
	url := fmt.Sprintf("%s%s/%s", CONSUL_KV_BASE_URL, store, key)
	log.Printf("Deleting KV pair for %s", url)
	req, err := http.NewRequest("DELETE", url, nil)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error creating KV pair for ", url, err)
		return ERROR
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	log.Println(string(body))
	return OK
}

type Store struct {
	cache map[string][]byte
}

// Local KV store cache for bootstrap node consul connection issues
var cache map[string]Store = make(map[string]Store)

func getAllFromCache(storeName string) ([][]byte, []int, bool) {
	store, ok := cache[storeName]
	if !ok {
		return nil, nil, false
	}
	vals := make([][]byte, 0)
	for _, val := range store.cache {
		vals = append(vals, val)
	}
	return vals, nil, true
}

func getFromCache(storeName string, key string) ([]byte, int, bool) {
	store, ok := cache[storeName]
	if !ok {
		return nil, 0, false
	}
	val, ok := store.cache[key]
	return val, 0, ok
}

func putInCache(storeName string, key string, value []byte, oldValue []byte) eccerror {
	store, ok := cache[storeName]
	if !ok {
		store = Store{make(map[string][]byte)}
		cache[storeName] = store
	}
	store.cache[key] = value
	return OK
}

func deleteFromCache(storeName string, key string) eccerror {
	store, ok := cache[storeName]
	if ok {
		delete(store.cache, key)
	}
	return OK
}

func populateKVStoreFromCache() {
	if !OfflineSupport {
		return
	}
	started = true
	for storeName, store := range cache {
		for key, val := range store.cache {
			go Put(storeName, key, val, nil)
		}
		delete(cache, storeName)
	}
}

// Watch related

const (
	NOTIFY_UPDATE_ADD = iota
	NOTIFY_UPDATE_MODIFY
	NOTIFY_UPDATE_DELETE
)

type NotifyUpdateType int

const (
	WATCH_TYPE_NODE = iota
	WATCH_TYPE_KEY
	WATCH_TYPE_STORE
	WATCH_TYPE_EVENT
)

type WatchType int

type watchData struct {
	listeners map[string][]Listener
}

var watches map[WatchType]watchData

type Listener interface {
	NotifyNodeUpdate(NotifyUpdateType, string)
	NotifyKeyUpdate(NotifyUpdateType, string, []byte)
	NotifyStoreUpdate(NotifyUpdateType, string, map[string][]byte)
}

func contains(wtype WatchType, key string, elem interface{}) bool {
	ws, ok := watches[wtype]
	if !ok {
		return false
	}

	list, ok := ws.listeners[key]
	if !ok {
		return false
	}

	v := reflect.ValueOf(list)
	for i := 0; i < v.Len(); i++ {
		if v.Index(i).Interface() == elem {
			return true
		}
	}
	return false
}

type watchconsul bool

func addListener(wtype WatchType, key string, listener Listener) watchconsul {
	var wc watchconsul = false
	if !contains(WATCH_TYPE_NODE, key, listener) {
		ws, ok := watches[wtype]
		if !ok {
			watches[wtype] = watchData{make(map[string][]Listener)}
			ws = watches[wtype]
		}

		listeners, ok := ws.listeners[key]
		if !ok {
			listeners = make([]Listener, 0)
			wc = true
		}
		ws.listeners[key] = append(listeners, listener)
	}
	return wc
}

func getListeners(wtype WatchType, key string) []Listener {
	ws, ok := watches[wtype]
	if !ok {
		return nil
	}

	list, ok := ws.listeners[key]
	if ok {
		return list
	}
	return nil
}

func register(params map[string]interface{}, handler watch.HandlerFunc) {
	// Create the watch
	wp, err := watch.Parse(params)
	if err != nil {
		fmt.Printf("Register error : %s", err)
		return
	}
	wp.Handler = handler
	cmdFlags := flag.NewFlagSet("watch", flag.ContinueOnError)
	httpAddr := command.HTTPAddrFlag(cmdFlags)
	// Run the watch
	if err := wp.Run(*httpAddr); err != nil {
		fmt.Printf("Error querying Consul agent: %s", err)
	}
}

var nodeCache []*consulapi.Node

func compare(X, Y []*consulapi.Node) []*consulapi.Node {
	m := make(map[string]bool)

	for _, y := range Y {
		m[y.Address] = true
	}

	var ret []*consulapi.Node
	for _, x := range X {
		if m[x.Address] {
			continue
		}
		ret = append(ret, x)
	}

	return ret
}

func updateNodeListeners(clusterNodes []*consulapi.Node) {
	toDelete := compare(nodeCache, clusterNodes)
	toAdd := compare(clusterNodes, nodeCache)
	nodeCache = clusterNodes
	listeners := getListeners(WATCH_TYPE_NODE, "")
	if listeners == nil {
		return
	}
	for _, deleteNode := range toDelete {
		for _, listener := range listeners {
			listener.NotifyNodeUpdate(NOTIFY_UPDATE_DELETE, deleteNode.Address)
		}
	}
	for _, addNode := range toAdd {
		for _, listener := range listeners {
			listener.NotifyNodeUpdate(NOTIFY_UPDATE_ADD, addNode.Address)
		}
	}
}

func RegisterForNodeUpdates(listener Listener) {
	wc := addListener(WATCH_TYPE_NODE, "", listener)
	if wc {
		// Compile the watch parameters
		params := make(map[string]interface{})
		params["type"] = "nodes"
		handler := func(idx uint64, data interface{}) {
			updateNodeListeners(data.([]*consulapi.Node))
		}
		register(params, handler)
	}
}

func RegisterForKeyUpdates(key string, listener Listener) {
	wc := addListener(WATCH_TYPE_KEY, key, listener)
	if wc {
		// Compile the watch parameters
		params := make(map[string]interface{})
		params["type"] = "key"
		params["key"] = key
		handler := func(idx uint64, data interface{}) {
			fmt.Println("NOT IMPLEMENTED Key Update :", idx, data)
		}
		register(params, handler)
	}
}

func RegisterForStoreUpdates(store string, listener Listener) {
	wc := addListener(WATCH_TYPE_STORE, store, listener)
	if wc {
		// Compile the watch parameters
		params := make(map[string]interface{})
		params["type"] = "keyprefix"
		params["prefix"] = store + "/"
		handler := func(idx uint64, data interface{}) {
			fmt.Println("NOT IMPLEMENTED Store Update :", idx, data)
		}
		register(params, handler)
	}
}
