package engine

import (
	"testing"

	"github.com/redhat-appstudio/rhtap-cli/pkg/config"

	o "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
)

// testYamlTmpl is a template to render a YAML payload based on the information
// available for the installer template engine.
const testYamlTmpl = `
---
root:
  namespace: {{ .Installer.Namespace }} 
  features:
{{- range $k, $v := .Installer.Features }}
  {{- $k | nindent 4 }}:
  {{- $v | toYaml | nindent 6 }}
{{- end }}
  dependencies:
	{{- .Installer.Dependencies | toYaml | nindent 4 }}
  catalogURL: {{ .Installer.Features.redHatDeveloperHub.Properties.catalogURL }}
`

func TestEngine_Render(t *testing.T) {
	g := o.NewWithT(t)

	cfg, err := config.NewConfigFromFile("../../installer/config.yaml")
	g.Expect(err).To(o.Succeed())

	variables := NewVariables()
	err = variables.SetInstaller(&cfg.Installer)
	g.Expect(err).To(o.Succeed())

	t.Logf("Template: %s", testYamlTmpl)

	e := NewEngine(nil, testYamlTmpl)
	payload, err := e.Render(variables)
	g.Expect(err).To(o.Succeed())
	g.Expect(payload).NotTo(o.BeEmpty())

	t.Logf("Output: %s", payload)

	// Unmarshal the rendered payload to check the actual structure of the YAML
	// file created with the template engine.
	var outputMap map[string]interface{}
	err = yaml.Unmarshal([]byte(payload), &outputMap)
	g.Expect(err).To(o.Succeed())
	g.Expect(outputMap).NotTo(o.BeEmpty())

	g.Expect(outputMap).To(o.HaveKey("root"))
	g.Expect(outputMap["root"]).NotTo(o.BeNil())

	root := outputMap["root"].(map[string]interface{})
	g.Expect(root).To(o.HaveKey("namespace"))
	g.Expect(root["namespace"]).To(o.Equal(cfg.Installer.Namespace))

	g.Expect(root).To(o.HaveKey("features"))
	g.Expect(root["features"]).NotTo(o.BeNil())

	g.Expect(root).To(o.HaveKey("dependencies"))
	g.Expect(root["dependencies"]).NotTo(o.BeNil())

	g.Expect(root).To(o.HaveKey("catalogURL"))
	g.Expect(root["catalogURL"]).NotTo(o.BeNil())

	feature, err := cfg.GetFeature(config.RedHatDeveloperHub)
	g.Expect(err).To(o.Succeed())
	g.Expect(root["catalogURL"]).To(o.Equal(feature.Properties["catalogURL"]))
}
