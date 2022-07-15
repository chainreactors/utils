package icps

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
	i, _ := ParseIP(ip)
	return &CIDR{*i, mask}
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

	if parsedIp, err := ParseIP(ip); err == nil {
		return &CIDR{*parsedIp, mask}, nil
	} else {
		return nil, err
	}
}

type CIDR struct {
	IP   IP
	Mask int
}

func (c CIDR) String() string {
	return fmt.Sprintf("%s/%d", c.IP.String(), c.Mask)
}

func (c CIDR) IPString() string {
	return c.IP.String()
}

func (c CIDR) FirstIP() *IP {
	return NewIPWithInt(c.IP.Int() & MaskToIPInt(c.Mask))
}

func (c CIDR) LastIP() *IP {
	return NewIPWithInt(c.IP.Int() | ^MaskToIPInt(c.Mask))
}

func (c CIDR) Net() *net.IPNet {
	return &net.IPNet{c.IP.IP, net.CIDRMask(c.Mask, 32)}
}

func (c CIDR) NetWithMask(mask int) *net.IPNet {
	return &net.IPNet{c.IP.IP, net.CIDRMask(mask, 32)}
}

func (c CIDR) IPMask() net.IPMask {
	return net.CIDRMask(c.Mask, 32)
}

func (c CIDR) Count() uint {
	return 1 << (32 - c.Mask)
}

func (c CIDR) Range() (first, final uint) {
	first = c.IP.Int()
	final = first | uint(math.Pow(2, float64(32-c.Mask))-1)
	return first, final
}

func (c CIDR) RangeIP() (first, final uint) {
	first = c.IP.Int()
	final = first | uint(math.Pow(2, float64(32-c.Mask))-1)
	return first, final
}

func (c CIDR) ContainsCIDR(cidr CIDR) bool {
	return c.Net().Contains(cidr.IP.IP)
}

func (c CIDR) ContainsIP(ip IP) bool {
	return c.Net().Contains(ip.IP)
}

type CIDRs []CIDR

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
		cs[i].IP = *cs[i].FirstIP()
		newCIDRs = append(newCIDRs, cs[i])
		i = j
	}
	return newCIDRs
}

func MaskToIPInt(mask int) uint {
	return uint(((uint64(4294967296) >> uint(32-mask)) - 1) << uint(32-mask))
}
