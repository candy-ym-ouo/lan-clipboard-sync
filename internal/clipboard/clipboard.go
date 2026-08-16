// Package clipboard supplies the small OS integration layer used by the daemon.
package clipboard

import (
	"errors"
	"os/exec"
	"runtime"
	"strings"
)

// Read returns the current system clipboard text on supported platforms.
func Read() (string, error) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbpaste")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard", "-o")
	default:
		return "", errors.New("clipboard reading is unsupported on this platform")
	}
	b, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Write replaces the system clipboard text on supported platforms.
func Write(text string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("pbcopy")
	case "linux":
		cmd = exec.Command("xclip", "-selection", "clipboard")
	default:
		return errors.New("clipboard writing is unsupported on this platform")
	}
	cmd.Stdin = strings.NewReader(text)
	return cmd.Run()
}
