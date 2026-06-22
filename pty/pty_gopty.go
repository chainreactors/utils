//go:build !windows

package pty

import (
	"io"
	"os/exec"

	gopty "github.com/aymanbagabas/go-pty"
)

type ptyHandle struct {
	pty gopty.Pty
	cmd *gopty.Cmd
}

func startPTY(cmd *exec.Cmd) (*ptyHandle, error) {
	p, err := gopty.New()
	if err != nil {
		return nil, err
	}

	args := cmd.Args
	if len(args) == 0 {
		args = []string{cmd.Path}
	}
	c := p.Command(cmd.Path, args[1:]...)
	c.Args = args
	c.Env = cmd.Env
	c.Dir = cmd.Dir
	c.SysProcAttr = cmd.SysProcAttr
	c.Cancel = cmd.Cancel

	if err := c.Start(); err != nil {
		_ = p.Close()
		return nil, err
	}
	return &ptyHandle{pty: p, cmd: c}, nil
}

func (p *ptyHandle) Read(buf []byte) (int, error) {
	return p.pty.Read(buf)
}

func (p *ptyHandle) Write(data []byte) (int, error) {
	return p.pty.Write(data)
}

func (p *ptyHandle) Close() error {
	return p.pty.Close()
}

func (p *ptyHandle) PID() int {
	if p.cmd == nil || p.cmd.Process == nil {
		return 0
	}
	return p.cmd.Process.Pid
}

func (p *ptyHandle) Wait() error {
	if p.cmd == nil {
		return nil
	}
	return p.cmd.Wait()
}

func (p *ptyHandle) Signal(hard bool) error {
	return signalProcessGroup(p.PID(), hard)
}

func (p *ptyHandle) Resize(cols, rows int) error {
	if cols <= 0 || rows <= 0 {
		return nil
	}
	return p.pty.Resize(cols, rows)
}

// pumpOutput copies data from the PTY master into the OutputBuffer until EOF.
// Closes the returned channel when pumping is done.
func pumpOutput(r io.Reader, buf *OutputBuffer) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		tmp := make([]byte, 4096)
		for {
			n, err := r.Read(tmp)
			if n > 0 {
				_, _ = buf.Write(tmp[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	return done
}
