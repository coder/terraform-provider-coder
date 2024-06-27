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

	mainlineVer := semver.MustParse("v0.0.0")
	for _, rel := range releases {
		if rel == "" {
			debug("ignoring untagged version %s\n", rel)
			continue
		}

		ver, err := semver.NewVersion(rel)
		if err != nil {
			debug("skipping invalid version %s\n", rel)
		}

		if ver.Compare(mainlineVer) > 0 {
			mainlineVer = ver
			continue
		}
	}

	mainline := fmt.Sprintf("v%d.%d.%d", mainlineVer.Major(), mainlineVer.Minor(), mainlineVer.Patch())
	_, _ = fmt.Fprintf(os.Stdout, "CODER_MAINLINE_VERSION=%q\n", mainline)

	expectedStableMinor := mainlineVer.Minor() - 1
	if expectedStableMinor < 0 {
		expectedStableMinor = 0
	}
	debug("expected stable minor: %d\n", expectedStableMinor)
	stableVer := semver.MustParse("v0.0.0")
	for _, rel := range releases {
		debug("check version %s\n", rel)
		if rel == "" {
			debug("ignoring untagged version %s\n", rel)
			continue
		}

		ver, err := semver.NewVersion(rel)
		if err != nil {
			debug("skipping invalid version %s\n", rel)
		}

		if ver.Minor() != expectedStableMinor {
			debug("skipping version %s\n", rel)
			continue
		}

		if ver.Compare(stableVer) > 0 {
			stableVer = ver
			continue
		}
	}

	stable := fmt.Sprintf("v%d.%d.%d", stableVer.Major(), stableVer.Minor(), stableVer.Patch())
	_, _ = fmt.Fprintf(os.Stdout, "CODER_STABLE_VERSION=%q\n", stable)
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
		fatal("get releases: %w", err)
	}
	defer resp.Body.Close()

	var releases []release
	if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
		fatal("parse releases: %w", err)
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
