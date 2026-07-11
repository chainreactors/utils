//go:build cstx_native

package cstx

/*
#include "cstx_ffi.h"
#include <stdlib.h>
*/
import "C"
import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"unsafe"

	"github.com/chainreactors/utils/parsers"
)


// Parse converts tool output into SCO-standardized nodes.
// tool is the artifact name: "gogo", "spray", "zombie", "neutron", "aiscan", etc.
// input accepts []byte (raw JSONL) or a slice of parsers result structs
// (e.g. []*parsers.GOGOResult, []*parsers.SprayResult).
func Parse(tool string, input any) ([]SCONode, error) {
	data, err := toJSONL(tool, input)
	if err != nil {
		return nil, err
	}
	raw, err := transform(tool, data)
	if err != nil {
		return nil, err
	}
	return parseSCONodes(raw)
}

func parseSCONodes(data []byte) ([]SCONode, error) {
	var rawNodes []json.RawMessage
	if err := json.Unmarshal(data, &rawNodes); err != nil {
		return nil, fmt.Errorf("cstx parse: %w", err)
	}
	nodes := make([]SCONode, 0, len(rawNodes))
	for _, r := range rawNodes {
		n, err := ParseSCONode(r)
		if err != nil {
			continue
		}
		if n != nil {
			nodes = append(nodes, n)
		}
	}
	return nodes, nil
}

func transform(tool string, data []byte) ([]byte, error) {
	cs := C.CString(tool)
	defer C.free(unsafe.Pointer(cs))

	var out *C.char
	var outLen C.size_t
	var dp *C.uchar
	if len(data) > 0 {
		dp = (*C.uchar)(unsafe.Pointer(&data[0]))
	}
	rc := C.cstx_transform(cs, dp, C.size_t(len(data)), &out, &outLen)
	if out != nil {
		defer C.cstx_free_string(out)
	}
	if rc != 0 {
		if out != nil {
			return nil, fmt.Errorf("cstx transform: %s", C.GoStringN(out, C.int(outLen)))
		}
		return nil, fmt.Errorf("cstx transform failed")
	}
	return C.GoBytes(unsafe.Pointer(out), C.int(outLen)), nil
}

func toJSONL(tool string, input any) ([]byte, error) {
	if b, ok := input.([]byte); ok {
		return b, nil
	}

	rv := reflect.ValueOf(input)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice {
		return marshalSingle(tool, input)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for i := 0; i < rv.Len(); i++ {
		elem := rv.Index(i).Interface()
		adapted := adaptForCSTX(tool, elem)
		if err := enc.Encode(adapted); err != nil {
			return nil, fmt.Errorf("cstx marshal [%d]: %w", i, err)
		}
	}
	return buf.Bytes(), nil
}

func marshalSingle(tool string, v any) ([]byte, error) {
	adapted := adaptForCSTX(tool, v)
	b, err := json.Marshal(adapted)
	if err != nil {
		return nil, err
	}
	b = append(b, '\n')
	return b, nil
}

// adaptForCSTX converts Go parsers types to CSTX-compatible JSON.
// Frameworks (map[string]*Framework) → []*Framework (array)
// Vulns (map[string]*Vuln) → []*Vuln (array)
func adaptForCSTX(tool string, v any) any {
	switch tool {
	case "gogo", "aiscan":
		return adaptGogo(v)
	case "spray":
		return adaptSpray(v)
	default:
		return v
	}
}

type gogoLine struct {
	IP         string              `json:"ip"`
	Port       string              `json:"port"`
	Protocol   string              `json:"protocol"`
	Status     string              `json:"status,omitempty"`
	Uri        string              `json:"uri,omitempty"`
	Host       string              `json:"host,omitempty"`
	Title      string              `json:"title,omitempty"`
	Midware    string              `json:"midware,omitempty"`
	Frameworks []*parsers.Framework `json:"frameworks,omitempty"`
	Vulns      []*parsers.Vuln      `json:"vulns,omitempty"`
	Extracteds map[string][]string  `json:"extracted,omitempty"`
}

func adaptGogo(v any) any {
	var r *parsers.GOGOResult
	switch x := v.(type) {
	case *parsers.GOGOResult:
		r = x
	case parsers.GOGOResult:
		r = &x
	default:
		return v
	}
	return &gogoLine{
		IP: r.Ip, Port: r.Port, Protocol: r.Protocol,
		Status: r.Status, Uri: r.Uri, Host: r.Host,
		Title: r.Title, Midware: r.Midware,
		Frameworks: mapToSlice(r.Frameworks),
		Vulns:      vulnMapToSlice(r.Vulns),
		Extracteds: r.Extracteds,
	}
}

type sprayLine struct {
	URL          string              `json:"url"`
	Title        string              `json:"title,omitempty"`
	Status       int                 `json:"status"`
	Host         string              `json:"host,omitempty"`
	Path         string              `json:"path,omitempty"`
	BodyLength   int                 `json:"body_length,omitempty"`
	HeaderLength int                 `json:"header_length,omitempty"`
	RedirectURL  string              `json:"redirect_url,omitempty"`
	ContentType  string              `json:"content_type,omitempty"`
	Frameworks   []*parsers.Framework `json:"frameworks,omitempty"`
}

func adaptSpray(v any) any {
	var r *parsers.SprayResult
	switch x := v.(type) {
	case *parsers.SprayResult:
		r = x
	case parsers.SprayResult:
		r = &x
	default:
		return v
	}
	return &sprayLine{
		URL: r.UrlString, Title: r.Title, Status: r.Status,
		Host: r.Host, Path: r.Path,
		BodyLength: r.BodyLength, HeaderLength: r.HeaderLength,
		RedirectURL: r.RedirectURL, ContentType: r.ContentType,
		Frameworks: mapToSlice(r.Frameworks),
	}
}

func mapToSlice(m parsers.Frameworks) []*parsers.Framework {
	if len(m) == 0 {
		return nil
	}
	s := make([]*parsers.Framework, 0, len(m))
	for _, f := range m {
		s = append(s, f)
	}
	return s
}

func vulnMapToSlice(m parsers.Vulns) []*parsers.Vuln {
	if len(m) == 0 {
		return nil
	}
	s := make([]*parsers.Vuln, 0, len(m))
	for _, v := range m {
		s = append(s, v)
	}
	return s
}
