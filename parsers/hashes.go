package parsers

import (
	"bytes"

	"github.com/chainreactors/utils/encode"
)

func SplitHttpRaw(content []byte) (body, header []byte, ok bool) {
	cs := bytes.Index(content, []byte("\r\n\r\n"))
	if cs != -1 && len(content) >= cs+4 {
		return content[cs+4:], content[:cs], true
	}
	return nil, nil, false
}

func NewHashes(content []byte) *Hashes {
	body, header, _ := SplitHttpRaw(content)
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
