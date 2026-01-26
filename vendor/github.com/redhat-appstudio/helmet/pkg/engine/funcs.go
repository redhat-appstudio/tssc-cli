package engine

import (
	"encoding/json"
	"errors"
	"strings"

	"gopkg.in/yaml.v3"
)

func toYAML(data interface{}) string {
	payload, err := yaml.Marshal(data)
	if err != nil {
		return ""
	}
	return strings.TrimSuffix(string(payload), "\n")
}

func fromYAML(str string) map[string]interface{} {
	m := map[string]interface{}{}
	if err := yaml.Unmarshal([]byte(str), &m); err != nil {
		m["Error"] = err.Error()
	}
	return m
}

func fromYAMLArray(str string) []interface{} {
	a := []interface{}{}
	if err := yaml.Unmarshal([]byte(str), &a); err != nil {
		a = []interface{}{err.Error()}
	}
	return a
}

func toJSON(v interface{}) string {
	data, err := json.Marshal(v)
	if err != nil {
		return ""
	}
	return string(data)
}

func fromJSON(str string) map[string]interface{} {
	m := make(map[string]interface{})
	if err := json.Unmarshal([]byte(str), &m); err != nil {
		m["Error"] = err.Error()
	}
	return m
}

func fromJSONArray(str string) []interface{} {
	a := []interface{}{}
	if err := json.Unmarshal([]byte(str), &a); err != nil {
		a = []interface{}{err.Error()}
	}
	return a
}

func required(name string, value interface{}) (interface{}, error) {
	if value == nil {
		return nil, errors.New(name + " is required")
	}
	return value, nil
}
