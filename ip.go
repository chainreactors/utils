package utils

import (
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

func IsIp(ip string) bool {
	if net.ParseIP(ip) != nil {
		return true
	}
	return false
}

func MaskToIPv4(mask int) net.IP {
	subnetMask := make([]byte, net.IPv4len) // 创建长度为4的字节数组
	for i := 0; i < mask; i++ {
		subnetMask[i/8] |= 1 << uint(7-i%8) // 根据子网掩码长度设置相应位为1
	}
	return subnetMask
}

func MaskToIPv6(mask int) net.IP {
	subnetMask := make([]byte, net.IPv6len) // 创建长度为4的字节数组
	for i := 0; i < mask; i++ {
		subnetMask[i/8] |= 1 << uint(7-i%8) // 根据子网掩码长度设置相应位为1
	}
	return subnetMask
}

func MaskToIP(mask, ver int) net.IP {
	if ver == 4 {
		return MaskToIPv4(mask)
	} else if ver == 6 {
		return MaskToIPv6(mask)
	}
	return nil
}

//func Ip2Int(ip string) uint {
//	s2ip := net.ParseIP(ip).To4()
//	return uint(binary.BigEndian.Uint32(s2ip))
//}
//
//func Int2Ip(ipint uint) string {
//	ip := net.IP{byte(ipint >> 24), byte(ipint >> 16), byte(ipint >> 8), byte(ipint)}
//	return ip.String()
//}

func DistinguishIPVersion(ip net.IP) int {
	switch len(ip) {
	case net.IPv4len:
		return 4
	case net.IPv6len:
		return 6
	}
	return 0
}

func ParseIP(s string) *IP {
	ip := net.ParseIP(s)
	if ip == nil {
		return nil
	}
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '.':
			return &IP{IP: ip, ver: 4}
		case ':':
			return &IP{IP: ip, ver: 6}
		}
	}
	return nil
}

func NewIP(ipint uint) *IP {
	return &IP{IP: net.IP{byte(ipint >> 24), byte(ipint >> 16), byte(ipint >> 8), byte(ipint)}, ver: 4}
}

// ParseHostToIP parse host to ip and validate ip format
func ParseHostToIP(target string) (*IP, error) {
	target = strings.TrimSpace(target)
	if ip := ParseIP(target); ip != nil {
		return ip, nil
	}

	iprecords, err := net.LookupIP(target)
	if err != nil {
		return nil, fmt.Errorf("Unable to resolve domain name:" + target + ". SKIPPED!")
	}

	for _, ip := range iprecords {
		if ip != nil {
			//Log.Important("parse domain SUCCESS, map " + target + " to " + ip.String())
			return &IP{ip, DistinguishIPVersion(ip), target}, nil
		}
	}
	return nil, fmt.Errorf("not found Ip address")
}

type IP struct {
	IP   net.IP
	ver  int
	Host string
}

func (ip IP) Len() int {
	return len(ip.IP)
}

func (ip IP) Int() uint {
	if ip.ver == 4 {
		return uint(binary.BigEndian.Uint32(ip.IP.To4()))
	}
	return 0
}

func (ip IP) String() string {
	return ip.IP.String()
}

func (ip IP) Mask(mask int) IP {
	return IP{IP: MaskToIP(mask, ip.ver)}
}

// NewIPs parse string to ip , auto skip wrong ip
func NewIPs(input []string) IPs {
	ips := make(IPs, len(input))
	for _, ip := range input {
		i, err := ParseHostToIP(ip)
		if err != nil {
			continue
		}
		ips = append(ips, i)
	}
	return ips
}

type IPs []*IP

func (is IPs) Less(i, j int) bool {
	ipi := is[i].Int()
	ipj := is[j].Int()
	if ipi < ipj {
		return true
	} else {
		return false
	}
}

func (is IPs) Swap(i, j int) {
	is[i], is[j] = is[j], is[i]
}

func (is IPs) Len() int {
	return len(is)
}

func (is IPs) Strings() []string {
	s := make([]string, len(is))
	for i, cidr := range is {
		s[i] = cidr.String()
	}
	return s
}

func (is IPs) Approx() CIDRs {
	cidrMap := make(map[string]*CIDR)

	for _, ip := range is {
		if n, ok := cidrMap[ip.Mask(24).String()]; ok {
			var baseNet byte
			var nowN, newN byte
			for i := 8; i > 0; i-- {
				nowN = n.IP.IP[3] & (1 << uint(i-1)) >> uint(i-1)
				newN = ip.IP[3] & (1 << uint(i-1)) >> uint(i-1)
				if nowN&newN == 1 {
					baseNet += 1 << uint(i-1)
				}
				if nowN^newN == 1 {
					n.Mask = 32 - i
					n.IP.IP[3] = baseNet
					break
				}
			}
		} else {
			cidrMap[ip.Mask(24).String()] = NewCIDR(ip.String(), 32)
		}
	}

	approxed := make(CIDRs, len(cidrMap))
	var index int
	for _, cidr := range cidrMap {
		approxed[index] = cidr
		index++
	}

	return approxed
}
