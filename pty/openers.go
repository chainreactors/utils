package pty

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const DefaultSessionTimeout = 24 * time.Hour

func DefaultEnv() []string {
	return []string{"TERM=xterm-256color", "COLORTERM=truecolor"}
}

func DefaultOpeners(mgr *Manager, timeout time.Duration, env []string) map[string]OpenFunc {
	if timeout <= 0 {
		timeout = DefaultSessionTimeout
	}
	if env == nil {
		env = DefaultEnv()
	}
	return map[string]OpenFunc{
		"shell":   ShellOpener(mgr, timeout, env),
		"command": CommandOpener(mgr, timeout, env),
	}
}

func ShellOpener(mgr *Manager, timeout time.Duration, env []string) OpenFunc {
	return func(_ context.Context, spec OpenSpec) (OpenResult, error) {
		if mgr == nil {
			return OpenResult{}, fmt.Errorf("pty manager unavailable")
		}
		binary, args := DefaultShellCommand()
		info, err := mgr.CreateCmdRaw("", binary, args, spec.Name, timeout, env, "")
		if err != nil {
			return OpenResult{}, err
		}
		mgr.SetKind(info.ID, "shell")
		info.Kind = "shell"
		return OpenResult{Info: info}, nil
	}
}

func CommandOpener(mgr *Manager, timeout time.Duration, env []string) OpenFunc {
	return func(_ context.Context, spec OpenSpec) (OpenResult, error) {
		if mgr == nil {
			return OpenResult{}, fmt.Errorf("pty manager unavailable")
		}
		command := strings.TrimSpace(spec.Command)
		if command == "" {
			return OpenResult{}, fmt.Errorf("pty command required")
		}
		info, err := mgr.CreateRaw("", command, spec.Name, timeout, env, "")
		if err != nil {
			return OpenResult{}, err
		}
		mgr.SetKind(info.ID, "command")
		info.Kind = "command"
		return OpenResult{Info: info}, nil
	}
}
