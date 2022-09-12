package ipcs

import (
	"fmt"
	"strings"
)

func NewAddrWithPort(ip, port string) *Addr {
	return &Addr{NewIP(ip), port}
}

func NewAddr(s string) *Addr {
	if pair := strings.Split(s, ":"); len(pair) == 2 {
		return &Addr{NewIP(pair[0]), pair[1]}
	}
	return nil
}

type Addr struct {
	IP   *IP
	Port string
}

func NewAddrs(ss []string) Addrs {
	addrs := make(Addrs, len(ss))
	for _, s := range ss {
		if addr := NewAddr(s); addr != nil {
			addrs = append(addrs, addr)
		}
	}
	return addrs
}

func NewAddrsWithDefaultPort(ss []string, port string) Addrs {
	addrs := make(Addrs, len(ss))
	for _, s := range ss {
		if addr := NewAddr(s); addr != nil {
			addrs = append(addrs, addr)
		} else if ip := NewIP(s); ip != nil {
			addrs = append(addrs, &Addr{ip, port})
		}
	}
	return addrs
}

type Addrs []*Addr

func (as Addrs) Set() {

}

func (a Addr) String() string {
	return fmt.Sprintf("%s:%s", a.IP.String(), a.Port)
}

func NewAddrsWithPorts(ips []string, ports interface{}) *AddrsGenerator {
	switch ports.(type) {
	case string:
		return &AddrsGenerator{NewIPs(ips), NewPorts(ports.(string))}
	default:
		return &AddrsGenerator{NewIPs(ips), ports.([]string)}
	}
}

type AddrsGenerator struct {
	IPs   IPs
	Ports Ports
}

func (as AddrsGenerator) Count() int {
	return len(as.IPs) * len(as.Ports)
}

func (as AddrsGenerator) GenerateWithIP() chan *Addr {
	gen := make(chan *Addr)
	go func() {
		for _, ip := range as.IPs {
			for _, port := range as.Ports {
				gen <- &Addr{ip, port}
			}
		}
		close(gen)
	}()
	return gen
}

func (as AddrsGenerator) GenerateWithPort() chan *Addr {
	gen := make(chan *Addr)
	go func() {
		for _, port := range as.Ports {
			for _, ip := range as.IPs {
				gen <- &Addr{ip, port}
			}
		}
		close(gen)
	}()
	return gen
}
