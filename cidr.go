package ipcs

import (
	"fmt"
	"math"
	"net"
	"sort"
	"strings"
)

func SplitCIDR(cidr string) (string, int) {
	tmp := strings.Split(cidr, "/")
	if len(tmp) == 2 {
		return tmp[0], toInt(tmp[1])
	} else {
		return tmp[0], 32
	}
}

func NewCIDR(ip string, mask int) *CIDR {
	c := &CIDR{IP: ParseIP(ip), Mask: mask}
	c.maskIP = MaskToIP(mask, c.ver)
	c.Reset()
	return c
}

func ParseCIDR(target string) (*CIDR, error) {
	// return ip, hosts
	var ip string
	var mask int
	target = ParseHost(target)
	if strings.Contains(target, "/") {
		ip, mask = SplitCIDR(target)
	} else {
		ip = target
		mask = 32
	}

	if parsedIp, err := ParseHostToIP(ip); err == nil {
		return NewCIDR(parsedIp.String(), mask), nil
	} else {
		return nil, err
	}
}

type CIDR struct {
	*IP
	Mask   int
	maskIP net.IP
	cur    net.IP
	count  int
	max    int
}

func (c *CIDR) Len() int {
	return c.IP.Len()
}

func (c *CIDR) String() string {
	return fmt.Sprintf("%s/%d", c.IP.String(), c.Mask)
}

func (c *CIDR) IPString() string {
	return c.IP.String()
}

func (c *CIDR) FirstIP() *IP {
	if c.ver == 4 {
		ip := make(net.IP, 4)
		for i := 0; i < 4; i++ {
			ip[i] = c.IP.IP[i] & c.maskIP[i]
		}
		return &IP{IP: ip}
	} else {
		ip := make(net.IP, 16)
		for i := 0; i < 16; i++ {
			ip[i] = c.IP.IP[i] & c.maskIP[i]
		}
		return &IP{IP: ip}
	}
}

//func (c *CIDR) FirstIP() *IP {
//	return NewIP(c.IP.Int() & MaskToIPInt(c.Mask))
//}

func (c *CIDR) LastIP() *IP {
	if c.ver == 4 {
		ip := make(net.IP, 4)
		for i := 0; i < 4; i++ {
			ip[i] = c.IP.IP[i] | ^c.maskIP[i]
		}
		return &IP{IP: ip}
	} else {
		ip := make(net.IP, 16)
		for i := 0; i < 16; i++ {
			ip[i] = c.IP.IP[i] | ^c.maskIP[i]
		}
		return &IP{IP: ip}
	}
}

//func (c *CIDR) LastIP() *IP {
//	return NewIP(c.IP.Int() | ^MaskToIPInt(c.Mask))
//}

func (c *CIDR) Net() *net.IPNet {
	return &net.IPNet{c.IP.IP, net.IPMask(MaskToIP(c.Mask, c.ver))}
}

func (c *CIDR) NetWithMask(mask int) *net.IPNet {
	return &net.IPNet{c.IP.IP, net.IPMask(MaskToIP(c.Mask, c.ver))}
}

func (c *CIDR) IPMask() net.IPMask {
	if c.ver == 4 {
		return net.CIDRMask(c.Mask, 32)
	} else {
		return net.CIDRMask(c.Mask, 128)
	}
}

func (c *CIDR) Count() int {
	if c.ver == 4 {
		return 1 << uint(32-c.Mask)
	} else {
		return 1 << uint(128-c.Mask)
	}
}

func (c *CIDR) Range() (first, final uint) {
	if c.ver == 6 {
		return 0, 0
	}
	first = c.FirstIP().Int()
	final = first | uint(math.Pow(2, float64(32-c.Mask))-1)
	return first, final
}

func (c *CIDR) RangeIP() (firstip, finalip *IP) {
	if c.ver == 6 {
		return nil, nil
	}
	firstip = c.FirstIP()
	finalip = NewIP(firstip.Int() | uint(math.Pow(2, float64(32-c.Mask))-1))
	return firstip, finalip
}

func (c *CIDR) ContainsCIDR(cidr *CIDR) bool {
	return c.Net().Contains(cidr.IP.IP)
}

func (c *CIDR) ContainsIP(ip *IP) bool {
	return c.Net().Contains(ip.IP)
}

func (c *CIDR) Next() *IP {
	if c.count == 0 {
		c.count++
		return &IP{IP: c.cur}
	}
	if c.count >= c.max {
		c.Reset()
		return c.Next()
	}

	c.cur[c.Len()-1]++
	c.count++
	for i := len(c.cur) - 1; i > 0; i-- {
		if c.cur[i] == 0 {
			c.cur[i-1]++
			if c.cur[i-1] != 0 {
				break
			} else {
				continue
			}
		} else {
			break
		}
	}
	newip := make(net.IP, c.Len())
	copy(newip, c.cur)
	return &IP{IP: newip}
}

func (c *CIDR) Reset() {
	c.max = c.Count()
	c.count = 0
	c.cur = c.FirstIP().IP
}

type CIDRs []*CIDR

func (cs CIDRs) Less(i, j int) bool {
	ipi := cs[i].FirstIP().Int()
	ipj := cs[j].FirstIP().Int()
	if ipi == ipj {
		if cs[i].Mask < cs[j].Mask {
			return true
		} else {
			return false
		}
	} else if ipi < ipj {
		return true
	} else {
		return false
	}
}

func (cs CIDRs) Swap(i, j int) {
	cs[i], cs[j] = cs[j], cs[i]
}

func (cs CIDRs) Len() int {
	return len(cs)
}

func (cs CIDRs) Strings() []string {
	s := make([]string, len(cs))
	for i, cidr := range cs {
		s[i] = cidr.String()
	}
	return s
}

func (cs CIDRs) Coalesce() CIDRs {
	sort.Sort(cs)
	var newCIDRs CIDRs
	for i := 0; i < len(cs)-1; i++ {
		j := i
		for j < len(cs)-1 {
			if !cs[j].ContainsCIDR(cs[j+1]) {
				break
			} else {
				j++
			}
		}
		cs[i].IP = cs[i].FirstIP()
		newCIDRs = append(newCIDRs, cs[i])
		i = j
	}
	return newCIDRs
}

func (cs CIDRs) Count() int {
	var sum int
	for _, c := range cs {
		sum += c.Count()
	}
	return sum
}
