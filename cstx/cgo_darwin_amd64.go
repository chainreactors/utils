//go:build cstx_native && darwin && amd64

package cstx

// #cgo LDFLAGS: -L${SRCDIR}/lib/darwin_amd64 -lcstx_ffi -lm -framework Security -framework CoreFoundation
import "C"
