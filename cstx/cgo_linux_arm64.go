//go:build cstx_native && linux && arm64

package cstx

// #cgo LDFLAGS: -L${SRCDIR}/lib/linux_arm64 -lcstx_ffi -lm -ldl -lpthread
import "C"
