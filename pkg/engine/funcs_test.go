package engine

import (
	"reflect"
	"strings"
	"testing"
)

func TestToYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{{
		name:     "empty input",
		input:    nil,
		expected: "null",
	}, {
		name: "valid input",
		input: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
		expected: "key1: value1\nkey2: 123",
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toYAML(tt.input)
			if result != tt.expected {
				t.Errorf("toYAML() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFromYAML(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
		wantErr  bool
	}{{
		name:     "empty input",
		input:    "",
		expected: map[string]interface{}{},
		wantErr:  false,
	}, {
		name:  "valid input",
		input: "key1: value1\nkey2: 123",
		expected: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
		wantErr: false,
	}, {
		name:    "invalid input",
		input:   `invalid yaml`,
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fromYAML(tt.input)
			if tt.wantErr {
				if _, ok := result["Error"]; !ok {
					t.Errorf("fromYAML() = %v, expected error", result)
				}
			} else if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("fromYAML() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFromYAMLArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []interface{}
		wantErr  bool
	}{{
		name:     "empty input",
		input:    "",
		expected: []interface{}{},
		wantErr:  false,
	}, {
		name:     "valid input",
		input:    "- value1\n- 123\n- true",
		expected: []interface{}{"value1", 123, true},
		wantErr:  false,
	}, {
		name:    "invalid input",
		input:   "invalid yaml",
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fromYAMLArray(tt.input)
			if tt.wantErr {
				if len(result) != 1 ||
					!strings.Contains(result[0].(string), "error") {
					t.Errorf("fromYAMLArray() = %v, expected error", result)
				}
			} else if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("fromYAMLArray() = %v, expected %v",
					result, tt.expected)
			}
		})
	}
}

func TestToJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{{
		name:     "empty input",
		input:    nil,
		expected: "null",
	}, {
		name: "valid input",
		input: map[string]interface{}{
			"key1": "value1",
			"key2": 123,
		},
		expected: `{"key1":"value1","key2":123}`,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toJSON(tt.input)
			if result != tt.expected {
				t.Errorf("toJSON() = %q, expected %q", result, tt.expected)
			}
		})
	}
}

func TestFromJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected map[string]interface{}
		wantErr  bool
	}{{
		name:     "empty input",
		input:    "{}",
		expected: map[string]interface{}{},
		wantErr:  false,
	}, {
		name:     "valid input",
		input:    `{"key1":"value1"}`,
		expected: map[string]interface{}{"key1": "value1"},
		wantErr:  false,
	}, {
		name:    "invalid input",
		input:   `invalid json`,
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fromJSON(tt.input)
			if tt.wantErr {
				if _, ok := result["Error"]; !ok {
					t.Errorf("fromJSON() = %v, expected error", result)
				}
			} else if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("fromJSON() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFromJSONArray(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []interface{}
		wantErr  bool
	}{{
		name:     "empty input",
		input:    "[]",
		expected: []interface{}{},
		wantErr:  false,
	}, {
		name:     "valid input",
		input:    `["value1", "value2"]`,
		expected: []interface{}{"value1", "value2"},
		wantErr:  false,
	}, {
		name:    "invalid input",
		input:   "invalid json",
		wantErr: true,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := fromJSONArray(tt.input)
			if tt.wantErr {
				if len(result) != 1 ||
					!strings.Contains(result[0].(string), "invalid") {
					t.Errorf("fromJSONArray() = %v, expected error", result)
				}
			} else if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("fromJSONArray() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestRequired(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected interface{}
		wantErr  bool
	}{{
		name:     "nil value",
		input:    nil,
		expected: nil,
		wantErr:  true,
	}, {
		name:     "non-nil value",
		input:    "value",
		expected: "value",
		wantErr:  false,
	}}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := required(tt.name, tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("required() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(result, tt.expected) {
				t.Errorf("required() result = %v, expected %v",
					result, tt.expected)
			}
		})
	}
}
