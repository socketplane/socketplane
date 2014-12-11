package ecc

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/mitchellh/cli"
)

// Embedded Consul Client
// Quick and Dirty way to embed Consul with any golang based application without the
// additional step of installing the Consul application in the host system

// Consul Agent related functions

func Start(serverMode bool, bootstrap bool, bindInterface string, dataDir string) error {
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
	log.SetOutput(ioutil.Discard)

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

func Get(store string, key string) ([]byte, int, bool) {
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
