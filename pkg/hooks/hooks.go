package hooks

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"

	"github.com/redhat-appstudio/tssc-cli/pkg/config"
)

// Hooks represent the hooks that can be executed before and after the Helm Chart
// installation, it provides the user the ability to customize the process using
// shell scripts. These scripts can rely on local tools, like "kubectl", "oc" and
// others, while the Helm Charts are only using Kubernetes resources.
// Ideally these scripts are temporary measures, and should be replaced by Helm
// Chart related resources as soon as possible.
type Hooks struct {
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
	// Extracting the script payload from the Chart instance, using the "hook"
	// directory as default location.
	scriptBytes := []byte{}
	for _, f := range h.dep.Chart.Files {
		if f.Name != path.Join("hooks", name) {
			continue
		}
		scriptBytes = f.Data
		break
	}
	// Ignoring when the script is not found, it means the given Chart does not
	// carry any hook scripts.
	if len(scriptBytes) == 0 {
		return nil
	}

	// Storing the script payload in a temporary file, adding permissions to
	// execute as a regular shell script.
	tmpFile, err := os.CreateTemp("/tmp", "tssc-hook-*.sh")
	if err != nil {
		return err
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.Write(scriptBytes); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Chmod(tmpFile.Name(), 0o755); err != nil {
		return err
	}

	return h.exec(tmpFile.Name(), vals)
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
	dep *config.Dependency,
	stdout io.Writer,
	stderr io.Writer,
) *Hooks {
	return &Hooks{
		dep:    dep,
		stdout: stdout,
		stderr: stderr,
	}
}
