package integration

import (
	"errors"
	"testing"
)

func TestValidateURL(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		location    string
		expectedErr error
	}{
		{
			name:        "Valid HTTP URL",
			location:    "http://example.com",
			expectedErr: nil,
		},
		{
			name:        "Valid HTTPS URL",
			location:    "https://example.com",
			expectedErr: nil,
		},
		{
			name:        "Invalid URL, no scheme",
			location:    "example.com",
			expectedErr: ErrInvalidURL,
		},
		{
			name:        "Invalid URL, unparseable",
			location:    "://example.com",
			expectedErr: ErrInvalidURL,
		},
		{
			name:        "Invalid scheme",
			location:    "ftp://example.com",
			expectedErr: ErrInvalidURL,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateURL(tc.location)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected err %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestValidateJSON(t *testing.T) {
	t.Parallel()
	testCases := []struct {
		name        string
		param       string
		json        string
		expectedErr error
	}{
		{
			name:        "Valid JSON",
			param:       "test",
			json:        `{"key":"value"}`,
			expectedErr: nil,
		},
		{
			name:        "Valid nested JSON",
			param:       "test",
			json:        `{"key":{"nested_key":"nested_value"}}`,
			expectedErr: nil,
		},
		{
			name:        "Invalid JSON syntax",
			param:       "test",
			json:        `{"key":"value}`,
			expectedErr: ErrInvalidJSON,
		},
		{
			name:        "JSON with space in key",
			param:       "test",
			json:        `{"key key":"value"}`,
			expectedErr: ErrJSONContainsSpaces,
		},
		{
			name:        "JSON with space in value",
			param:       "test",
			json:        `{"key":"value value"}`,
			expectedErr: ErrJSONContainsSpaces,
		},
		{
			name:        "JSON with space in nested key",
			param:       "test",
			json:        `{"key":{"nested key":"nested_value"}}`,
			expectedErr: ErrJSONContainsSpaces,
		},
		{
			name:        "JSON with space in nested value",
			param:       "test",
			json:        `{"key":{"nested_key":"nested value"}}`,
			expectedErr: ErrJSONContainsSpaces,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateJSON(tc.param, tc.json)
			if !errors.Is(err, tc.expectedErr) {
				t.Errorf("expected err %v, got %v", tc.expectedErr, err)
			}
		})
	}
}
