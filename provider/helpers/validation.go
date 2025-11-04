package helpers

import (
	"fmt"
	"net/url"
)

// ValidateURL validates that value is a valid URL string.
// Accepts empty strings, local file paths, file:// URLs, and http/https URLs.
// Example: for `icon = "/icon/region.svg"`, value is `/icon/region.svg` and label is `icon`.
func ValidateURL(value any, label string) ([]string, []error) {
	val, ok := value.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected %q to be a string", label)}
	}

	if _, err := url.Parse(val); err != nil {
		return nil, []error{err}
	}

	return nil, nil
}
