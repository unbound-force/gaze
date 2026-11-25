package main

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// shipWorkflow is the pinned reusable workflow release.yml calls. Bumping
// ship means updating this pin and the version comment in release.yml.
const (
	shipWorkflow = "jflowers/ship/.github/workflows/go-release.yml@7c6c0e199099d732e0db93c8c9d86b0064c2759c"
	shipVersion  = "v0.9.0"
)

type releaseWorkflow struct {
	Name string                `yaml:"name"`
	Jobs map[string]releaseJob `yaml:"jobs"`
}

type releaseJob struct {
	Needs   []string          `yaml:"needs"`
	Uses    string            `yaml:"uses"`
	With    map[string]string `yaml:"with"`
	Secrets yaml.Node         `yaml:"secrets"`
}

// The release pipeline lives in jflowers/ship's go-release.yml; this repo's
// release.yml only gates and calls it. These checks guard what the caller
// still decides: which inputs and secrets reach ship and under which names,
// which ship commit runs, and that the local gates run first.
func TestReleaseWorkflow_CallsShip(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "release.yml"))
	if err != nil {
		t.Fatalf("read release workflow: %v", err)
	}
	var wf releaseWorkflow
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parse release workflow: %v", err)
	}

	// post-release-install-check.yml runs on completion of "Release".
	if wf.Name != "Release" {
		t.Errorf(`workflow name = %q, want "Release" (post-release-install-check.yml listens for it)`, wf.Name)
	}

	// The pipeline moved to ship; none of the old jobs may come back.
	var jobs []string
	for name := range wf.Jobs {
		jobs = append(jobs, name)
	}
	sort.Strings(jobs)
	wantJobs := []string{"release", "validate-ai-report-default", "validate-module-major", "validate-workflow-floor"}
	if strings.Join(jobs, ",") != strings.Join(wantJobs, ",") {
		t.Errorf("jobs = %v, want %v", jobs, wantJobs)
	}

	release, ok := wf.Jobs["release"]
	if !ok {
		t.Fatal("release workflow must contain the release job")
	}

	if release.Uses != shipWorkflow {
		t.Errorf("release job uses %q, want %q", release.Uses, shipWorkflow)
	}
	if !strings.Contains(string(raw), "uses: "+shipWorkflow+" # "+shipVersion+"\n") {
		t.Errorf("the ship pin must carry the comment %q", "# "+shipVersion)
	}

	wantNeeds := "validate-module-major,validate-workflow-floor,validate-ai-report-default"
	if got := strings.Join(release.Needs, ","); got != wantNeeds {
		t.Errorf("release needs = %q, want %q so the local gates run before anything is tagged", got, wantNeeds)
	}

	wantWith := map[string]string{
		"tag":                   "${{ inputs.tag }}",
		"ci_checks":             `["Unit + Integration Tests (Go 1.24)", "Unit + Integration Tests (Go 1.25)", "MegaLinter"]`,
		"allow_prerelease":      "true",
		"skip_semver_check":     "${{ inputs.skip_semver_check }}",
		"skip_ci_checks":        "${{ inputs.skip_ci_checks }}",
		"skip_unreleased_check": "${{ inputs.skip_unreleased_check }}",
		"tap":                   "unbound-force/homebrew-tap",
	}
	assertExactMap(t, "with", release.With, wantWith)

	// Secrets are mapped by name: `secrets: inherit` doesn't cross into
	// another owner's workflow, so they would arrive empty and the release
	// would ship unsigned. Ship signs only when all six MACOS_* are present
	// and fails on a partial set.
	if release.Secrets.Kind != yaml.MappingNode {
		t.Fatalf("release job secrets must be a mapping of names, not %q", release.Secrets.Value)
	}
	var secrets map[string]string
	if err := release.Secrets.Decode(&secrets); err != nil {
		t.Fatalf("decode release job secrets: %v", err)
	}
	wantSecrets := map[string]string{"HOMEBREW_TAP_TOKEN": "${{ secrets.HOMEBREW_TAP_GITHUB_TOKEN }}"}
	for _, name := range []string{
		"MACOS_SIGN_P12",
		"MACOS_SIGN_PASSWORD",
		"MACOS_SIGN_IDENTITY",
		"MACOS_NOTARY_KEY",
		"MACOS_NOTARY_KEY_ID",
		"MACOS_NOTARY_ISSUER_ID",
	} {
		wantSecrets[name] = "${{ secrets." + name + " }}"
	}
	assertExactMap(t, "secrets", secrets, wantSecrets)

	for _, forbidden := range []string{`vars.MACOS_SIGN_IDENTITY`, `Developer ID Application:`} {
		if strings.Contains(string(raw), forbidden) {
			t.Errorf("release workflow must not contain %q", forbidden)
		}
	}
}

func assertExactMap(t *testing.T, what string, got, want map[string]string) {
	t.Helper()
	for k, w := range want {
		if g, ok := got[k]; !ok {
			t.Errorf("release job %s is missing %s", what, k)
		} else if g != w {
			t.Errorf("release job %s.%s = %q, want %q", what, k, g, w)
		}
	}
	for k := range got {
		if _, ok := want[k]; !ok {
			t.Errorf("release job %s has unexpected %s", what, k)
		}
	}
}
