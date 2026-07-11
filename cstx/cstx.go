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
)

// Parse converts tool output into SCO-standardized nodes.
// tool is the artifact name: "gogo", "spray", "zombie", "neutron", "aiscan", etc.
// input accepts []byte (raw JSONL) or a slice of parsers result structs
// (e.g. []*parsers.GOGOResult, []*parsers.SprayResult).
func Parse(tool string, input any) ([]SCONode, error) {
	data, err := toJSONL(input)
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

func toJSONL(input any) ([]byte, error) {
	if b, ok := input.([]byte); ok {
		return b, nil
	}

	rv := reflect.ValueOf(input)
	if rv.Kind() == reflect.Ptr {
		rv = rv.Elem()
	}
	if rv.Kind() != reflect.Slice {
		b, err := json.Marshal(input)
		if err != nil {
			return nil, err
		}
		return append(b, '\n'), nil
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	for i := 0; i < rv.Len(); i++ {
		if err := enc.Encode(rv.Index(i).Interface()); err != nil {
			return nil, fmt.Errorf("cstx marshal [%d]: %w", i, err)
		}
	}
	return buf.Bytes(), nil
}
