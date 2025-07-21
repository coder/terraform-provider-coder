package helpers

import (
	"fmt"
	"net/url"
)

// ValidateURL checks that the given value is a valid URL string.
// Example: for `icon = "/icon/region.svg"`, value is `/icon/region.svg` and label is `icon`.
func ValidateURL(value interface{}, label string) ([]string, []error) {
	val, ok := value.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected %q to be a string", label)}
	}
	if _, err := url.Parse(val); err != nil {
		return nil, []error{err}
	}
	return nil, nil
}
