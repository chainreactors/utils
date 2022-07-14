package icps

import (
	"encoding/binary"
	"fmt"
	"net"
	"sort"
	"strings"
)

func IsIPv4(ip string) bool {
	address := net.ParseIP(ip).To4()
	if address != nil {
		return true
	}
	return false
}

func ParseIP(target string) (string, error) {
	target = strings.TrimSpace(target)
	if IsIPv4(target) {
		return target, nil
	}
	iprecords, err := net.LookupIP(target)
	if err != nil {
		return "", fmt.Errorf("Unable to resolve domain name:" + target + ". SKIPPED!")
	}

	for _, ip := range iprecords {
		if ip.To4() != nil {
			//Log.Important("parse domain SUCCESS, map " + target + " to " + ip.String())
			return ip.String(), nil
		}
	}
	return "", fmt.Errorf("not found Ip address")
}

func Ip2Int(ip string) uint {
	s2ip := net.ParseIP(ip).To4()
	return uint(binary.BigEndian.Uint32(s2ip))
}

func Int2Ip(ipint uint) string {
	ip := net.IP{byte(ipint >> 24), byte(ipint >> 16), byte(ipint >> 8), byte(ipint)}
	return ip.String()
}

func SortIP(ips []string) []string {
	sort.Slice(ips, func(i, j int) bool {
		return Ip2Int(ips[i]) < Ip2Int(ips[j])
	})
	return ips
}
