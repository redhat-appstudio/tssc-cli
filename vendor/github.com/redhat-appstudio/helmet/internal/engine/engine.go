package engine

import (
	"bytes"
	"html/template"

	"github.com/redhat-appstudio/helmet/internal/constants"
	"github.com/redhat-appstudio/helmet/internal/k8s"

	"github.com/Masterminds/sprig/v3"
)

// Engine represents the template engine.
type Engine struct {
	funcMap         template.FuncMap // template functions
	templatePayload string           // template payload
}

// Render renders the template with the given variables.
func (e *Engine) Render(variables *Variables) ([]byte, error) {
	tmpl, err := template.New(constants.ValuesFilename).
		Funcs(e.funcMap).
		Parse(e.templatePayload)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err = tmpl.Execute(&buf, variables); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// NewEngine instantiates the template engine.
func NewEngine(kube *k8s.Kube, templatePayload string) *Engine {
	funcMap := sprig.TxtFuncMap()

	funcMap["toYaml"] = toYAML
	funcMap["fromYaml"] = fromYAML
	funcMap["fromYamlArray"] = fromYAMLArray

	funcMap["toJson"] = toJSON
	funcMap["fromJson"] = fromJSON
	funcMap["fromJsonArray"] = fromJSONArray

	funcMap["required"] = required

	l := NewLookupFuncs(kube)
	funcMap["lookup"] = l.Lookup()

	return &Engine{
		templatePayload: templatePayload,
		funcMap:         funcMap,
	}
}
