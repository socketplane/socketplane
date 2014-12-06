package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os/exec"
	"time"
)
import b64 "encoding/base64"

const ipamUrl = "http://localhost:8500/v1/kv/ipam/"

func IsIpamAvailable() bool {
	cmd := exec.Command("which", "consul")
	app, err := cmd.Output()
	if err != nil || string(app) == "" {
		return false
	}
	return true
}

func ManageIPAddress(bindInterface string, bootstrap bool) error {
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
	errCh := make(chan error)
	go startConsul(bindAddress, bootstrap, errCh)

	select {
	case res := <-errCh:
		return res
	case <-time.After(time.Second * 5):
	}
	return nil
}

func GetAnAddress(subnet string) (string, error) {
	return checkAndUpdateConsulKV(subnet)
}

func Join(address string) {
	var cmd *exec.Cmd
	cmd = exec.Command(consulApp, "join", address)
	_, err := cmd.Output()

	if err != nil {
		log.Println(err)
	}
}

func Leave() {
	var cmd *exec.Cmd
	cmd = exec.Command(consulApp, "leave")
	_, err := cmd.Output()

	if err != nil {
		log.Println(err)
	}
	time.Sleep(time.Second * 1)
}

func createConsulKV(subnet string) (string, error) {
	address, _, err := net.ParseCIDR(subnet)
	address = address.To4()
	if err != nil || address == nil {
		log.Printf("%v is not an IPv4 address\n", address)
		return "", errors.New(subnet + "is not an IPv4 address")
	}
	// Hack : Not a bigger hack than this entire naive and dirty IPAM solution :-)
	address[3] = 2
	log.Printf("Creating KV pair for %s %s", ipamUrl+subnet, address.String())
	req, err := http.NewRequest("PUT", ipamUrl+subnet, bytes.NewBuffer([]byte(address.String())))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Println("Error creating KV pair for ", ipamUrl+subnet, err)
		return "", errors.New("Error creating KV pair ")
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if string(body) == "false" {
		return checkAndUpdateConsulKV(subnet)
	} else {
		address[3] = 1
		return address.String(), nil
	}
}

type consulBody struct {
	CreateIndex int    `json:"CreateIndex,omitempty"`
	ModifyIndex int    `json:"ModifyIndex,omitempty"`
	Key         string `json:"Key,omitempty"`
	Flags       int    `json:"Flags,omitempty"`
	Value       string `json:"Value,omitempty"`
}

func checkAndUpdateConsulKV(subnet string) (string, error) {
	_, network, err := net.ParseCIDR(subnet)
	if err != nil {
		log.Printf("invalid subnet : %s", subnet)
		return "", errors.New("invalid subnet " + subnet)
	}

	subnet = network.String()
	resp, err := http.Get(ipamUrl + subnet)
	if err != nil {
		log.Printf("Error getting subnet information %v for %s\n", err, ipamUrl+subnet)
		return "", errors.New("Unable to obtain subnet info from Consul " + ipamUrl + subnet)
	}
	defer resp.Body.Close()
	log.Printf("Status of Get %s %d for %s", resp.Status, resp.StatusCode, ipamUrl+subnet)
	if resp.StatusCode >= 400 && resp.StatusCode < 500 {
		return createConsulKV(subnet)
	} else if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		var jsonBody []consulBody
		body, err := ioutil.ReadAll(resp.Body)
		err = json.Unmarshal(body, &jsonBody)
		addressStr, err := b64.StdEncoding.DecodeString(jsonBody[0].Value)
		fmt.Printf("b=%s j=%s %s\n", body, jsonBody[0].Value, string(addressStr[:]))
		address := net.ParseIP(string(addressStr[:]))
		if err != nil || address.To4() == nil {
			log.Printf("%s is not an IPv4 address : %v\n", address.String(), err)
			return "", errors.New("Not a valid Ipv4 address " + address.String())
		}
		address = address.To4()
		address[3] = address[3] + 1

		url := fmt.Sprintf("%s%s?cas=%d", ipamUrl, subnet, jsonBody[0].ModifyIndex)
		log.Printf("Updating KV pair for %s %s %d", ipamUrl+subnet, address.String(), jsonBody[0].ModifyIndex)
		req, err := http.NewRequest("PUT", url, bytes.NewBuffer([]byte(address.String())))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Println("Error creating KV pair for ", ipamUrl+subnet, err)
			return "", errors.New("Error creating KV pair for " + ipamUrl + subnet)
		}
		defer resp.Body.Close()

		body, _ = ioutil.ReadAll(resp.Body)
		if string(body) == "false" {
			checkAndUpdateConsulKV(subnet)
		}
		address[3] = address[3] - 1
		return address.String(), nil
	}
	return "", errors.New("Error updating consul KV pair " + resp.Status)
}

var consulApp string

func startConsul(bindAddress string, bootstrap bool, errCh chan error) {
	var cmd *exec.Cmd
	cmd = exec.Command("which", "consul")
	app, err := cmd.Output()
	if err != nil {
		log.Printf("Error starting Consul : %v", err)
		errCh <- err
		return
	}
	consulApp = string(app[:len(app)-1])

	args := []string{"agent", "-server", "-data-dir", "/tmp/consul"}
	if bootstrap {
		args = append(args, "-bootstrap")
	}

	if bindAddress != "" {
		args = append(args, "-bind")
		args = append(args, bindAddress)
	}

	cmd = exec.Command(consulApp, args...)
	_, err = cmd.Output()

	if err != nil {
		log.Println(err)
		errCh <- err
	}
}

/*
func main() {
    ManageIPAddress("192.168.1.7", []string{"10.6.0.0/24"})
    time.Sleep(time.Second * 10)
    for i := 0; i < 10; i++ {
        addr, err := GetAnAddress("10.6.0.0/24")
        fmt.Println(addr, err)
        if err != nil {
            leave()
            return
        }
    }
    select {}
}
*/
