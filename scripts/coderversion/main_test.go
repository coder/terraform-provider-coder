package main

import (
	"testing"

	"github.com/masterminds/semver"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func int64Ptr(v int64) *int64 { return &v }

func TestHighestRelease(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		releases    []string
		filterMinor *int64
		expected    string
	}{
		{
			name:     "skips RC versions",
			releases: []string{"v2.32.0-rc.0", "v2.31.2", "v2.31.1"},
			expected: "v2.31.2",
		},
		{
			name:     "skips all prerelease variants",
			releases: []string{"v2.32.0-rc.0", "v2.32.0-beta.1", "v2.31.0"},
			expected: "v2.31.0",
		},
		{
			name:     "returns highest when no prereleases",
			releases: []string{"v2.31.2", "v2.31.1", "v2.30.0"},
			expected: "v2.31.2",
		},
		{
			name:        "filters by minor version",
			releases:    []string{"v2.32.1", "v2.31.3", "v2.31.2"},
			filterMinor: int64Ptr(31),
			expected:    "v2.31.3",
		},
		{
			name:        "filters by minor AND skips prerelease",
			releases:    []string{"v2.31.0-rc.1", "v2.31.2", "v2.30.0"},
			filterMinor: int64Ptr(31),
			expected:    "v2.31.2",
		},
		{
			name:     "skips invalid version strings",
			releases: []string{"v2.31.0", "not-a-version", "v2.30.0"},
			expected: "v2.31.0",
		},
		{
			name:     "handles empty strings",
			releases: []string{"", "v2.31.0"},
			expected: "v2.31.0",
		},
		{
			name:     "all prereleases returns v0.0.0",
			releases: []string{"v2.32.0-rc.0", "v2.31.0-beta.1"},
			expected: "v0.0.0",
		},
		{
			name:     "empty input returns v0.0.0",
			releases: []string{},
			expected: "v0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := highestRelease(tt.releases, tt.filterMinor)
			require.NotNil(t, got)
			expected := semver.MustParse(tt.expected)
			assert.Equal(t, 0, got.Compare(expected), "expected %s but got %s", tt.expected, got.Original())
		})
	}
}

func TestFormatVersion(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "standard version",
			input:    "v2.31.2",
			expected: "v2.31.2",
		},
		{
			name:     "zero version",
			input:    "v0.0.0",
			expected: "v0.0.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			v := semver.MustParse(tt.input)
			assert.Equal(t, tt.expected, formatVersion(v))
		})
	}
}
