//go:build windows

package pty

func processAlive(int) bool {
	return true
}
