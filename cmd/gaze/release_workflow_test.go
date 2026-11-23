package main

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The release pipeline lives in jflowers/ship's go-release.yml; this repo's
// release.yml only gates and calls it. These checks guard what the caller
// still decides: which secrets reach ship and under which names, that ship
// is pinned, and that the local gates run first.
func TestReleaseWorkflow_CallsShip(t *testing.T) {
	workflowPath := filepath.Join("..", "..", ".github", "workflows", "release.yml")
	workflow, err := os.ReadFile(workflowPath)
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	contents := string(workflow)

	// post-release-install-check.yml runs on completion of "Release".
	if !strings.Contains(contents, "\nname: Release\n") {
		t.Error(`release workflow must be named "Release" (post-release-install-check.yml listens for it)`)
	}

	pinned := regexp.MustCompile(`uses: jflowers/ship/\.github/workflows/go-release\.yml@[0-9a-f]{40} # v\d+\.\d+\.\d+`)
	if !pinned.MatchString(contents) {
		t.Error("release workflow must call jflowers/ship's go-release.yml pinned to a full commit SHA with its version comment")
	}

	releaseStart := strings.Index(contents, "\n  release:\n")
	if releaseStart < 0 {
		t.Fatal("release workflow must contain the release job")
	}
	release := contents[releaseStart:]

	needs := "needs: [validate-module-major, validate-workflow-floor, validate-ai-report-default]"
	if !strings.Contains(release, needs) {
		t.Errorf("release job must contain %q so the local gates run before anything is tagged", needs)
	}

	// Every signing secret goes to ship under its own name; ship signs only
	// when all of them are present and fails on a partial set.
	for _, name := range []string{
		"MACOS_SIGN_P12",
		"MACOS_SIGN_PASSWORD",
		"MACOS_SIGN_IDENTITY",
		"MACOS_NOTARY_KEY",
		"MACOS_NOTARY_KEY_ID",
		"MACOS_NOTARY_ISSUER_ID",
	} {
		mapping := name + ": ${{ secrets." + name + " }}"
		if !strings.Contains(release, mapping) {
			t.Errorf("release job must pass %q", mapping)
		}
	}
	tapToken := "HOMEBREW_TAP_TOKEN: ${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}"
	if !strings.Contains(release, tapToken) {
		t.Errorf("release job must pass %q", tapToken)
	}

	for _, forbidden := range []string{
		`vars.MACOS_SIGN_IDENTITY`,
		`Developer ID Application:`,
		// inherit doesn't cross into another owner's workflow, so secrets
		// would silently arrive empty and the release would ship unsigned.
		`secrets: inherit`,
	} {
		if strings.Contains(contents, forbidden) {
			t.Errorf("release workflow must not contain %q", forbidden)
		}
	}
}
