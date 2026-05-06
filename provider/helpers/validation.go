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

// WarnDirNotHome returns a warning if dir is set to a value other
// than $HOME, because this breaks Coder Desktop file sync. The dir
// attribute is deprecated and will be removed in a future release.
func WarnDirNotHome(val interface{}, _ string) ([]string, []error) {
	d, ok := val.(string)
	if !ok || d == "" || d == "$HOME" || d == "~" {
		return nil, nil
	}
	return []string{
		`"dir" is deprecated and will be removed in a future release.`,
		`Setting "dir" to a value other than $HOME will break Coder Desktop file sync.`,
	}, nil
}
