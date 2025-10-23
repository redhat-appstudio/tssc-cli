package resolver

import (
	"errors"
	"fmt"
	"strings"

	"github.com/google/cel-go/cel"
)

// CEL represents the CEL environment with provided integration names, the
// integrations present in the cluster are represented by a map of integration
// name and boolean, indicating the integration is configured in the cluster.
type CEL struct {
	env *cel.Env // all known integrations names
}

var (
	// ErrInvalidExpression the expression is not a valid CEL expression.
	ErrInvalidExpression = errors.New("invalid CEL expression")
	// ErrMissingIntegrations one or more integrations aren't configured.
	ErrMissingIntegrations = errors.New("missing integrations")
)

// Evaluate evaluates the provided CEL expression against the current context of
// integration names and a boolean indicating whether it's configured.
func (c *CEL) Evaluate(configured map[string]bool, expression string) error {
	// Instantiaging the AST with the informed expression, and checking for
	// expression issues.
	ast, issues := c.env.Compile(expression)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("%w: %q", ErrInvalidExpression, expression)
	}

	// Generating a checked AST, where the types are validated, this allows
	// extracing the actual integration names referenced in the expression.
	checkedAST, issues := c.env.Check(ast)
	if issues != nil && issues.Err() != nil {
		return fmt.Errorf("%w: %q", ErrInvalidExpression, expression)
	}
	referenced := []string{}
	for _, ref := range checkedAST.NativeRep().ReferenceMap() {
		if ref.Name != "" {
			referenced = append(referenced, ref.Name)
		}
	}

	// Generating the program from the AST, and evaluating it against the context
	// based on the configured integrations.
	prg, err := c.env.Program(ast)
	if err != nil {
		return fmt.Errorf("%w: %q fails to compile: %s",
			ErrInvalidExpression, expression, err)
	}
	evalContext := make(map[string]any, len(configured))
	for k, v := range configured {
		evalContext[k] = v
	}
	result, _, err := prg.Eval(evalContext)
	if err != nil {
		return err
	}

	// All expressions must evaluate to true, meaning all required integrations
	// are configured.
	if result.Value() == true {
		return nil
	}

	// Using the referenced integration names to determine which integrations are
	// missing, as in should be configured in the cluster but aren't found.
	missing := []string{}
	for _, ref := range referenced {
		if !configured[ref] {
			missing = append(missing, ref)
		}
	}
	return fmt.Errorf("%w: %s",
		ErrMissingIntegrations, strings.Join(missing, ", "))
}

// NewCEL creates a new CEL instance with the all valid integration names. These
// names are considered variables in the CEL expression, limiting the scope of the
// expression to only valid integrations.
func NewCEL(integrationNames ...string) (*CEL, error) {
	// Registering all integration names as options, boolean variables.
	options := []cel.EnvOption{}
	for _, option := range integrationNames {
		options = append(options, cel.Variable(option, cel.BoolType))
	}
	// Creating a CEL environment using the integration names as options.
	env, err := cel.NewEnv(options...)
	if err != nil {
		return nil, err
	}
	return &CEL{env: env}, nil
}
