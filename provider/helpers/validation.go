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

// ValidateExternalURL validates that a URL intended for external use contains
// a scheme (e.g. http://, https://, vscode://, jetbrains-gateway://).
// Go's url.Parse is permissive and accepts bare strings like "my-repo" as
// relative URLs, but JavaScript's new URL() requires a scheme and will crash
// if one is not present.
func ValidateExternalURL(value any, label string) ([]string, []error) {
	val, ok := value.(string)
	if !ok {
		return nil, []error{fmt.Errorf("expected %q to be a string", label)}
	}

	if val == "" {
		return nil, nil
	}

	parsed, err := url.Parse(val)
	if err != nil {
		return nil, []error{err}
	}

	if parsed.Scheme == "" {
		return nil, []error{fmt.Errorf(
			"%q must have a URL scheme (e.g. https://, vscode://, jetbrains-gateway://), got %q",
			label, val,
		)}
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
