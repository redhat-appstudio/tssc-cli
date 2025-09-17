package integration

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrInvalidURL is an error returned when a URL is invalid, malformed.
var ErrInvalidURL = errors.New("invalid URL")

// ErrInvalidJSON is an error returned when a string is not a valid JSON.
var ErrInvalidJSON = errors.New("invalid JSON")

// ErrJSONContainsSpaces is an error returned when a JSON key or value contains spaces.
var ErrJSONContainsSpaces = errors.New("contains unexpected spaces")

// ValidateURL check if the informed URL is valid.
func ValidateURL(location string) error {
	u, err := url.Parse(location)
	if err != nil {
		return fmt.Errorf("%w: invalid url %q: %s", ErrInvalidURL, location, err)
	}
	if !strings.HasPrefix(u.Scheme, "http") {
		return fmt.Errorf("%w: invalid scheme %q, expected http or https",
			ErrInvalidURL, location)
	}
	return nil
}

// ValidateJSON checks if the given string is a valid JSON and that there
// is no space character in any of the keys and values of a JSON object.
func ValidateJSON(p string, s string) error {
	var data interface{}

	if err := json.Unmarshal([]byte(s), &data); err != nil {
		return fmt.Errorf("%w in --%s: %s", ErrInvalidJSON, p, err)
	}

	return checkJSONSpaces(p, data)
}

func checkJSONSpaces(p string, data interface{}) error {
	switch v := data.(type) {
	case string:
		if strings.Contains(v, " ") {
			return fmt.Errorf("--%s %w in string %q", p, ErrJSONContainsSpaces, v)
		}
	case map[string]interface{}:
		for key, val := range v {
			if err := checkJSONSpaces(p, key); err != nil {
				return err
			}
			if err := checkJSONSpaces(p, val); err != nil {
				return err
			}
		}
	case []interface{}:
		for _, val := range v {
			if err := checkJSONSpaces(p, val); err != nil {
				return err
			}
		}
	}
	return nil
}
