package hooks

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/redhat-appstudio/rhtap-cli/pkg/chartfs"
	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
)

// Hooks represent the hooks that can be executed before and after the Helm Chart
// installation, it provides the user the ability to customize the process using
// shell scripts. These scripts can rely on local tools, like "kubectl", "oc" and
// others, while the Helm Charts are only using Kubernetes resources.
// Ideally these scripts are temporary measures, and should be replaced by Helm
// Chart related resources as soon as possible.
type Hooks struct {
	cfs    *chartfs.ChartFS   // chart file system
	dep    *config.Dependency // helm chart dependency
	stdout io.Writer          // standard output
	stderr io.Writer          // standard error
}

const envPrefix = "INSTALLER"

// exec executes the script with the given environment variables.
func (h *Hooks) exec(scriptPath string, vals map[string]interface{}) error {
	cmd := exec.Command(scriptPath)
	cmd.Env = os.Environ()
	// Transforming the given values into environment variables.
	for k, v := range valuesToEnv(vals, envPrefix) {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", k, v))
	}
	cmd.Stdout = h.stdout
	cmd.Stderr = h.stderr
	return cmd.Run()
}

// runHookScript executes the hook script with the given values.
func (h *Hooks) runHookScript(name string, vals map[string]interface{}) error {
	// Extracting the script payload from the Chart directory, using the "hook"
	// directory as default location.
	scriptPath := path.Join(h.dep.Chart, "hooks", name)
	scriptBytes, err := h.cfs.ReadFile(scriptPath)
	if err != nil {
		// Ignoring when the script is not found, it means the given Chart does
		// not carry any hook scripts.
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("running script %q: %w", scriptPath, err)
	}

	// Storing the script payload in a temporary file, adding permissions to
	// execute as a regular shell script.
	tmpfile, err := os.CreateTemp("/tmp", "rhtap-cli-hook-*.sh")
	if err != nil {
		return err
	}
	defer os.Remove(tmpfile.Name())
	if _, err := tmpfile.Write(scriptBytes); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpfile.Name(), 0o755); err != nil {
		return err
	}

	return h.exec(tmpfile.Name(), vals)
}

// PreDeploy executes the "pre-deploy.sh" hook script with the given values.
func (h *Hooks) PreDeploy(vals map[string]interface{}) error {
	return h.runHookScript("pre-deploy.sh", vals)
}

// PostDeploy executes the "post-deploy.sh" hook script with the given values.
func (h *Hooks) PostDeploy(vals map[string]interface{}) error {
	return h.runHookScript("post-deploy.sh", vals)
}

// NewHooks instantiates a hooks handler for the given ChartFS and Dependency.
func NewHooks(
	cfs *chartfs.ChartFS,
	dep *config.Dependency,
	stdout io.Writer,
	stderr io.Writer,
) *Hooks {
	return &Hooks{
		cfs:    cfs,
		dep:    dep,
		stdout: stdout,
		stderr: stderr,
	}
}
