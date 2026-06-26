package httputils

import (
	"mime"
	"regexp"
	"strings"

	"golang.org/x/text/encoding/htmlindex"
)

var (
	headerCharsetRE = regexp.MustCompile(`(?i)Content-Type:.*?charset=([^\s;]+)`)
	metaCharsetRE   = regexp.MustCompile(`(?i)<meta[^>]*?charset=["']?\s*([^\s"'>;]+)`)
)

// DecodeCharset decodes body from the charset declared in contentType to UTF-8.
// Returns the original body unchanged if the charset is utf-8, empty, or unknown.
func DecodeCharset(body []byte, contentType string) []byte {
	charset := charsetFromContentType(contentType)
	if charset == "" {
		return body
	}
	return decodeToUTF8(body, charset)
}

// AutoDecodeBody detects charset from contentType first, then falls back to
// scanning the body for an HTML <meta charset="..."> tag. Returns UTF-8 bytes.
func AutoDecodeBody(body []byte, contentType string) []byte {
	charset := charsetFromContentType(contentType)
	if charset == "" {
		charset = charsetFromMeta(body)
	}
	if charset == "" {
		return body
	}
	return decodeToUTF8(body, charset)
}

// AutoDecodeRaw detects charset from raw HTTP data (header + body) by scanning
// the Content-Type header line and HTML <meta> tags, then decodes to UTF-8.
func AutoDecodeRaw(raw []byte) []byte {
	charset := charsetFromHeader(raw)
	if charset == "" {
		charset = charsetFromMeta(raw)
	}
	if charset == "" {
		return raw
	}
	return decodeToUTF8(raw, charset)
}

func charsetFromContentType(contentType string) string {
	if contentType == "" {
		return ""
	}
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return ""
	}
	return normalizeCharset(params["charset"])
}

func charsetFromHeader(raw []byte) string {
	m := headerCharsetRE.FindSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return normalizeCharset(string(m[1]))
}

func charsetFromMeta(body []byte) string {
	m := metaCharsetRE.FindSubmatch(body)
	if len(m) < 2 {
		return ""
	}
	return normalizeCharset(string(m[1]))
}

func normalizeCharset(charset string) string {
	charset = strings.TrimSpace(strings.ToLower(charset))
	if charset == "" || charset == "utf-8" || charset == "utf8" {
		return ""
	}
	return charset
}

func decodeToUTF8(body []byte, charset string) []byte {
	enc, err := htmlindex.Get(charset)
	if err != nil || enc == nil {
		return body
	}
	decoded, err := enc.NewDecoder().Bytes(body)
	if err != nil {
		return body
	}
	return decoded
}
