package hooks

import (
	"fmt"
	"os"
	"os/exec"
	"path"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
	"helm.sh/helm/v3/pkg/chartutil"
)

type Hooks struct {
	dep config.Dependency
}

func (h *Hooks) runHookScript(name string, vals chartutil.Values) error {
	scriptPath := path.Join(h.dep.Chart, "hooks", name)
	_, err := os.Stat(scriptPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	cmd := exec.Command(scriptPath)
	cmd.Env = os.Environ()
	for k, v := range vals {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%v", k, v))
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

func (h *Hooks) PreDeploy(vals chartutil.Values) error {
	return h.runHookScript("pre-deploy.sh", vals)
}

func (h *Hooks) PostDeploy(vals chartutil.Values) error {
	return h.runHookScript("post-deploy.sh", vals)
}

func NewHooks(dep config.Dependency) *Hooks {
	return &Hooks{dep: dep}
}
