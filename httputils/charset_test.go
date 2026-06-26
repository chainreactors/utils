package httputils

import (
	"testing"

	"golang.org/x/text/encoding/simplifiedchinese"
)

func TestDecodeCharsetGBK(t *testing.T) {
	utf8Text := "电子文档安全管理系统"
	gbkBytes, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(utf8Text))
	if err != nil {
		t.Fatal(err)
	}

	for _, ct := range []string{
		"text/html; charset=gbk",
		"text/html; charset=GB2312",
		"text/html; charset=gb18030",
		"text/html;charset=GBK",
	} {
		got := DecodeCharset(gbkBytes, ct)
		if string(got) != utf8Text {
			t.Errorf("DecodeCharset(%q): got %q, want %q", ct, got, utf8Text)
		}
	}
}

func TestDecodeCharsetUTF8Passthrough(t *testing.T) {
	body := []byte("hello world 你好")
	for _, ct := range []string{
		"text/html; charset=utf-8",
		"text/html; charset=UTF-8",
		"text/html",
		"",
	} {
		got := DecodeCharset(body, ct)
		if string(got) != string(body) {
			t.Errorf("DecodeCharset(%q): body was modified", ct)
		}
	}
}

func TestAutoDecodeBodyFallbackMeta(t *testing.T) {
	utf8Text := "测试"
	gbkBytes, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(utf8Text))
	if err != nil {
		t.Fatal(err)
	}

	html := []byte(`<html><head><meta charset="gbk"></head><body>`)
	html = append(html, gbkBytes...)
	html = append(html, []byte(`</body></html>`)...)

	got := AutoDecodeBody(html, "text/html")
	if string(got) == string(html) {
		t.Error("AutoDecodeBody did not decode GBK from meta tag")
	}
}

func TestDecodeCharsetShiftJIS(t *testing.T) {
	got := DecodeCharset([]byte{0x82, 0xb1, 0x82, 0xf1, 0x82, 0xc9, 0x82, 0xbf, 0x82, 0xcd}, "text/html; charset=shift_jis")
	if string(got) != "こんにちは" {
		t.Errorf("Shift_JIS decode: got %q", got)
	}
}

func TestAutoDecodeRawFromHeader(t *testing.T) {
	utf8Text := "电子文档安全管理系统"
	gbkBody, err := simplifiedchinese.GBK.NewEncoder().Bytes([]byte(utf8Text))
	if err != nil {
		t.Fatal(err)
	}

	raw := append([]byte("HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=gbk\r\n\r\n"), gbkBody...)
	got := AutoDecodeRaw(raw)
	if string(got) == string(raw) {
		t.Error("AutoDecodeRaw did not decode GBK from header")
	}
}
