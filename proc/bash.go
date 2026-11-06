package proc

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// BashCommand executes the Bash language on every host. Unlike ShellCommand,
// it never substitutes cmd.exe or a WSL launcher for a native Bash runtime.
// Interactive terminals can still use DefaultShellCommand independently.
func BashCommand(line string) (*exec.Cmd, error) {
	name := "bash"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if !filepath.IsAbs(dir) {
			continue
		}
		candidate := filepath.Join(dir, name)
		if runtime.GOOS == "windows" {
			root := os.Getenv("SystemRoot")
			if root != "" && strings.EqualFold(filepath.Clean(dir), filepath.Join(root, "System32")) {
				continue // This bash.exe launches a different OS and filesystem.
			}
		}
		if path, err := exec.LookPath(candidate); err == nil {
			return exec.Command(path, "--noprofile", "--norc", "-c", line), nil
		}
	}
	return nil, fmt.Errorf("native Bash is required in PATH (Windows: Git Bash or MSYS2; WSL bash.exe is not a native runtime)")
}
