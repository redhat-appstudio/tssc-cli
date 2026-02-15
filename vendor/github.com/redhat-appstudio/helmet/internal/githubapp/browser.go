package githubapp

import (
	"os/exec"
	"runtime"
)

func OpenInBrowser(u string) error {
	var cmd string
	var args []string
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start", u}
	case "darwin":
		cmd = "open"
		args = []string{u}
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
		args = []string{u}
	}
	// Short-lived browser open command.
	//nolint:noctx
	return exec.Command(cmd, args...).Start()
}
