package bonjour

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync"

	"github.com/socketplane/socketplane/Godeps/_workspace/src/github.com/miekg/dns"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/golang.org/x/net/ipv4"
	"github.com/socketplane/socketplane/Godeps/_workspace/src/golang.org/x/net/ipv6"
)

// Main client data structure to run browse/lookup queries
type Resolver struct {
	c    *client
	Exit chan<- bool
}

// Resolver structure constructor
func NewResolver(iface *net.Interface, serviceChan chan<- *ServiceEntry) (*Resolver, error) {
	c, err := newClient(iface, serviceChan)
	if err != nil {
		return nil, err
	}
	go c.mainloop()
	return &Resolver{c, c.closedCh}, nil
}

// Browse for all services of a fiven type in a given domain
func (r *Resolver) Browse(service, domain string) error {
	params := defaultParams(service)
	if domain != "" {
		params.Domain = domain
	}
	r.c.addLookupParams(params)

	err := r.c.query(params)
	if err != nil {
		r.Exit <- true
		return err
	}

	return nil
}

// Look up a specific service by its name and type in a given domain
func (r *Resolver) Lookup(instance, service, domain string) error {
	params := defaultParams(service)
	params.Instance = instance
	if domain != "" {
		params.Domain = domain
	}
	r.c.addLookupParams(params)

	err := r.c.query(params)
	if err != nil {
		r.Exit <- true
		return err
	}

	return nil
}

// defaultParams is used to return a default set of QueryParam's
func defaultParams(service string) *LookupParams {
	return NewLookupParams("", service, "local", make(chan *ServiceEntry))
}

func (c *client) addLookupParams(params *LookupParams) {
	if params.ServiceInstanceName() != "" {
		c.lookupParams[params.ServiceInstanceName()] = params
	} else {
		c.lookupParams[params.ServiceName()] = params
	}
}

func (c *client) getLookupParams(serviceName string) *LookupParams {
	return c.lookupParams[serviceName]
}

func (c *client) getMatchingParamsSuffix(serviceName string) (*LookupParams, bool) {
	for key, val := range c.lookupParams {
		if strings.HasSuffix(serviceName, key) {
			return val, true
		}
	}
	return nil, false
}

// Client structure incapsulates both IPv4/IPv6 UDP connections
type client struct {
	lookupParams map[string]*LookupParams
	serviceChan  chan<- *ServiceEntry
	ipv4conn     *net.UDPConn
	ipv6conn     *net.UDPConn
	closed       bool
	closedCh     chan bool
	closeLock    sync.Mutex
}

// Client structure constructor
func newClient(iface *net.Interface, serviceChan chan<- *ServiceEntry) (*client, error) {
	// Create wildcard connections (because :5353 can be already taken by other apps)
	ipv4conn, err := net.ListenUDP("udp4", mdnsWildcardAddrIPv4)
	if err != nil {
		log.Printf("[ERR] bonjour: Failed to bind to udp4 port: %v", err)
	}
	ipv6conn, err := net.ListenUDP("udp6", mdnsWildcardAddrIPv6)
	if err != nil {
		log.Printf("[ERR] bonjour: Failed to bind to udp6 port: %v", err)
	}
	if ipv4conn == nil && ipv6conn == nil {
		return nil, fmt.Errorf("[ERR] bonjour: Failed to bind to any udp port!")
	}

	// Join multicast groups to receive announcements from server
	p1 := ipv4.NewPacketConn(ipv4conn)
	p2 := ipv6.NewPacketConn(ipv6conn)
	if iface != nil {
		if err := p1.JoinGroup(iface, &net.UDPAddr{IP: mdnsGroupIPv4}); err != nil {
			return nil, err
		}
		if err := p2.JoinGroup(iface, &net.UDPAddr{IP: mdnsGroupIPv6}); err != nil {
			return nil, err
		}
	} else {
		ifaces, err := net.Interfaces()
		if err != nil {
			return nil, err
		}
		errCount1, errCount2 := 0, 0
		for _, iface := range ifaces {
			if err := p1.JoinGroup(&iface, &net.UDPAddr{IP: mdnsGroupIPv4}); err != nil {
				errCount1++
			}
			if err := p2.JoinGroup(&iface, &net.UDPAddr{IP: mdnsGroupIPv6}); err != nil {
				errCount2++
			}
		}
		if len(ifaces) == errCount1 && len(ifaces) == errCount2 {
			return nil, fmt.Errorf("Failed to join multicast group on all interfaces!")
		}
	}

	c := &client{
		lookupParams: make(map[string]*LookupParams),
		serviceChan:  serviceChan,
		ipv4conn:     ipv4conn,
		ipv6conn:     ipv6conn,
		closedCh:     make(chan bool),
	}

	return c, nil
}

// Start listeners and waits for the shutdown signal from exit channel
func (c *client) mainloop() {
	// start listening for responses
	msgCh := make(chan *dns.Msg, 32)
	if c.ipv4conn != nil {
		go c.recv(c.ipv4conn, msgCh)
	}
	if c.ipv6conn != nil {
		go c.recv(c.ipv6conn, msgCh)
	}

	// Iterate through channels from listeners goroutines
	var entries, sentEntries map[string]*ServiceEntry
	sentEntries = make(map[string]*ServiceEntry)
	for !c.closed {
		select {
		case <-c.closedCh:
			c.shutdown()
		case msg := <-msgCh:
			entries = make(map[string]*ServiceEntry)
			sections := append(msg.Answer, msg.Ns...)
			sections = append(sections, msg.Extra...)
			for _, answer := range sections {
				switch rr := answer.(type) {
				case *dns.PTR:
					params := c.getLookupParams(rr.Hdr.Name)
					if params == nil {
						continue
					}
					if _, ok := entries[rr.Ptr]; !ok {
						entries[rr.Ptr] = NewServiceEntry(
							trimDot(strings.Replace(rr.Ptr, rr.Hdr.Name, "", -1)),
							params.Service,
							params.Domain)
					}
					entries[rr.Ptr].TTL = rr.Hdr.Ttl
				case *dns.SRV:
					params := c.getLookupParams(rr.Hdr.Name)
					if params == nil {
						var ok bool
						params, ok = c.getMatchingParamsSuffix(rr.Hdr.Name)
						if !ok {
							continue
						}
					}
					if _, ok := entries[rr.Hdr.Name]; !ok {
						entries[rr.Hdr.Name] = NewServiceEntry(
							trimDot(strings.Replace(rr.Hdr.Name, params.ServiceName(), "", 1)),
							params.Service,
							params.Domain)
					}
					entries[rr.Hdr.Name].HostName = rr.Target
					entries[rr.Hdr.Name].Port = int(rr.Port)
					entries[rr.Hdr.Name].TTL = rr.Hdr.Ttl
				case *dns.TXT:
					params := c.getLookupParams(rr.Hdr.Name)
					if params == nil {
						var ok bool
						params, ok = c.getMatchingParamsSuffix(rr.Hdr.Name)
						if !ok {
							continue
						}
					}
					if _, ok := entries[rr.Hdr.Name]; !ok {
						entries[rr.Hdr.Name] = NewServiceEntry(
							trimDot(strings.Replace(rr.Hdr.Name, params.ServiceName(), "", 1)),
							params.Service,
							params.Domain)
					}
					entries[rr.Hdr.Name].Text = rr.Txt
					entries[rr.Hdr.Name].TTL = rr.Hdr.Ttl
				case *dns.A:
					for k, e := range entries {
						if e.HostName == rr.Hdr.Name && entries[k].AddrIPv4 == nil {
							entries[k].AddrIPv4 = rr.A
						}
					}
				case *dns.AAAA:
					for k, e := range entries {
						if e.HostName == rr.Hdr.Name && entries[k].AddrIPv6 == nil {
							entries[k].AddrIPv6 = rr.AAAA
						}
					}
				}
			}
		}

		if len(entries) > 0 {
			for k, e := range entries {
				if e.TTL == 0 {
					delete(entries, k)
					delete(sentEntries, k)
					continue
				}
				c.serviceChan <- e
			}
			// reset entries
			entries = make(map[string]*ServiceEntry)
		}
	}
}

// Shutdown client will close currently open connections & channel
func (c *client) shutdown() {
	c.closeLock.Lock()
	defer c.closeLock.Unlock()

	if c.closed {
		return
	}
	c.closed = true
	close(c.closedCh)

	if c.ipv4conn != nil {
		c.ipv4conn.Close()
	}
	if c.ipv6conn != nil {
		c.ipv6conn.Close()
	}
}

// Data receiving routine reads from connection, unpacks packets into dns.Msg
// structures and sends them to a given msgCh channel
func (c *client) recv(l *net.UDPConn, msgCh chan *dns.Msg) {
	if l == nil {
		return
	}
	buf := make([]byte, 65536)
	for !c.closed {
		n, _, err := l.ReadFrom(buf)
		if err != nil {
			continue
		}
		msg := new(dns.Msg)
		if err := msg.Unpack(buf[:n]); err != nil {
			log.Printf("[ERR] mdns: Failed to unpack packet: %v", err)
			continue
		}
		select {
		case msgCh <- msg:
		case <-c.closedCh:
			return
		}
	}
}

// Performs the actual query by service name (browse) or service instance name (lookup),
// start response listeners goroutines and loops over the entries channel.
func (c *client) query(params *LookupParams) error {
	var serviceName, serviceInstanceName string
	serviceName = fmt.Sprintf("%s.%s.", trimDot(params.Service), trimDot(params.Domain))
	if params.Instance != "" {
		serviceInstanceName = fmt.Sprintf("%s.%s", params.Instance, serviceName)
	}

	// send the query
	m := new(dns.Msg)
	if serviceInstanceName != "" {
		m.Question = []dns.Question{
			dns.Question{serviceInstanceName, dns.TypeSRV, dns.ClassINET},
			dns.Question{serviceInstanceName, dns.TypeTXT, dns.ClassINET},
		}
		m.RecursionDesired = false
	} else {
		m.SetQuestion(serviceName, dns.TypePTR)
		m.RecursionDesired = false
	}
	if err := c.sendQuery(m); err != nil {
		return err
	}

	return nil
}

// Pack the dns.Msg and write to available connections (multicast)
func (c *client) sendQuery(msg *dns.Msg) error {
	buf, err := msg.Pack()
	if err != nil {
		return err
	}
	if c.ipv4conn != nil {
		c.ipv4conn.WriteTo(buf, ipv4Addr)
	}
	if c.ipv4conn != nil {
		c.ipv4conn.WriteTo(buf, ipv6Addr)
	}
	return nil
}
