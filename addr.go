package ipcs

import "fmt"

type Addr struct {
	IP   *IP
	Port string
}

func (a Addr) String() string {
	return fmt.Sprintf("%s:%s", a.IP.String(), a.Port)
}

type Addrs struct {
	IPs   IPs
	Ports Ports
}

func (as Addrs) Count() int {
	return len(as.IPs) * len(as.Ports)
}

func (as Addrs) GenerateWithIP() chan *Addr {
	gen := make(chan *Addr)
	go func() {
		for _, ip := range as.IPs {
			for _, port := range as.Ports {
				gen <- &Addr{ip, port}
			}
		}
	}()
	return gen
}

func (as Addrs) GenerateWithPort() chan *Addr {
	gen := make(chan *Addr)
	go func() {
		for _, port := range as.Ports {
			for _, ip := range as.IPs {
				gen <- &Addr{ip, port}
			}
		}
	}()
	return gen
}
