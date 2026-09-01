package helpers

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name          string
		value         any
		label         string
		expectError   bool
		errorContains string
	}{
		// Valid cases
		{
			name:        "empty string",
			value:       "",
			label:       "url",
			expectError: false,
		},
		{
			name:        "valid http URL",
			value:       "http://example.com",
			label:       "url",
			expectError: false,
		},
		{
			name:        "valid https URL",
			value:       "https://example.com/path",
			label:       "url",
			expectError: false,
		},
		{
			name:        "absolute file path",
			value:       "/path/to/file",
			label:       "url",
			expectError: false,
		},
		{
			name:        "relative file path",
			value:       "./file.txt",
			label:       "url",
			expectError: false,
		},
		{
			name:        "relative path up directory",
			value:       "../config.json",
			label:       "url",
			expectError: false,
		},
		{
			name:        "simple filename",
			value:       "file.txt",
			label:       "url",
			expectError: false,
		},
		{
			name:        "URL with query params",
			value:       "https://example.com/search?q=test",
			label:       "url",
			expectError: false,
		},
		{
			name:        "URL with fragment",
			value:       "https://example.com/page#section",
			label:       "url",
			expectError: false,
		},

		// Various URL schemes that url.Parse accepts
		{
			name:        "file URL scheme",
			value:       "file:///path/to/file",
			label:       "url",
			expectError: false,
		},
		{
			name:        "ftp scheme",
			value:       "ftp://files.example.com/file.txt",
			label:       "url",
			expectError: false,
		},
		{
			name:        "mailto scheme",
			value:       "mailto:user@example.com",
			label:       "url",
			expectError: false,
		},
		{
			name:        "tel scheme",
			value:       "tel:+1234567890",
			label:       "url",
			expectError: false,
		},
		{
			name:        "data scheme",
			value:       "data:text/plain;base64,SGVsbG8=",
			label:       "url",
			expectError: false,
		},

		// Invalid cases
		{
			name:          "non-string type - int",
			value:         123,
			label:         "url",
			expectError:   true,
			errorContains: "expected \"url\" to be a string",
		},
		{
			name:          "non-string type - nil",
			value:         nil,
			label:         "config_url",
			expectError:   true,
			errorContains: "expected \"config_url\" to be a string",
		},
		{
			name:          "invalid URL with spaces",
			value:         "http://example .com",
			label:         "url",
			expectError:   true,
			errorContains: "invalid character",
		},
		{
			name:          "malformed URL",
			value:         "http://[::1:80",
			label:         "endpoint",
			expectError:   true,
			errorContains: "missing ']'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := ValidateURL(tt.value, tt.label)

			if tt.expectError {
				require.Len(t, errors, 1, "expected an error but got none")
				require.Contains(t, errors[0].Error(), tt.errorContains)
			} else {
				require.Empty(t, errors, "expected no errors but got: %v", errors)
			}

			// Should always return nil for warnings
			require.Nil(t, warnings, "expected warnings to be nil but got: %v", warnings)
		})
	}
}

func TestWarnDirNotHome(t *testing.T) {
	tests := []struct {
		name      string
		value     interface{}
		warnCount int
	}{
		// No warnings expected.
		{
			name:      "empty string",
			value:     "",
			warnCount: 0,
		},
		{
			name:      "$HOME",
			value:     "$HOME",
			warnCount: 0,
		},
		{
			name:      "tilde",
			value:     "~",
			warnCount: 0,
		},
		{
			name:      "non-string type",
			value:     123,
			warnCount: 0,
		},
		// Warnings expected.
		{
			name:      "absolute path",
			value:     "/workspace",
			warnCount: 2,
		},
		{
			name:      "relative path",
			value:     "projects/foo",
			warnCount: 2,
		},
		{
			name:      "tilde subdir",
			value:     "~/projects",
			warnCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, errors := WarnDirNotHome(tt.value, "dir")
			require.Empty(t, errors, "expected no errors")
			require.Len(t, warnings, tt.warnCount)
		})
	}
}

func TestValidateExternalURL(t *testing.T) {
	tests := []struct {
		name          string
		value         string
		errorContains string
	}{
		// Valid: a scheme and a host.
		{
			name:  "https URL",
			value: "https://example.com",
		},
		{
			name:  "https URL with path and query",
			value: "https://example.com/path?q=test",
		},
		{
			name:  "custom vscode scheme",
			value: "vscode://coder.coder-remote/open?token=$SESSION_TOKEN",
		},
		{
			name:  "reverse DNS scheme with a path",
			value: "com.example.app://callback",
		},
		{
			name:  "jetbrains gateway",
			value: "jetbrains-gateway://connect#provider=ssh&host=192.168.1.1&port=22&user=ubuntu&projectPath=/home/ubuntu/projects/my-app",
		},

		// Invalid: a bare host:port is read as a scheme with an opaque body, so
		// "localhost:8080" parses as the scheme "localhost" and leaves no host.
		{
			name:          "host:port without scheme",
			value:         "localhost:8080",
			errorContains: `"localhost" URLs must include a host`,
		},
		{
			name:          "domain and port without scheme",
			value:         "example.com:8080",
			errorContains: `"example.com" URLs must include a host`,
		},
		{
			name:          "host:port with a path",
			value:         "localhost:8080/path",
			errorContains: `"localhost" URLs must include a host`,
		},

		// Invalid: opaque URLs have no host, so there is nothing to navigate to.
		{
			name:          "opaque scheme",
			value:         "mailto:user@example.com",
			errorContains: "must include a host",
		},
		{
			name:          "opaque scheme with a numeric body",
			value:         "tel:5551234",
			errorContains: "must include a host",
		},

		// Invalid: no scheme, so new URL() throws.
		{
			name:          "bare string",
			value:         "my-repo",
			errorContains: "must include a scheme",
		},
		{
			name:          "absolute path",
			value:         "/some/path",
			errorContains: "must include a scheme",
		},
		{
			name:          "relative path",
			value:         "./file.txt",
			errorContains: "must include a scheme",
		},
		{
			name:          "protocol relative",
			value:         "//example.com",
			errorContains: "must include a scheme",
		},
		{
			name:          "empty string",
			value:         "",
			errorContains: "must include a scheme",
		},

		// Invalid: Go accepts a hostless special scheme, the browser does not.
		{
			name:          "https with no host",
			value:         "https://",
			errorContains: "must include a host",
		},

		// Invalid: unparsable by either parser.
		{
			name:          "malformed URL",
			value:         "http://[::1:80",
			errorContains: "missing ']'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExternalURL(tt.value)

			if tt.errorContains == "" {
				require.NoError(t, err)
				return
			}
			require.Error(t, err)
			require.Contains(t, err.Error(), tt.errorContains)
		})
	}
}
