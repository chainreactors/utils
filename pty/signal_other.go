//go:build !unix && !windows

package pty

func signalProcessGroup(_ int, _ bool) error {
	return nil
}
