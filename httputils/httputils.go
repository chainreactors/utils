package httputils

import (
	"bufio"
	"bytes"
	"github.com/chainreactors/utils/iutils"
	"io"
	"io/ioutil"
	"net/http"
)

var (
	// from feroxbuster
	RandomUA = []string{
		"Mozilla/5.0 (Linux; Android 8.0.0; SM-G960F Build/R16NW) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/62.0.3202.84 Mobile Safari/537.36",
		"Mozilla/5.0 (iPhone; CPU iPhone OS 12_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/12.0 Mobile/15E148 Safari/604.1",
		"Mozilla/5.0 (Windows Phone 10.0; Android 6.0.1; Microsoft; RM-1152) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/52.0.2743.116 Mobile Safari/537.36 Edge/15.15254",
		"Mozilla/5.0 (Linux; Android 7.0; Pixel C Build/NRD90M; wv) AppleWebKit/537.36 (KHTML, like Gecko) Version/4.0 Chrome/52.0.2743.98 Safari/537.36",
		"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/42.0.2311.135 Safari/537.36 Edge/12.246",
		"Mozilla/5.0 (X11; CrOS x86_64 8172.45.0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/51.0.2704.64 Safari/537.36",
		"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_11_2) AppleWebKit/601.3.9 (KHTML, like Gecko) Version/9.0.2 Safari/601.3.9",
		"Mozilla/5.0 (Windows NT 6.1; WOW64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/47.0.2526.111 Safari/537.36",
		"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:15.0) Gecko/20100101 Firefox/15.0.1",
		"Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)",
		"Mozilla/5.0 (compatible; bingbot/2.0; +http://www.bing.com/bingbot.htm)",
		"Mozilla/5.0 (compatible; Yahoo! Slurp; http://help.yahoo.com/help/us/ysearch/slurp)",
	}
)

func GetRandomUA() string {
	return iutils.RandomChoice(RandomUA)
}

func NewResponseWithRaw(raw []byte) *http.Response {
	if bytes.HasPrefix(raw, []byte("http/")) {
		raw = append([]byte("HTTP/"), raw[5:]...)
	}
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(raw)), nil)
	if err != nil {
		return nil
	}

	return resp
}

func SplitHttpRaw(content []byte) (body, header []byte, ok bool) {
	cs := bytes.Index(content, []byte("\r\n\r\n"))
	if cs != -1 && len(content) >= cs+4 {
		body = content[cs+4:]
		header = content[:cs]
		return body, header, true
	}
	return nil, nil, false
}

func ReadRaw(resp *http.Response) []byte {
	var raw bytes.Buffer
	raw.Write(ReadRawHeader(resp))
	raw.WriteString("\r\n")
	raw.Write(ReadBody(resp))
	return raw.Bytes()
}

func ReadRawWithSize(resp *http.Response, size int64) []byte {
	var raw bytes.Buffer
	raw.Write(ReadRawHeader(resp))
	raw.WriteString("\r\n")
	raw.Write(ReadBodyWithSize(resp, size))
	return raw.Bytes()
}

func ReadBodyWithSize(resp *http.Response, size int64) []byte {
	//io.LimitReader(resp.Body, size)
	body, err := ioutil.ReadAll(io.LimitReader(resp.Body, size))
	if err != nil {
		return body
	}
	return body
}

func ReadBody(resp *http.Response) []byte {
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return body
	}
	_ = resp.Body.Close()
	return body
}

func ReadHeader(resp *http.Response) []byte {
	var header bytes.Buffer
	for k, v := range resp.Header {
		for _, i := range v {
			header.WriteString(k + ": " + i + "\r\n")
		}
	}
	return header.Bytes()
}

func ReadRawHeader(resp *http.Response) []byte {
	var raw bytes.Buffer
	raw.WriteString(resp.Proto + " " + resp.Status + "\r\n")
	raw.Write(ReadHeader(resp))
	return raw.Bytes()
}

func ReadCookie(resp *http.Response) map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range resp.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}
