package resolver

import (
	"fmt"
	"strings"
	"testing"
)

func TestCEL_Evaluate(t *testing.T) {
	c, err := NewCEL("a", "b", "c")
	if err != nil {
		t.Errorf("NewCEL() failed: %v", err)
		return
	}

	tests := []struct {
		name           string
		configured     map[string]bool
		expression     string
		wantErrContain string
	}{{
		name:           "single valid element",
		configured:     map[string]bool{"a": true},
		expression:     `a`,
		wantErrContain: "",
	}, {
		name:           "multiple valid elements",
		configured:     map[string]bool{"a": true, "b": true},
		expression:     `a && b`,
		wantErrContain: "",
	}, {
		name:           "missing element",
		configured:     map[string]bool{"a": true, "b": false},
		expression:     `a && b`,
		wantErrContain: fmt.Sprintf("%s: b", ErrMissingIntegrations),
	}, {
		name:           "unknown element",
		configured:     map[string]bool{},
		expression:     `d`,
		wantErrContain: ErrInvalidExpression.Error(),
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotErr := c.Evaluate(tt.configured, tt.expression)
			if gotErr != nil {
				if tt.wantErrContain == "" {
					t.Errorf("Evaluate() failed: %v", gotErr)
				}
				if !strings.Contains(gotErr.Error(), tt.wantErrContain) {
					t.Errorf("Evaluate() error %q does not contain expected: %q",
						gotErr, tt.wantErrContain)
				}
				return
			}
			if tt.wantErrContain != "" {
				t.Fatal("Evaluate() succeeded unexpectedly")
			}
		})
	}
}
