package main

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/go-fastping"
)

type response struct {
	addr *net.IPAddr
	rtt  time.Duration
}

func main() {
	once := true
	if len(os.Args) < 2 {
		fmt.Fprintf(os.Stderr, "Usage: %s {hostname}\n", os.Args[0])
		os.Exit(1)
	}
	p := fastping.NewPinger()
	p.Debug = false

	netProto := "ip4:icmp"
	if strings.Index(os.Args[1], ":") != -1 {
		netProto = "ip6:ipv6-icmp"
	}
	ra, err := net.ResolveIPAddr(netProto, os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	if len(os.Args) > 2 && os.Args[2] != "" {
		iface, err := net.InterfaceByName(os.Args[2])
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		addrs, err := iface.Addrs()
		if err == nil {
			for i := 0; i < len(addrs); i++ {
				addr := addrs[i].String()
				ip, _, err := net.ParseCIDR(addr)
				if err == nil && ip != nil {
					if ip.To4() != nil {
						p.ListenAddr, _ = net.ResolveIPAddr("ip4", ip.To4().String())
					}
				}
			}
		}
	}

	results := make(map[string]*response)
	results[ra.String()] = nil
	p.AddIPAddr(ra)

	onRecv, onIdle := make(chan *response), make(chan bool)
	p.OnRecv = func(addr *net.IPAddr, t time.Duration) {
		fmt.Printf("%s : %v\n", addr.IP.String(), t)
		if !once {
			onRecv <- &response{addr: addr, rtt: t}
		}
	}
	p.OnIdle = func() {
		fmt.Printf("OnIdle %v", p.Err())
		if !once {
			onIdle <- true
		}
	}

	p.OnErr = func(addr *net.IPAddr, t int) {
		fmt.Printf("Error %s : %d\n", addr.IP.String(), t)
	}

	p.MaxRTT = time.Second
	if once {
		p.Run()
	} else {
		p.RunLoop()

		c := make(chan os.Signal, 1)
		signal.Notify(c, os.Interrupt)
		signal.Notify(c, syscall.SIGTERM)

	loop:
		for {
			select {
			case <-c:
				fmt.Println("get interrupted")
				break loop
			case res := <-onRecv:
				results[res.addr.String()] = res
			case <-onIdle:
				for host, r := range results {
					if r == nil {
						fmt.Printf("%s : unreachable %v\n", host, time.Now())
					} else {
						fmt.Printf("%s : %v %v\n", host, r.rtt, time.Now())
					}
					results[host] = nil
				}
			case <-p.Done():
				if err = p.Err(); err != nil {
					fmt.Println("Ping failed:", err)
				}
				break loop
			}
		}
		signal.Stop(c)
		p.Stop()
	}
}
