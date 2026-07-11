//go:build !cstx_native

package cstx

import "fmt"

var errStub = fmt.Errorf("cstx: native FFI not compiled (build with -tags cstx_native)")

func Parse(tool string, input any) ([]SCONode, error) {
	return nil, errStub
}
