package installer

import (
	"github.com/redhat-appstudio/rhtap-cli/pkg/config"
)

type Installer struct {
	cfg *config.Spec // installer configuration
}

func (i *Installer) Install() error {
	return nil
}

func NewInstaller() *Installer {
	return &Installer{}
}
