package integration

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
)

// ErrInvalidURL is an error returned when a URL is invalid, malformed.
var ErrInvalidURL = errors.New("invalid URL")

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
