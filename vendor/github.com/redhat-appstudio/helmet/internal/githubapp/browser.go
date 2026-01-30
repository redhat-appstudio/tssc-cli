package githubapp

import (
	"os/exec"
	"runtime"
)

func OpenInBrowser(u string) error {
	cmd := ""
	args := []string{}
	switch runtime.GOOS {
	case "windows":
		cmd = "cmd"
		args = []string{"/c", "start"}
	case "darwin":
		cmd = "open"
	default: // "linux", "freebsd", "openbsd", "netbsd"
		cmd = "xdg-open"
	}
	args = append(args, u)
	return exec.Command(cmd, args...).Start()
}
