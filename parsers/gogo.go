package parsers

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/chainreactors/logs"
	"github.com/chainreactors/utils/fileutils"
)

func ParseGogoData(filename string) (*GOGOData, error) {
	var err error
	file, err := fileutils.Open(filename)
	if err != nil {
		return nil, err
	}
	content := fileutils.DecryptFile(file, fileutils.Key)
	content = bytes.TrimSpace(content) // 去除前后空格
	lines := bytes.Split(content, []byte{0x0a})

	var finished bool = true
	// 判断扫描是否结束
	if !bytes.Equal(lines[len(lines)-1], []byte("[\"done\"]")) {
		finished = false
		logs.Log.Important("Task has not been completed,auto fix json")
		logs.Log.Important("Task has not been completed,auto fix json")
		logs.Log.Important("Task has not been completed,auto fix json")
	}

	// 删除最后一行
	var last int
	if finished {
		last = len(lines) - 1
	} else {
		last = len(lines)
	}
	var res bytes.Buffer
	rd := &GOGOData{}
	res.WriteString("[")
	res.Write(bytes.Join(lines[1:last], []byte{','}))
	res.WriteString("]")
	err = json.Unmarshal(res.Bytes(), &rd.Data)
	if err != nil {
		return nil, err
	}
	return rd, nil
}

func NewGOGOResult(ip, port string) *GOGOResult {
	return &GOGOResult{
		Ip:         ip,
		Port:       port,
		Protocol:   "tcp",
		Status:     "tcp",
		Frameworks: make(Frameworks),
		Vulns:      make(Vulns),
	}
}

type GOGOResult struct {
	Ip         string              `json:"ip"`
	Port       string              `json:"port"`
	Protocol   string              `json:"protocol"`
	Status     string              `json:"status"`
	Uri        string              `json:"uri,omitempty"`
	Host       string              `json:"host,omitempty"`
	Frameworks Frameworks   `json:"frameworks,omitempty"`
	Vulns      Vulns        `json:"vulns,omitempty"`
	Extracteds map[string][]string `json:"extracted,omitempty"`
	Title      string              `json:"title,omitempty"`
	Midware    string              `json:"midware,omitempty"`
	Timing     int64               `json:"timing,omitempty"`
}

func (result *GOGOResult) IsHttp() bool {
	if strings.HasPrefix(result.Protocol, "http") {
		return true
	}
	return false
}

func (result *GOGOResult) GetBaseURL() string {
	return fmt.Sprintf("%s://%s:%s", result.Protocol, result.Ip, result.Port)
}

func (result *GOGOResult) GetTarget() string {
	return fmt.Sprintf("%s:%s", result.Ip, result.Port)
}

func (result *GOGOResult) GetURL() string {
	if result.IsHttp() {
		return result.GetBaseURL() + result.Uri
	} else {
		return result.GetBaseURL()
	}
}

func (result *GOGOResult) GetExtractStat() string {
	if len(result.Extracteds) > 0 {
		var s []string
		for name, ss := range result.Extracteds {
			if tmps := strings.Join(ss, ","); len(tmps) < 50 {
				s = append(s, fmt.Sprintf("%s:%s", name, tmps))
			} else {
				s = append(s, fmt.Sprintf("%s:%d items", name, len(ss)))
			}
		}
		return fmt.Sprintf("[ extracts: %s ]", strings.Join(s, ", "))
	} else {
		return ""
	}
}

func (result *GOGOResult) NoFramework() bool {
	if len(result.Frameworks) == 0 {
		return true
	}
	return false
}

func (result *GOGOResult) GetFirstFramework() *Framework {
	if !result.NoFramework() {
		for _, frame := range result.Frameworks {
			return frame
		}
	}
	return nil
}

func (result *GOGOResult) Get(key string) string {
	switch key {
	case "ip":
		return result.Ip
	case "port":
		return result.Port
	case "status", "stat":
		return result.Status
	case "frameworks", "framework", "frame":
		return result.Frameworks.String()
	case "cpe", "fsb":
		return strings.Join(result.Frameworks.CPE(), ",")
	case "uri":
		return strings.Join(result.Frameworks.URI(), ",")
	case "wfn":
		return strings.Join(result.Frameworks.WFN(), ",")
	case "vulns", "vuln":
		return result.Vulns.String()
	case "host", "cert":
		return result.Host
	case "title":
		return result.Title
	case "target":
		return result.GetTarget()
	case "url":
		return result.GetBaseURL()
	case "midware":
		return result.Midware
	case "timing":
		return strconv.FormatInt(result.Timing, 10)
	case "protocol", "scheme":
		return result.Protocol
	case "extract", "extracts":
		var s strings.Builder
		s.WriteString("[ ")
		for k, v := range result.Extracteds {
			s.WriteString(fmt.Sprintf("%s:%s,", k, strings.Join(v, ",")))
		}
		s.WriteString(" ]")
		return s.String()
	default:
		return ""
	}
}

func (result *GOGOResult) FramesColorString() string {
	var ss []string
	for _, f := range result.Frameworks {
		if f.IsFocus {
			ss = append(ss, logs.RedBold(strings.Replace(f.String(), "focus:", "", -1)))
			//s.WriteString(logs.RedBold(" [" + strings.Replace(f.String(), "focus:", "", -1) + "]"))
		} else {
			ss = append(ss, logs.Cyan(f.String()))
			//s.WriteString(logs.Cyan(" [" + f.String() + "]"))
		}
	}
	return strings.Join(ss, "||")
}

func (result *GOGOResult) ColorOutput() string {
	s := fmt.Sprintf("[+] %s %s\t%s\t%s\t%s [%s] %s %s\n", logs.Yellow(fmt.Sprintf("[%dms]", result.Timing)), result.GetURL(), result.Midware, result.FramesColorString(), result.Host, logs.Yellow(result.Status), logs.GreenLine(result.Title), logs.Red(result.Vulns.String()))
	return s
}

func (result *GOGOResult) FullOutput() string {
	s := fmt.Sprintf("[+] [%dms] %s\t%s\t%s\t%s [%s] %s %s %s\n", result.Timing, result.GetURL(), result.Midware, result.Frameworks.String(), result.Host, result.Status, result.Title, result.Vulns.String(), result.GetExtractStat())
	return s
}

func (result *GOGOResult) OutputLine() string {
	if result == nil {
		return ""
	}
	target := result.GetTarget()
	if result.IsHttp() {
		target = result.GetBaseURL()
	}
	time := ""
	if result.Timing > 0 {
		time = fmt.Sprintf("%dms", result.Timing)
	}
	return JoinOutput(
		target,
		result.Status,
		result.Protocol,
		result.Host,
		result.Midware,
		result.Title,
		FrameworkOutput(result.Frameworks),
		strings.TrimSpace(result.Vulns.String()),
		strings.TrimSpace(result.GetExtractStat()),
		time,
	)
}

func (result *GOGOResult) String() string {
	if result == nil {
		return ""
	}
	target := result.GetTarget()
	if result.IsHttp() {
		target = result.GetBaseURL()
	}
	parts := []string{target}
	if result.Protocol != "" {
		parts = append(parts, "proto="+result.Protocol)
	}
	if result.Status != "" {
		parts = append(parts, "status="+result.Status)
	}
	if result.Midware != "" {
		parts = append(parts, "midware="+result.Midware)
	}
	if result.Host != "" {
		parts = append(parts, "host="+result.Host)
	}
	if result.Title != "" {
		parts = append(parts, fmt.Sprintf("title=%q", result.Title))
	}
	if frameworks := strings.Trim(result.Frameworks.String(), "|"); frameworks != "" {
		parts = append(parts, "frameworks="+frameworks)
	}
	if vulns := strings.TrimSpace(result.Vulns.String()); vulns != "" {
		parts = append(parts, "vulns="+vulns)
	}
	if extract := strings.TrimSpace(result.GetExtractStat()); extract != "" {
		parts = append(parts, extract)
	}
	if result.Timing > 0 {
		parts = append(parts, fmt.Sprintf("time=%dms", result.Timing))
	}
	return strings.Join(parts, " ")
}

func (result *GOGOResult) JsonOutput() string {
	jsons, _ := json.Marshal(result)
	return string(jsons)
}

func (result *GOGOResult) CsvOutput() string {
	var sb strings.Builder
	w := csv.NewWriter(&sb)
	record := []string{
		result.Ip,
		result.Port,
		result.GetURL(),
		result.Status,
		result.Title,
		result.Host,
		result.Midware,
		result.Frameworks.String(),
		result.Vulns.String(),
	}
	w.Write(record)
	w.Flush()
	return sb.String()
}

func (result *GOGOResult) ValuesOutput(outType string) string {
	outs := strings.Split(outType, ",")
	for i, out := range outs {
		outs[i] = result.Get(out)
	}
	return strings.Join(outs, "\t") + "\n"
}

func (result *GOGOResult) Filter(k, v, op string) bool {
	var matchfunc func(string, string) bool
	if op == "::" {
		matchfunc = strings.Contains
	} else if op == "==" {
		if k == "tag" {
			return result.Frameworks.HasTag(v)
		} else if k == "from" {
			return result.Frameworks.HasFrom(v)
		} else {
			matchfunc = strings.EqualFold
		}
	} else if op == "!:" {
		matchfunc = notContains
	} else if op == "!=" {
		if k == "tag" {
			return !result.Frameworks.HasTag(v)
		} else if k == "from" {
			return !result.Frameworks.HasFrom(v)
		} else {
			matchfunc = notEqualFlod
		}
	} else {
		logs.Log.Error("illegal operator, please input one of [::, ==, !:, !=]")
	}

	if matchfunc(strings.ToLower(result.Get(k)), strings.ToLower(v)) {
		return true
	}
	return false
}

type GOGOConfig struct {
	IP            string   `json:"ip"`
	IPlist        []string `json:"ips"`
	Ports         string   `json:"ports"`
	JsonFile      string   `json:"json_file"`
	ListFile      string   `json:"list_file"`
	Threads       int      `json:"threads"` // 线程数
	Mod           string   `json:"mod"`     // 扫描模式
	NoScan        bool     `json:"no_scan"`
	AliveSprayMod []string `json:"alive_spray"`
	PortSpray     bool     `json:"port_spray"`
	Exploit       string   `json:"exploit"`
	JsonType      string   `json:"json_type"`
	VersionLevel  int      `json:"version_level"`
}

func (config *GOGOConfig) GetTargetName() string {
	var target string
	if config.ListFile != "" {
		target = config.ListFile
	} else if config.JsonFile != "" {
		target = config.JsonFile
	} else if config.Mod == "a" {
		target = "auto"
	} else if config.IP != "" {
		target = config.IP
	}
	return target
}

type GOGOResults []*GOGOResult

func (rs GOGOResults) FilterWithString(name string) GOGOResults {
	// 过滤指定数据
	var results GOGOResults
	if name == "focus" {
		results = rs.Filter("frame", "focus", "::")
	} else if name == "vuln" {
		results = rs.Filter("vuln", "high", "::")
		results = append(results, rs.Filter("vuln", "critical", "::")...)
	} else if name == "domain" {
		//todo
	} else {
		// 过滤指定数据
		if strings.Contains(name, "::") {
			kv := strings.Split(name, "::")
			results = rs.Filter(kv[0], kv[1], "::")
		} else if strings.Contains(name, "==") {
			kv := strings.Split(name, "==")
			results = rs.Filter(kv[0], kv[1], "==")
		} else if strings.Contains(name, "!=") {
			kv := strings.Split(name, "!=")
			results = rs.Filter(kv[0], kv[1], "!=")
		} else if strings.Contains(name, "!:") {
			kv := strings.Split(name, "!:")
			results = rs.Filter(kv[0], kv[1], "!:")
		}
	}

	return results
}

func (rs GOGOResults) Filter(k, v, op string) GOGOResults {
	var filtedres GOGOResults
	for _, result := range rs {
		if result.Filter(k, v, op) {
			filtedres = append(filtedres, result)
		}
	}
	return filtedres
}

func (rs GOGOResults) GetValues(key string) []string {
	values := make([]string, len(rs))
	for i, result := range rs {
		//if focus && !result.Frameworks.IsFocus() {
		//	// 如果需要focus, 则跳过非focus标记的framework
		//	continue
		//}
		values[i] = result.Get(key)
	}
	return values
}

type GOGOData struct {
	Config *GOGOConfig `json:"config"`
	IP     string      `json:"ip"`
	Data   GOGOResults `json:"data"`
}

func (rd *GOGOData) Filter(name string) GOGOResults {
	return rd.Data.FilterWithString(name)
}

func (rd *GOGOData) ToConfig() string {
	// 输出配置信息
	var configstr string
	configstr = fmt.Sprintf("Scan Target: %s, Ports: %s, Mod: %s \n", rd.Config.GetTargetName(), rd.Config.Ports, rd.Config.Mod)
	configstr += fmt.Sprintf("Exploit: %s, Version level: %d \n", rd.Config.Exploit, rd.Config.VersionLevel)
	if rd.IP != "" {
		configstr += fmt.Sprintf("Internet IP: %s", rd.IP)
	}
	return configstr
}

func (rd *GOGOData) ToValues(outType string) string {
	outs := strings.Split(outType, ",")
	outvalues := make([][]string, len(outs))
	ss := make([]string, len(rd.Data))
	for i, out := range outs {
		outvalues[i] = rd.Data.GetValues(out)
	}

	for i := 0; i < len(ss); i++ {
		for j := 0; j < len(outvalues); j++ {
			ss[i] += outvalues[j][i] + "\t"
		}
		strings.TrimSpace(ss[i])
	}

	return strings.Join(ss, "\n")
}

func (rd *GOGOData) ToZombie() []*ZombieInput {
	var zms []*ZombieInput
	for _, r := range rd.Data {
		for name, _ := range r.Frameworks {
			if service, ok := ZombieMap[name]; ok {
				zms = append(zms, &ZombieInput{
					Scheme:  r.Protocol,
					IP:      r.Ip,
					Port:    r.Port,
					Service: service,
				})
			}
		}
		for _, v := range r.Vulns {
			for _, tag := range v.Tags {
				if service, ok := ZombieMap[tag]; ok {
					zms = append(zms, &ZombieInput{
						Scheme:  r.Protocol,
						IP:      r.Ip,
						Port:    r.Port,
						Service: service,
					})
				}
			}
		}
	}
	return zms
}

func (rd *GOGOData) ToJson() string {
	content, _ := json.Marshal(rd)
	return string(content)
}

func (rd *GOGOData) ToCsv() string {
	var s strings.Builder
	s.WriteString("ip,port,url,status,title,host,language,midware,frame,vuln,extract\n")
	for _, r := range rd.Data {
		s.WriteString(r.CsvOutput())
	}
	return s.String()
}
