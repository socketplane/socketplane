package bonjour

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/socketplane/go-fastping"
)

type response struct {
	addr *net.IPAddr
	rtt  time.Duration
}

const (
	ECHO_REPLY = iota
	NO_REPLY
	ERROR
)

func echo(address string, ip *net.IP) (int, error) {
	p := fastping.NewPinger()
	p.Debug = false
	netProto := "ip4:icmp"
	if strings.Index(address, ":") != -1 {
		netProto = "ip6:ipv6-icmp"
	}
	ra, err := net.ResolveIPAddr(netProto, address)
	if err != nil {
		return ERROR, err
	}

	if ip != nil && ip.To4() != nil {
		p.ListenAddr, _ = net.ResolveIPAddr("ip4", ip.To4().String())
	}

	results := make(map[string]*response)
	results[ra.String()] = nil
	p.AddIPAddr(ra)

	onRecv, onIdle, onErr := make(chan *response), make(chan bool), make(chan int)

	p.OnRecv = func(addr *net.IPAddr, t time.Duration) {
		onRecv <- &response{addr: addr, rtt: t}
	}
	p.OnIdle = func() {
		onIdle <- true
	}

	p.OnErr = func(addr *net.IPAddr, t int) {
		onErr <- t
	}

	p.MaxRTT = time.Second
	go p.Run()

	ret := NO_REPLY
	select {
	case <-onRecv:
		ret = ECHO_REPLY
	case <-onIdle:
		ret = NO_REPLY
	case res := <-onErr:
		errId := fmt.Sprintf("%d", res)
		err = errors.New(errId)
		ret = ERROR
	}
	return ret, err
}
