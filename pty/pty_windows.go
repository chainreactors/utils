//go:build windows

package pty

import (
	"io"
	"os/exec"

	gopty "github.com/aymanbagabas/go-pty"
	"golang.org/x/sys/windows"
)

type ptyHandle struct {
	pty    gopty.Pty
	gcmd   *gopty.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	cmd    *exec.Cmd
}

func startPTY(cmd *exec.Cmd) (*ptyHandle, error) {
	if conPTYAvailable() {
		if p, err := startGoPTY(cmd); err == nil {
			return p, nil
		}
	}
	return startPipePTY(cmd)
}

func startGoPTY(cmd *exec.Cmd) (*ptyHandle, error) {
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
	return &ptyHandle{pty: p, gcmd: c}, nil
}

func startPipePTY(cmd *exec.Cmd) (*ptyHandle, error) {
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return nil, err
	}
	cmd.Stderr = cmd.Stdout
	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		_ = stdout.Close()
		return nil, err
	}
	return &ptyHandle{stdin: stdin, stdout: stdout, cmd: cmd}, nil
}

func (p *ptyHandle) Read(buf []byte) (int, error) {
	if p.pty != nil {
		return p.pty.Read(buf)
	}
	return p.stdout.Read(buf)
}

func (p *ptyHandle) Write(data []byte) (int, error) {
	if p.pty != nil {
		return p.pty.Write(data)
	}
	return p.stdin.Write(data)
}

func (p *ptyHandle) Close() error {
	if p.pty != nil {
		return p.pty.Close()
	}
	_ = p.stdin.Close()
	return p.stdout.Close()
}

func (p *ptyHandle) PID() int {
	switch {
	case p.gcmd != nil && p.gcmd.Process != nil:
		return p.gcmd.Process.Pid
	case p.cmd != nil && p.cmd.Process != nil:
		return p.cmd.Process.Pid
	default:
		return 0
	}
}

func (p *ptyHandle) Wait() error {
	switch {
	case p.gcmd != nil:
		return p.gcmd.Wait()
	case p.cmd != nil:
		return p.cmd.Wait()
	default:
		return nil
	}
}

func (p *ptyHandle) Signal(hard bool) error {
	return signalProcessGroup(p.PID(), hard)
}

func (p *ptyHandle) Resize(cols, rows int) error {
	if p.pty == nil || cols <= 0 || rows <= 0 {
		return nil
	}
	return p.pty.Resize(cols, rows)
}

func conPTYAvailable() bool {
	kernel32 := windows.NewLazySystemDLL("kernel32.dll")
	for _, name := range []string{"CreatePseudoConsole", "ClosePseudoConsole", "ResizePseudoConsole"} {
		if err := kernel32.NewProc(name).Find(); err != nil {
			return false
		}
	}
	return true
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
