package httputils

import (
	"bufio"
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

func NewResponseWithRaw(raw []byte) *http.Response {
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
	raw.WriteString(resp.Proto + " " + resp.Status + "\r\n")
	raw.Write(ReadHeader(resp))
	raw.WriteString("\r\n")
	raw.Write(ReadBody(resp))
	return raw.Bytes()
}

func ReadRawWithSize(resp *http.Response, size int64) []byte {
	var raw bytes.Buffer
	raw.WriteString(resp.Proto + " " + resp.Status + "\r\n")
	raw.Write(ReadHeader(resp))
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
		if len(v) > 0 {
			header.WriteString(k + ": " + v[0] + "\r\n")
		}
	}
	return header.Bytes()
}

func ReadCookie(resp *http.Response) map[string]string {
	cookies := make(map[string]string)
	for _, cookie := range resp.Cookies() {
		cookies[cookie.Name] = cookie.Value
	}
	return cookies
}
