package icps

import (
	"fmt"
	"strings"
)

func ParseCIDR(target string) (string, error) {
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
		return fmt.Sprintf("%s:%s", parsedIp, mask), nil
	} else {
		return "", err
	}
}

func SplitCIDR(cidr string) (string, int) {
	tmp := strings.Split(cidr, "/")
	if len(tmp) == 2 {
		return tmp[0], toInt(tmp[1])
	} else {
		return tmp[0], 32
	}
}
