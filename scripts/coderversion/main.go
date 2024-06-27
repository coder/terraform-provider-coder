package main

import (
	"context"
	"fmt"
	"os"

	"github.com/google/go-github/github"
	"github.com/masterminds/semver"
)

func main() {
	client := github.NewClient(nil)
	// We consider the latest release as 'mainline'
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	latest, _, err := client.Repositories.GetLatestRelease(ctx, "coder", "coder")
	if err != nil {
		fatal("get mainline release: %w\n", err)
	}

	if latest.TagName == nil {
		fatal("latest release is not tagged", err)
	}

	mainlineVer, err := semver.NewVersion(*latest.TagName)
	if err != nil {
		fatal("invalid mainline tag name: %w\n", err)
	}
	mainline := fmt.Sprintf("v%d.%d.%d", mainlineVer.Major(), mainlineVer.Minor(), mainlineVer.Patch())

	expectedStableMinor := mainlineVer.Minor() - 1
	if expectedStableMinor < 0 {
		fatal("unexpected minor version 0")
	}

	previousReleases, _, err := client.Repositories.ListReleases(ctx, "coder", "coder", &github.ListOptions{
		Page:    0,
		PerPage: 100,
	})
	if err != nil {
		fatal("list previous releases: %w\n", err)
	}

	var stable string
	for _, rel := range previousReleases {
		if rel.TagName == nil {
			debug("ignoring untagged version %s\n", rel.String())
			continue
		}

		ver, err := semver.NewVersion(*rel.TagName)
		if err != nil {
			debug("skipping invalid version %s\n", *rel.TagName)
		}

		// Assuming that the first one we find with minor-1 is what we want
		if ver.Minor() == expectedStableMinor {
			debug("found stable version %s\n", *rel.TagName)
			stable = fmt.Sprintf("v%d.%d.%d", ver.Major(), ver.Minor(), ver.Patch())
			break
		}

	}
	_, _ = fmt.Fprintf(os.Stdout, "CODER_MAINLINE_VERSION=%q\n", mainline)
	_, _ = fmt.Fprintf(os.Stdout, "CODER_STABLE_VERSION=%q\n", stable)
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
