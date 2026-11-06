package proc

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

func TestShellCommandWithoutBashInPath(t *testing.T) {
	path := t.TempDir()
	if runtime.GOOS == "windows" {
		path = filepath.Join(os.Getenv("SystemRoot"), "System32")
	}
	t.Setenv("PATH", path)
	for _, cmd := range []*exec.Cmd{ShellCommand("echo shell-ok"), ShellCommandContext(t.Context(), "echo shell-ok")} {
		output, err := cmd.CombinedOutput()
		if err != nil || !strings.Contains(string(output), "shell-ok") {
			t.Fatalf("platform shell failed: %v, output=%q", err, output)
		}
	}
	if binary, _ := DefaultShellCommand(); binary == "" {
		t.Fatal("interactive shell is unavailable")
	}
}

func TestShellCommandContextCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	if err := ShellCommandContext(ctx, "echo canceled").Run(); !errors.Is(err, context.Canceled) {
		t.Fatalf("canceled command returned %v", err)
	}
}

func TestBashCommandRequiresNativeBashInPath(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "bin")
	if err := os.Mkdir(dir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	if cmd, err := BashCommand("echo bash-ok"); cmd != nil || err == nil {
		t.Fatalf("missing Bash returned cmd=%v, err=%v", cmd, err)
	}
	name := "bash"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, nil, 0755); err != nil {
		t.Fatal(err)
	}
	cmd, err := BashCommand("echo bash-ok")
	if err != nil {
		t.Fatal(err)
	}
	if cmd.Path != path || !slices.Equal(cmd.Args[1:], []string{"--noprofile", "--norc", "-c", "echo bash-ok"}) {
		t.Fatalf("unexpected Bash command: %+v", cmd)
	}
	if runtime.GOOS == "windows" {
		t.Setenv("SystemRoot", root)
		system32 := filepath.Join(root, "System32")
		if err := os.MkdirAll(system32, 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(system32, name), nil, 0755); err != nil {
			t.Fatal(err)
		}
		t.Setenv("PATH", system32)
		if cmd, err := BashCommand("echo bash-ok"); cmd != nil || err == nil {
			t.Fatalf("WSL launcher accepted as native Bash: cmd=%v, err=%v", cmd, err)
		}
	}
}
