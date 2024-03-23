package engine

import (
	"bytes"
	"html/template"

	"github.com/Masterminds/sprig/v3"
)

// Engine represents the template engine.
type Engine struct {
	templatePayload string // template payload
}

// Render renders the template with the given variables.
func (e *Engine) Render(variables *Variables) ([]byte, error) {
	funcMap := sprig.TxtFuncMap()

	tmpl, err := template.New("values.yaml.tpl").
		Funcs(funcMap).
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
func NewEngine(templatePayload string) *Engine {
	return &Engine{templatePayload: templatePayload}
}
