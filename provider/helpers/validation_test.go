package helpers

import (
	"strings"
	"testing"
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
				if len(errors) == 0 {
					t.Errorf("expected an error but got none")
					return
				}

				if tt.errorContains != "" {
					errorStr := errors[0].Error()
					if !strings.Contains(errorStr, tt.errorContains) {
						t.Errorf("expected error to contain %q, got %q", tt.errorContains, errorStr)
					}
				}
			} else {
				if len(errors) > 0 {
					t.Errorf("expected no errors but got: %v", errors)
				}

				// Should always return nil for warnings
				if warnings != nil {
					t.Errorf("expected warnings to be nil, got %v", warnings)
				}
			}
		})
	}
}
