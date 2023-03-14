package utils

import (
	"bytes"
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

func MaskToIPv4(mask int) *IP {
	subnetMask := make([]byte, net.IPv4len) // 创建长度为4的字节数组
	for i := 0; i < mask; i++ {
		subnetMask[i/8] |= 1 << uint(7-i%8) // 根据子网掩码长度设置相应位为1
	}
	return &IP{IP: subnetMask, ver: 4}
}

func MaskToIPv6(mask int) *IP {
	subnetMask := make([]byte, net.IPv6len) // 创建长度为4的字节数组
	for i := 0; i < mask; i++ {
		subnetMask[i/8] |= 1 << uint(7-i%8) // 根据子网掩码长度设置相应位为1
	}
	return &IP{IP: subnetMask, ver: 6}
}

func MaskToIP(mask, ver int) *IP {
	if ver == 4 {
		return MaskToIPv4(mask)
	} else if ver == 6 {
		return MaskToIPv6(mask)
	}
	return nil
}

func Ip2Intv4(ip string) uint {
	s2ip := net.ParseIP(ip).To4()
	return uint(binary.BigEndian.Uint32(s2ip))
}

func Int2Ipv4(ipint uint) string {
	ip := net.IP{byte(ipint >> 24), byte(ipint >> 16), byte(ipint >> 8), byte(ipint)}
	return ip.String()
}

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
			return &IP{IP: ip.To4(), ver: 4}
		case ':':
			return &IP{IP: ip.To16(), ver: 6}
		}
	}
	return nil
}

//func NewIP(ipint uint) *IP {
//	return &IP{IP: net.IP{byte(ipint >> 24), byte(ipint >> 16), byte(ipint >> 8), byte(ipint)}, ver: 4}
//}
func NewIP(ip net.IP) *IP {
	i := &IP{IP: ip}
	if len(i.IP) == net.IPv4len {
		i.ver = 4
	} else {
		i.ver = 6
	}
	return i
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

func (ip *IP) Len() int {
	if ip.ver == 4 {
		return net.IPv4len
	} else if ip.ver == 6 {
		return net.IPv6len
	} else {
		return 0
	}
}

func (ip *IP) Int() uint {
	if ip.ver == 4 {
		return uint(binary.BigEndian.Uint32(ip.IP.To4()))
	}
	return 0
}

func (ip *IP) String() string {
	return ip.IP.String()
}

func (ip *IP) Mask(mask int) *IP {
	newip := make(net.IP, ip.Len())
	maskip := MaskToIP(mask, ip.ver)
	for i := 0; i < ip.Len(); i++ {
		newip[i] = ip.IP[i] & maskip.IP[i]
	}
	return &IP{IP: newip, ver: ip.ver}
}

func (ip *IP) MaskNet(mask *IP) *IP {
	newip := make(net.IP, ip.Len())
	for i := 0; i < ip.Len(); i++ {
		newip[i] = ip.IP[i] & mask.IP[i]
	}
	return &IP{IP: newip, ver: ip.ver}
}

func (ip *IP) Mask24() *IP {
	i := ip.Copy()
	i.IP[3] = 0
	return i
}

func (ip *IP) Mask16() *IP {
	i := ip.Copy()
	i.IP[3] = 0
	i.IP[4] = 0
	return i
}

func (ip *IP) Equal(other *IP) bool {
	return bytes.Equal(ip.IP, other.IP)
}

func (ip *IP) Compare(other *IP) int {
	if ip.ver != other.ver {
		return 1
	}
	return bytes.Compare(ip.IP, other.IP)
}

func (ip *IP) Copy() *IP {
	newip := make(net.IP, ip.Len())
	copy(newip, ip.IP)
	return &IP{IP: newip, ver: ip.ver, Host: ip.Host}
}

func (ip *IP) Next() *IP {
	ip.IP[ip.Len()-1]++
	for i := ip.Len() - 1; i > 0; i-- {
		if ip.IP[i] == 0 {
			ip.IP[i-1]++
			if ip.IP[i-1] != 0 {
				break
			} else {
				continue
			}
		} else {
			break
		}
	}
	return ip
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
	if is[i].Compare(is[j]) < 0 {
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
