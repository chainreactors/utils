package ipcs

import (
	"strconv"
	"strings"
)

var (
	NameMap PortMapper
	PortMap PortMapper
	TagMap  PortMapper
)

type PortMapper map[string][]string

func ParsePort(portstring string) []string {
	portstring = strings.TrimSpace(portstring)
	portstring = strings.Replace(portstring, "\r", "", -1)
	return ParsePorts(strings.Split(portstring, ","))
}

func ParsePorts(ports []string) []string {
	var portSlice []string
	for _, portname := range ports {
		portSlice = append(portSlice, choicePorts(portname)...)
	}
	portSlice = expandPorts(portSlice)
	portSlice = sliceUnique(portSlice)
	return portSlice
}

func expandPorts(ports []string) []string {
	// 将string格式的port range 转为单个port组成的slice
	var tmpports []string
	for _, pr := range ports {
		if len(pr) == 0 {
			continue
		}
		pr = strings.TrimSpace(pr)
		if pr[0] == 45 {
			pr = "1" + pr
		}
		if pr[len(pr)-1] == 45 {
			pr = pr + "65535"
		}
		tmpports = append(tmpports, expandPort(pr)...)
	}
	return tmpports
}

func expandPort(port string) []string {
	var tmpports []string
	if strings.Contains(port, "-") {
		sf := strings.Split(port, "-")
		start, _ := strconv.Atoi(sf[0])
		fin, _ := strconv.Atoi(sf[1])
		for port := start; port <= fin; port++ {
			tmpports = append(tmpports, strconv.Itoa(port))
		}
	} else {
		tmpports = append(tmpports, port)
	}
	return tmpports
}

// 端口预设
func choicePorts(portname string) []string {
	var ports []string
	if portname == "all" {
		for p := range PortMap {
			ports = append(ports, p)
		}
		return ports
	}

	if NameMap[portname] != nil {
		ports = append(ports, NameMap[portname]...)
		return ports
	} else if TagMap[portname] != nil {
		ports = append(ports, TagMap[portname]...)
		return ports
	} else {
		return []string{portname}
	}
}
