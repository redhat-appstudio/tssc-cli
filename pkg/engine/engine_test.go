package engine

import (
	"testing"

	"github.com/otaviof/rhtap-installer-cli/pkg/config"

	o "github.com/onsi/gomega"
)

const testTemplate = `
Namespace: {{ .Installer.Namespace }}
Features:
{{- range $k, $v := .Installer.Features }} 
  {{ $k }}:
  {{- range $name, $value := $v }}
    - {{ $name }}: {{ $value }}
  {{- end }}
{{- end }}
Dependencies
{{- range $v := .Installer.Dependencies }} 
  {{- range $name, $value := $v }}
  - {{ $name }}: {{ $value }}
  {{- end }}
{{- end }}
`

func TestEngine_Render(t *testing.T) {
	g := o.NewWithT(t)

	cfg, err := config.NewConfigFromFile("../../config.yaml")
	g.Expect(err).To(o.Succeed())

	variables := NewVariables()
	err = variables.SetInstaller(&cfg.Installer)
	g.Expect(err).To(o.Succeed())

	t.Logf("Template: %s", testTemplate)

	e := NewEngine(testTemplate)
	payload, err := e.Render(variables)
	g.Expect(err).To(o.Succeed())
	g.Expect(payload).NotTo(o.BeEmpty())

	t.Logf("Output: %s", payload)
	g.Expect(payload).To(o.ContainSubstring("Namespace: "))
}
