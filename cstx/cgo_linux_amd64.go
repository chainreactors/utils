//go:build cstx_native && linux && amd64

package cstx

// #cgo LDFLAGS: -L${SRCDIR}/lib/linux_amd64 -lcstx_ffi -lm -ldl -lpthread
import "C"
