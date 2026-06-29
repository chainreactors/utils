package parsers

import (
	"regexp"
	"strings"
)

var (
	TitleRegexp  = regexp.MustCompile("(?Uis)<title>(.*)</title>")
	ServerRegexp = regexp.MustCompile("(?i)Server: ([\x20-\x7e]+)")
	//XPBRegexp           = regexp.MustCompile("(?i)X-Powered-By: ([\x20-\x7e]+)")
	//SessionRegexp       = regexp.MustCompile("(?i) (.*SESS.*?ID)")
	HeaderCharsetRegexp = regexp.MustCompile("(?i)Content-Type:.*charset=(.+)")
	BodyCharsetRegexp   = regexp.MustCompile("(?i)<meta.*?charset=[\"']?(.*?)[\"' >]")
)

func MatchOne(reg *regexp.Regexp, s []byte) (string, bool) {
	matched := reg.FindSubmatch(s)
	if len(matched) == 0 {
		return "", false
	}
	if len(matched) == 1 {
		return "", true
	} else {
		return strings.TrimSpace(string(matched[1])), true
	}
}

func MatchCharset(content []byte) string {
	charset, ok := MatchOne(HeaderCharsetRegexp, content)
	if ok {
		return charset
	}

	charset, ok = MatchOne(BodyCharsetRegexp, content)
	if ok {
		return charset
	}
	return ""
}

func MatchTitle(content []byte) string {
	title, ok := MatchOne(TitleRegexp, content)
	if ok {
		return title
	}
	return ""
}

func MatchCharacter(content []byte) string {
	if len(content) > 13 {
		return string(content[0:13])
	} else {
		return string(content)
	}
}

//func MatchLanguage(resp *http.Response) string {
//	var powered string
//	powered = resp.Header.Get("X-Powered-By")
//	if powered != "" {
//		return powered
//	}
//
//	cookies := getCookies(resp)
//	if cookies["JSESSIONID"] != "" {
//		return "JAVA"
//	} else if cookies["ASP.NET_SessionId"] != "" {
//		return "ASP"
//	} else if cookies["PHPSESSID"] != "" {
//		return "PHP"
//	} else {
//		return ""
//	}
//}

//func MatchLanguageWithRaw(content []byte) string {
//	powered, ok := MatchOne(XPBRegexp, content)
//	if ok {
//		return powered
//	}
//
//	sessionid, ok := MatchOne(SessionRegexp, content)
//	if ok {
//		switch sessionid {
//		case "JSESSIONID":
//			return "JAVA"
//		case "ASP.NET_SessionId":
//			return "ASP.NET"
//		case "PHPSESSID":
//			return "PHP"
//		}
//	}
//	return ""
//}

func notContains(s, substr string) bool {
	return !strings.Contains(s, substr)
}

func notEqualFlod(s, substr string) bool {
	return !strings.EqualFold(s, substr)
}
