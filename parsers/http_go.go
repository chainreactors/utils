//go:build !go1.17
// +build !go1.17

package parsers

import (
	"bufio"
	"bytes"
	"github.com/chainreactors/utils/encode"
	"github.com/chainreactors/utils/httputils"
	"net/http"
	"strings"
)

func NewResponse(resp *http.Response, size int64) *Response {
	r := &Response{
		Resp: resp,
	}
	if size > 0 {
		r.Content = NewContent(httputils.ReadRawWithSize(resp, size))
	} else {
		r.Content = NewContent(httputils.ReadRaw(resp))
	}

	if title := MatchTitle(r.Raw); title != "" {
		r.HasTitle = true
		r.Title = title
	} else {
		r.Title = MatchCharacter(r.Raw)
	}
	r.Server = resp.Header.Get("Server")
	if resp.TLS != nil {
		r.SSLHost = resp.TLS.PeerCertificates[0].DNSNames
	}

	if resp.Request != nil {
		for resp = resp.Request.Response; resp != nil; {
			content := NewContent(httputils.ReadRaw(resp))
			if resp.TLS != nil {
				content.SSLHost = resp.TLS.PeerCertificates[0].DNSNames
			}
			r.History = append(r.History, content)
			resp = resp.Request.Response
		}
	}

	return r
}

func NewResponseWithRaw(raw []byte) *Response {
	resp, err := http.ReadResponse(bufio.NewReader(bytes.NewReader(raw)), nil)
	if err != nil {
		return nil
	}

	return NewResponse(resp, 0)
}

type Response struct {
	Server   string     `json:"server"`
	Title    string     `json:"title"`
	HasTitle bool       `json:"-"` // html title: true , bytes[:13]: false
	History  []*Content `json:"history"`
	Resp     *http.Response
	*Content
	*Hashes `json:"hashes"`
}

func NewContent(raw []byte) *Content {
	body, header, _ := httputils.SplitHttpRaw(raw)
	return &Content{
		Body:   body,
		Header: header,
		Raw:    raw,
	}
}

type Content struct {
	Body    []byte   `json:"-"`
	Header  []byte   `json:"-"`
	Raw     []byte   `json:"raw"`
	SSLHost []string `json:"sslhsot"`
}

func (content *Content) ContentMap() map[string]interface{} {
	return map[string]interface{}{
		"content": content.Raw,
		"cert":    strings.Join(content.SSLHost, ","),
	}
}

func (r *Response) Hash() {
	r.Hashes = NewHashes(r.Raw)
}

func NewHashes(content []byte) *Hashes {
	body, header, _ := httputils.SplitHttpRaw(content)
	return &Hashes{
		BodyMd5:       encode.Md5Hash(body),
		HeaderMd5:     encode.Md5Hash(header),
		RawMd5:        encode.Md5Hash(content),
		BodySimhash:   encode.Simhash(body),
		HeaderSimhash: encode.Simhash(header),
		RawSimhash:    encode.Simhash(content),
		BodyMmh3:      encode.Mmh3Hash32(body),
	}
}

type Hashes struct {
	BodyMd5       string `json:"body-md5"`
	HeaderMd5     string `json:"header-md5"`
	RawMd5        string `json:"raw-md5"`
	BodySimhash   string `json:"body-simhash"`
	HeaderSimhash string `json:"header-simhash"`
	RawSimhash    string `json:"raw-simhash"`
	BodyMmh3      string `json:"body-mmh3"`
}

var SimhashThreshold uint8 = 8

func (hs *Hashes) Compare(other *Hashes) (uint8, uint8, uint8) {
	return encode.SimhashCompare(hs.BodySimhash, other.BodySimhash), encode.SimhashCompare(hs.HeaderSimhash, other.HeaderSimhash), encode.SimhashCompare(hs.RawSimhash, other.RawSimhash)
}
