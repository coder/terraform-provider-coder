package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/masterminds/semver"
)

func main() {
	releases := fetchReleases()

	mainlineVer := highestRelease(releases, nil)
	_, _ = fmt.Fprintf(os.Stdout, "CODER_MAINLINE_VERSION=%q\n", formatVersion(mainlineVer))

	stableMinor := mainlineVer.Minor() - 1
	if stableMinor < 0 {
		stableMinor = 0
	}
	debug("expected stable minor: %d\n", stableMinor)
	stableVer := highestRelease(releases, &stableMinor)
	_, _ = fmt.Fprintf(os.Stdout, "CODER_STABLE_VERSION=%q\n", formatVersion(stableVer))

	oldStableMinor := mainlineVer.Minor() - 2
	if oldStableMinor < 0 {
		oldStableMinor = 0
	}
	debug("expected old stable minor: %d\n", oldStableMinor)
	oldStableVer := highestRelease(releases, &oldStableMinor)
	_, _ = fmt.Fprintf(os.Stdout, "CODER_OLDSTABLE_VERSION=%q\n", formatVersion(oldStableVer))
}

// highestRelease returns the highest non-prerelease semver from releases,
// optionally filtered to a specific minor version.
func highestRelease(releases []string, filterMinor *int64) *semver.Version {
	best := semver.MustParse("v0.0.0")
	for _, rel := range releases {
		if rel == "" {
			continue
		}
		ver, err := semver.NewVersion(rel)
		if err != nil {
			debug("skipping invalid version %s\n", rel)
			continue
		}
		if ver.Prerelease() != "" {
			debug("skipping prerelease version %s\n", rel)
			continue
		}
		if filterMinor != nil && ver.Minor() != *filterMinor {
			continue
		}
		if ver.Compare(best) > 0 {
			best = ver
		}
	}
	return best
}

// formatVersion formats a semver version as "vMAJOR.MINOR.PATCH",
// stripping any prerelease or build metadata.
func formatVersion(v *semver.Version) string {
	return fmt.Sprintf("v%d.%d.%d", v.Major(), v.Minor(), v.Patch())
}

type release struct {
	TagName string `json:"tag_name"`
}

const releasesURL = "https://api.github.com/repos/coder/coder/releases"

// fetchReleases fetches the releases of coder/coder
// this is done directly via JSON API to avoid pulling in the entire
// github client
func fetchReleases() []string {
	resp, err := http.Get(releasesURL)
	if err != nil {
		fatal("get releases: %s", err.Error())
	}
	defer resp.Body.Close()

	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		fatal("parse releases: %s", err.Error())
	}

	var ss []string
	for _, rel := range releases {
		if rel.TagName != "" {
			ss = append(ss, rel.TagName)
		}
	}
	return ss
}

func debug(format string, args ...any) {
	if _, ok := os.LookupEnv("VERBOSE"); ok {
		_, _ = fmt.Fprintf(os.Stderr, format, args...)
	}
}

func fatal(format string, args ...any) {
	_, _ = fmt.Fprintf(os.Stderr, format, args...)
	os.Exit(1)
}
