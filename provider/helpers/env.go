package helpers

import (
	"fmt"
	"os"
)

// RequireEnv requires environment variable to be present.
func RequireEnv(name string) (string, error) {
	val := os.Getenv(name)
	if val == "" {
		return "", fmt.Errorf("%s is required", name)
	}
	return val, nil
}

// OptionalEnv returns the value for environment variable if it exists,
// otherwise returns an empty string.
func OptionalEnv(name string) string {
	return OptionalEnvOrDefault(name, "")
}

// OptionalEnvOrDefault returns the value for environment variable if it exists,
// otherwise returns the default value.
func OptionalEnvOrDefault(name, defaultValue string) string {
	val := os.Getenv(name)
	if val == "" {
		return defaultValue
	}
	return val
}
