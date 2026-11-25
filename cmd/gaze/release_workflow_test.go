package main

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// The ship pin is format-pinned, not value-pinned: Dependabot's ship
// group in .github/dependabot.yml bumps the SHA on each ship release,
// and a literal-SHA assertion would fail CI on those bumps and block
// the auto-merge behind them. The guards below still pin the shape —
// the exact go-release.yml path, a full 40-hex commit SHA (CI-001, no
// mutable tags), and a semver version comment (CI-003). Which commit
// the SHA names is resolved by Dependabot itself or verified at
// authoring time per CI-002.
var (
	shipWorkflowRe = regexp.MustCompile(`^jflowers/ship/\.github/workflows/go-release\.yml@[0-9a-f]{40}$`)

	// shipPinCommentRe matches the release job's uses line including its
	// trailing version comment, e.g.
	//
	//	uses: jflowers/ship/.github/workflows/go-release.yml@86af… # v0.2.1
	shipPinCommentRe = regexp.MustCompile(`(?m)^ *uses: jflowers/ship/\.github/workflows/go-release\.yml@[0-9a-f]{40} # v[0-9]+\.[0-9]+\.[0-9]+$`)
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
// that the ship pin stays a full-SHA pin of the exact workflow (which
// commit runs is Dependabot's ship group or authoring-time CI-002
// verification, not a literal here), and that the local gates run first.
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

	if !shipWorkflowRe.MatchString(release.Uses) {
		t.Errorf("release job uses %q, want jflowers/ship/.github/workflows/go-release.yml pinned to a full 40-hex commit SHA — a mutable tag would silently change what the release runs (CI-001)", release.Uses)
	}
	if !shipPinCommentRe.MatchString(string(raw)) {
		t.Error("the ship pin must carry a semver version comment, e.g. `uses: jflowers/ship/.github/workflows/go-release.yml@<40-hex SHA> # v0.2.1` (CI-003)")
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
