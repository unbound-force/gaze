package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// ufCapabilitiesFixture is a realistic pre-registration
// reviewer-capabilities.yaml: operational comments, mixed capability
// entries, and the trailing "# scaffolded by uf" scaffold marker.
const ufCapabilitiesFixture = `# Canonical eligibility policy for Divisor personas.
version: 1

reviewers:
  - agent: divisor-adversary
    capability: review
    scopes:
      - security
      - dependencies
  - agent: divisor-testing
    capability: review
    scopes:
      - test-quality
  - agent: divisor-scribe
    capability: content
    scopes: []
# scaffolded by uf vdev
`

// ufMatrixFixture is a realistic pre-registration review-matrix.yaml:
// comments, always overrides, advisor lists carrying the generic
// divisor-testing, and the trailing scaffold marker.
const ufMatrixFixture = `# Canonical model policy for Divisor review dispatch.
version: 3

# These overrides apply only when the advisor generates a run.
always:
  divisor-adversary: heavy

advisor:
  code:
    - divisor-adversary
    - divisor-guard
    - divisor-sre
    - divisor-testing
    - divisor-entropy
  spec:
    - divisor-adversary
    - divisor-guard
    - divisor-sre
    - divisor-testing
  triage:
    - divisor-adversary
    - divisor-guard
    - divisor-sre
    - divisor-testing
  feedback:
    - divisor-adversary
    - divisor-guard
    - divisor-sre
    - divisor-testing
    - divisor-entropy
# scaffolded by uf vdev
`

// ufTriageBlockNeedle is the advisor.triage block in ufMatrixFixture.
// Both derived fixtures are cut from this one literal so they cannot
// disagree about what the base fixture contains.
const ufTriageBlockNeedle = "  triage:\n    - divisor-adversary\n    - divisor-guard\n    - divisor-sre\n    - divisor-testing\n"

// matrixFixtureWithTriage derives ufMatrixFixture with its advisor.triage
// block replaced by replacement. strings.Replace returns the input
// unchanged when the needle stops matching, so the helper fails the test
// on a no-op replacement: otherwise base-fixture drift would surface as a
// misleading success-path diagnostic in the two skip cases.
func matrixFixtureWithTriage(t *testing.T, replacement string) string {
	t.Helper()
	derived := strings.Replace(ufMatrixFixture, ufTriageBlockNeedle, replacement, 1)
	if derived == ufMatrixFixture {
		t.Fatalf("ufMatrixFixture no longer contains ufTriageBlockNeedle %q; update the needle", ufTriageBlockNeedle)
	}
	return derived
}

// ufSyncCase drives the table-driven syncUFReviewConfig branch
// matrix: each case stages .uf files (or their absence) and asserts
// the registration outcome, the stdout notices/warnings, and the
// on-disk end-state.
type ufSyncCase struct {
	name           string
	files          map[string]string // keyed by path relative to the target dir
	ufIsFile       bool              // stage .uf as a regular file, not a directory
	preRegister    bool              // run one sync before the case run (already-registered setup)
	wantRegistered []string
	wantStdout     []string // substrings that must appear in stdout
	wantNoStdout   []string // substrings that must not appear in stdout
	verify         func(t *testing.T, dir string)
}

// TestSyncUFReviewConfig exercises every syncUFReviewConfig branch:
// absent .uf, registration, no-op, missing files, non-directory .uf,
// malformed YAML, and missing expected keys. Malformed or missing
// files warn and are skipped without failing the operation; the two
// config files are handled independently.
func TestSyncUFReviewConfig(t *testing.T) {
	// Derived fixtures are staged once so each case input and its
	// byte-identity expectation cannot drift apart.
	noTriage := matrixFixtureWithTriage(t, "")
	triageNotSequence := matrixFixtureWithTriage(t, "  triage: {}\n")

	cases := []ufSyncCase{
		{
			name:           "absent .uf directory prints notice and creates nothing",
			files:          map[string]string{},
			wantRegistered: nil,
			wantStdout:     []string{"no .uf directory found", "Gaze never creates .uf files"},
			verify: func(t *testing.T, dir string) {
				if _, err := os.Stat(filepath.Join(dir, ".uf")); !os.IsNotExist(err) {
					t.Errorf(".uf directory must not be created (stat err=%v)", err)
				}
			},
		},
		{
			name: "registers persona in both config files",
			files: map[string]string{
				".uf/reviewer-capabilities.yaml": ufCapabilitiesFixture,
				".uf/review-matrix.yaml":         ufMatrixFixture,
			},
			wantRegistered: []string{ufCapabilitiesRelPath, ufMatrixRelPath},
			wantNoStdout:   []string{"Warning"},
			verify: func(t *testing.T, dir string) {
				assertCapabilitiesRegistered(t, filepath.Join(dir, ".uf", "reviewer-capabilities.yaml"))
				assertMatrixSwapped(t, filepath.Join(dir, ".uf", "review-matrix.yaml"))
			},
		},
		{
			name: "already registered is a no-op with no warnings",
			files: map[string]string{
				".uf/reviewer-capabilities.yaml": ufCapabilitiesFixture,
				".uf/review-matrix.yaml":         ufMatrixFixture,
			},
			// The setup pre-registers via a first sync run; the case
			// run must then be an idempotent no-op.
			preRegister:  true,
			wantNoStdout: []string{"Warning", "Notice"},
			verify: func(t *testing.T, dir string) {
				assertCapabilitiesRegistered(t, filepath.Join(dir, ".uf", "reviewer-capabilities.yaml"))
				assertMatrixSwapped(t, filepath.Join(dir, ".uf", "review-matrix.yaml"))
			},
		},
		{
			name: "missing capabilities file warns and is not created",
			files: map[string]string{
				".uf/review-matrix.yaml": ufMatrixFixture,
			},
			wantRegistered: []string{ufMatrixRelPath},
			wantStdout:     []string{"reviewer-capabilities.yaml not found; skipping"},
			verify: func(t *testing.T, dir string) {
				if _, err := os.Stat(filepath.Join(dir, ".uf", "reviewer-capabilities.yaml")); !os.IsNotExist(err) {
					t.Errorf("capabilities file must not be created (stat err=%v)", err)
				}
			},
		},
		{
			name: "missing matrix file warns and is not created",
			files: map[string]string{
				".uf/reviewer-capabilities.yaml": ufCapabilitiesFixture,
			},
			wantRegistered: []string{ufCapabilitiesRelPath},
			wantStdout:     []string{"review-matrix.yaml not found; skipping"},
			verify: func(t *testing.T, dir string) {
				if _, err := os.Stat(filepath.Join(dir, ".uf", "review-matrix.yaml")); !os.IsNotExist(err) {
					t.Errorf("matrix file must not be created (stat err=%v)", err)
				}
			},
		},
		{
			name:       ".uf as a regular file warns and skips",
			ufIsFile:   true,
			wantStdout: []string{"is not a directory"},
			verify: func(t *testing.T, dir string) {
				// The .uf file itself must be left untouched.
				data, err := os.ReadFile(filepath.Join(dir, ".uf"))
				if err != nil || len(data) == 0 {
					t.Errorf(".uf file was modified or unreadable: %v", err)
				}
			},
		},
		{
			name: "malformed capabilities warns and stays unchanged while matrix merges",
			files: map[string]string{
				".uf/reviewer-capabilities.yaml": "version: 1\nreviewers: [unclosed\n",
				".uf/review-matrix.yaml":         ufMatrixFixture,
			},
			wantRegistered: []string{ufMatrixRelPath},
			wantStdout:     []string{"parsing", "reviewer-capabilities.yaml"},
			verify: func(t *testing.T, dir string) {
				assertFileContent(t, filepath.Join(dir, ".uf", "reviewer-capabilities.yaml"),
					"version: 1\nreviewers: [unclosed\n")
			},
		},
		{
			name: "matrix missing advisor key warns and stays unchanged",
			files: map[string]string{
				".uf/reviewer-capabilities.yaml": ufCapabilitiesFixture,
				".uf/review-matrix.yaml":         "version: 3\nruns: {}\n",
			},
			wantRegistered: []string{ufCapabilitiesRelPath},
			wantStdout:     []string{"review-matrix.yaml", "advisor"},
			verify: func(t *testing.T, dir string) {
				assertFileContent(t, filepath.Join(dir, ".uf", "review-matrix.yaml"),
					"version: 3\nruns: {}\n")
			},
		},
		{
			name: "matrix missing one advisor mode skips the whole file",
			files: map[string]string{
				".uf/review-matrix.yaml": strings.Replace(ufMatrixFixture, "  feedback:\n    - divisor-adversary\n    - divisor-guard\n    - divisor-sre\n    - divisor-testing\n    - divisor-entropy\n", "", 1),
			},
			wantRegistered: nil,
			wantStdout:     []string{"advisor.feedback"},
			verify: func(t *testing.T, dir string) {
				data, err := os.ReadFile(filepath.Join(dir, ".uf", "review-matrix.yaml"))
				if err != nil {
					t.Fatalf("reading matrix: %v", err)
				}
				if strings.Contains(string(data), ufGazeTestingAgent) {
					t.Errorf("matrix must not be partially registered when a mode is missing")
				}
			},
		},
		{
			name: "matrix without advisor.triage skips the whole file",
			files: map[string]string{
				".uf/review-matrix.yaml": noTriage,
			},
			wantRegistered: nil,
			wantStdout:     []string{"advisor.triage"},
			verify: func(t *testing.T, dir string) {
				assertFileContent(t, filepath.Join(dir, ".uf", "review-matrix.yaml"), noTriage)
			},
		},
		{
			name: "matrix with non-sequence advisor.triage skips the whole file",
			files: map[string]string{
				".uf/review-matrix.yaml": triageNotSequence,
			},
			wantRegistered: nil,
			wantStdout:     []string{"advisor.triage"},
			verify: func(t *testing.T, dir string) {
				assertFileContent(t, filepath.Join(dir, ".uf", "review-matrix.yaml"), triageNotSequence)
			},
		},
		{
			name: "capabilities missing reviewers key warns and stays unchanged",
			files: map[string]string{
				".uf/reviewer-capabilities.yaml": "# comment\nversion: 1\n",
			},
			wantRegistered: nil,
			wantStdout:     []string{"reviewer-capabilities.yaml", "reviewers"},
			verify: func(t *testing.T, dir string) {
				assertFileContent(t, filepath.Join(dir, ".uf", "reviewer-capabilities.yaml"),
					"# comment\nversion: 1\n")
			},
		},
		{
			name: "empty capabilities file warns and stays unchanged",
			files: map[string]string{
				".uf/reviewer-capabilities.yaml": "",
			},
			wantRegistered: nil,
			wantStdout:     []string{"reviewer-capabilities.yaml", "no document root"},
			verify: func(t *testing.T, dir string) {
				assertFileContent(t, filepath.Join(dir, ".uf", "reviewer-capabilities.yaml"), "")
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			if len(tc.files) > 0 {
				if err := os.MkdirAll(filepath.Join(dir, ".uf"), 0o755); err != nil {
					t.Fatalf("creating .uf: %v", err)
				}
			}
			for rel, content := range tc.files {
				full := filepath.Join(dir, filepath.FromSlash(rel))
				if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
					t.Fatalf("writing %s: %v", rel, err)
				}
			}
			if tc.ufIsFile {
				if err := os.RemoveAll(filepath.Join(dir, ".uf")); err != nil {
					t.Fatalf("removing .uf dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(dir, ".uf"), []byte("not a dir\n"), 0o644); err != nil {
					t.Fatalf("writing .uf file: %v", err)
				}
			}
			if tc.preRegister {
				var pre bytes.Buffer
				syncUFReviewConfig(Options{TargetDir: dir, Stdout: &pre})
			}

			var buf bytes.Buffer
			registered := syncUFReviewConfig(Options{TargetDir: dir, Stdout: &buf})

			assertStringSlicesEqual(t, "registered", tc.wantRegistered, registered)
			output := buf.String()
			for _, want := range tc.wantStdout {
				if !strings.Contains(output, want) {
					t.Errorf("stdout should contain %q, got:\n%s", want, output)
				}
			}
			for _, banned := range tc.wantNoStdout {
				if strings.Contains(output, banned) {
					t.Errorf("stdout should NOT contain %q, got:\n%s", banned, output)
				}
			}
			if tc.verify != nil {
				tc.verify(t, dir)
			}
		})
	}
}

// TestSyncUFReviewConfig_IdempotentNoRewrite verifies the
// byte-stability contract: after the first registration, a second
// run registers nothing, prints no warnings, and never rewrites the
// files. The no-write proof uses read-only files — any write attempt
// would fail loudly and print a warning.
func TestSyncUFReviewConfig_IdempotentNoRewrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".uf"), 0o755); err != nil {
		t.Fatalf("creating .uf: %v", err)
	}
	paths := map[string]string{
		".uf/reviewer-capabilities.yaml": ufCapabilitiesFixture,
		".uf/review-matrix.yaml":         ufMatrixFixture,
	}
	for rel, content := range paths {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", rel, err)
		}
	}

	var buf1 bytes.Buffer
	if registered := syncUFReviewConfig(Options{TargetDir: dir, Stdout: &buf1}); len(registered) != 2 {
		t.Fatalf("first run should register both files, got %v", registered)
	}

	firstPass, err := readUFFiles(dir)
	if err != nil {
		t.Fatalf("reading first-pass files: %v", err)
	}

	// Make both files read-only: a second-run write attempt would
	// fail and surface as a warning, proving the no-write contract
	// by contradiction.
	for rel := range paths {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.Chmod(full, 0o444); err != nil {
			t.Fatalf("chmod %s: %v", rel, err)
		}
		defer func(p string) { _ = os.Chmod(p, 0o644) }(full)
	}

	var buf2 bytes.Buffer
	registered := syncUFReviewConfig(Options{TargetDir: dir, Stdout: &buf2})
	if len(registered) != 0 {
		t.Errorf("second run must register nothing, got %v", registered)
	}
	if out := buf2.String(); strings.Contains(out, "Warning") {
		t.Errorf("second run must not warn (a warning implies a write attempt), got:\n%s", out)
	}

	secondPass, err := readUFFiles(dir)
	if err != nil {
		t.Fatalf("reading second-pass files: %v", err)
	}
	for rel, first := range firstPass {
		if !bytes.Equal(first, secondPass[rel]) {
			t.Errorf("%s: second run must be byte-identical (first %d bytes, second %d bytes)",
				rel, len(first), len(secondPass[rel]))
		}
	}
}

// TestSyncUFReviewConfig_CommentPreservation verifies that the
// yaml.Node round-trip preserves operational comments and the
// trailing "# scaffolded by uf" scaffold marker.
func TestSyncUFReviewConfig_CommentPreservation(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".uf"), 0o755); err != nil {
		t.Fatalf("creating .uf: %v", err)
	}
	for rel, content := range map[string]string{
		".uf/reviewer-capabilities.yaml": ufCapabilitiesFixture,
		".uf/review-matrix.yaml":         ufMatrixFixture,
	} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", rel, err)
		}
	}

	var buf bytes.Buffer
	syncUFReviewConfig(Options{TargetDir: dir, Stdout: &buf})

	for rel, wants := range map[string][]string{
		".uf/reviewer-capabilities.yaml": {
			"# Canonical eligibility policy for Divisor personas.",
			"# scaffolded by uf vdev",
		},
		".uf/review-matrix.yaml": {
			"# Canonical model policy for Divisor review dispatch.",
			"# These overrides apply only when the advisor generates a run.",
			"# scaffolded by uf vdev",
		},
	} {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			t.Fatalf("reading %s: %v", rel, err)
		}
		for _, want := range wants {
			if !strings.Contains(string(data), want) {
				t.Errorf("%s: comment %q not preserved, got:\n%s", rel, want, data)
			}
		}
	}
}

// TestSyncUFReviewConfig_RepoConfigIsEndState enforces design D6:
// the gaze repository's own .uf config files are already in the
// exact end-state syncUFReviewConfig produces, so running the sync
// against them is an idempotent no-op.
func TestSyncUFReviewConfig_RepoConfigIsEndState(t *testing.T) {
	projectRoot, err := findProjectRoot()
	if err != nil {
		t.Fatalf("finding project root: %v", err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".uf"), 0o755); err != nil {
		t.Fatalf("creating .uf: %v", err)
	}
	for _, name := range []string{"reviewer-capabilities.yaml", "review-matrix.yaml"} {
		data, err := os.ReadFile(filepath.Join(projectRoot, ".uf", name))
		if err != nil {
			t.Fatalf("reading repo .uf/%s: %v", name, err)
		}
		if err := os.WriteFile(filepath.Join(dir, ".uf", name), data, 0o644); err != nil {
			t.Fatalf("staging .uf/%s: %v", name, err)
		}
	}

	var buf bytes.Buffer
	registered := syncUFReviewConfig(Options{TargetDir: dir, Stdout: &buf})
	if len(registered) != 0 {
		t.Errorf("repo .uf config must already be in the sync end-state, but sync rewrote: %v", registered)
	}
	if out := buf.String(); strings.Contains(out, "Warning") {
		t.Errorf("sync against repo .uf config must not warn, got:\n%s", out)
	}
}

// TestCleanupOrphanedTestingReviewer verifies the orphan-cleanup
// contract: only gaze-marker-bearing reviewer-testing.md copies are
// removed; marker-less (user-customized) copies and absent files are
// left alone silently.
func TestCleanupOrphanedTestingReviewer(t *testing.T) {
	cases := []struct {
		name        string
		content     string // empty + absent=true means no file
		absent      bool
		wantRemoved bool
		wantStdout  []string
	}{
		{
			name: "marker-bearing orphan removed and reported",
			content: "---\ndescription: old persona\n---\n" +
				"<!-- scaffolded by gaze v1.0.0 -->\n# Testing reviewer\n",
			wantRemoved: true,
		},
		{
			name:        "marker with surrounding whitespace removed",
			content:     "# Body\n  <!-- scaffolded by gaze dev -->  \nmore\n",
			wantRemoved: true,
		},
		{
			name:        "marker-less customized copy untouched",
			content:     "---\ndescription: my custom persona\n---\n# My customizations\n",
			wantRemoved: false,
		},
		{
			name:        "non-gaze HTML comment is not a marker",
			content:     "<!-- a regular comment -->\n# Body\n",
			wantRemoved: false,
		},
		{
			name:        "absent file is a silent no-op",
			absent:      true,
			wantRemoved: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			legacyPath := filepath.Join(dir, ".opencode", "agents", "reviewer-testing.md")
			if !tc.absent {
				if err := os.MkdirAll(filepath.Dir(legacyPath), 0o755); err != nil {
					t.Fatalf("creating agents dir: %v", err)
				}
				if err := os.WriteFile(legacyPath, []byte(tc.content), 0o644); err != nil {
					t.Fatalf("writing orphan: %v", err)
				}
			}

			var buf bytes.Buffer
			removed := cleanupOrphanedTestingReviewer(Options{TargetDir: dir, Stdout: &buf})

			if tc.wantRemoved {
				if len(removed) != 1 || removed[0] != legacyTestingReviewerRelPath {
					t.Errorf("expected removal reported as [%s], got %v", legacyTestingReviewerRelPath, removed)
				}
				if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
					t.Errorf("marker-bearing orphan must be deleted (stat err=%v)", err)
				}
			} else {
				if len(removed) != 0 {
					t.Errorf("expected no removals, got %v", removed)
				}
				if !tc.absent {
					if _, err := os.Stat(legacyPath); err != nil {
						t.Errorf("marker-less orphan must be left on disk: %v", err)
					}
				}
				if out := buf.String(); out != "" {
					t.Errorf("non-removal paths must be silent, got stdout:\n%s", out)
				}
			}
		})
	}
}

// TestRun_UFRegistrationAndOrphanCleanup verifies the Run() wiring
// end to end: a uf-using target project gets the persona registered,
// a gaze-marked reviewer-testing.md orphan is removed and reported in
// the summary, and the renamed agent file is scaffolded.
func TestRun_UFRegistrationAndOrphanCleanup(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(dir, ".uf"), 0o755); err != nil {
		t.Fatalf("creating .uf: %v", err)
	}
	for rel, content := range map[string]string{
		".uf/reviewer-capabilities.yaml": ufCapabilitiesFixture,
		".uf/review-matrix.yaml":         ufMatrixFixture,
	} {
		if err := os.WriteFile(filepath.Join(dir, filepath.FromSlash(rel)), []byte(content), 0o644); err != nil {
			t.Fatalf("writing %s: %v", rel, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dir, ".opencode", "agents"), 0o755); err != nil {
		t.Fatalf("creating agents dir: %v", err)
	}
	orphan := "---\ndescription: old\n---\n<!-- scaffolded by gaze v1.0.0 -->\n# Old\n"
	if err := os.WriteFile(filepath.Join(dir, ".opencode", "agents", "reviewer-testing.md"), []byte(orphan), 0o644); err != nil {
		t.Fatalf("writing orphan: %v", err)
	}

	var buf bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	if len(result.Created) != 9 {
		t.Errorf("expected 9 created files, got %d: %v", len(result.Created), result.Created)
	}
	assertStringSlicesEqual(t, "registered", []string{ufCapabilitiesRelPath, ufMatrixRelPath}, result.Registered)
	assertStringSlicesEqual(t, "removed", []string{legacyTestingReviewerRelPath}, result.Removed)

	if _, err := os.Stat(filepath.Join(dir, ".opencode", "agents", "reviewer-testing.md")); !os.IsNotExist(err) {
		t.Errorf("orphan must be removed from disk (stat err=%v)", err)
	}
	if _, err := os.Stat(filepath.Join(dir, ".opencode", "agents", "divisor-gaze-testing.md")); err != nil {
		t.Errorf("renamed agent must be scaffolded: %v", err)
	}

	output := buf.String()
	for _, want := range []string{
		"registered: .uf/reviewer-capabilities.yaml (divisor-gaze-testing)",
		"registered: .uf/review-matrix.yaml (divisor-gaze-testing)",
		"removed: .opencode/agents/reviewer-testing.md",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("summary should contain %q, got:\n%s", want, output)
		}
	}
	if strings.Contains(output, "Notice") {
		t.Errorf(".uf present: no absence notice expected, got:\n%s", output)
	}
}

// TestRun_NoUF_PrintsNotice verifies that a project without .uf
// still completes init with the informational notice.
func TestRun_NoUF_PrintsNotice(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n"), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}

	var buf bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "1.0.0", Stdout: &buf})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(result.Created) != 9 {
		t.Errorf("expected 9 created files, got %d", len(result.Created))
	}
	if len(result.Registered) != 0 || len(result.Removed) != 0 {
		t.Errorf("no .uf project: expected no registrations/removals, got %v / %v",
			result.Registered, result.Removed)
	}
	if _, err := os.Stat(filepath.Join(dir, ".uf")); !os.IsNotExist(err) {
		t.Errorf(".uf must not be created (stat err=%v)", err)
	}
	if !strings.Contains(buf.String(), "no .uf directory found") {
		t.Errorf("summary should contain the notice, got:\n%s", buf.String())
	}
}

// assertCapabilitiesRegistered asserts the registered end-state of a
// reviewer-capabilities.yaml: the gaze persona entry present, the
// generic entry retained.
func assertCapabilitiesRegistered(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading capabilities: %v", err)
	}
	var caps struct {
		Reviewers []struct {
			Agent      string   `yaml:"agent"`
			Capability string   `yaml:"capability"`
			Scopes     []string `yaml:"scopes"`
		} `yaml:"reviewers"`
	}
	if err := yaml.Unmarshal(data, &caps); err != nil {
		t.Fatalf("parsing registered capabilities: %v\n%s", err, data)
	}
	var gaze, generic bool
	for _, r := range caps.Reviewers {
		switch r.Agent {
		case ufGazeTestingAgent:
			gaze = true
			if r.Capability != "review" {
				t.Errorf("gaze entry capability = %q, want review", r.Capability)
			}
			if len(r.Scopes) != 1 || r.Scopes[0] != "test-quality" {
				t.Errorf("gaze entry scopes = %v, want [test-quality]", r.Scopes)
			}
		case ufGenericTestingAgent:
			generic = true
		}
	}
	if !gaze {
		t.Errorf("gaze persona entry missing from:\n%s", data)
	}
	if !generic {
		t.Errorf("generic divisor-testing entry must be retained in:\n%s", data)
	}
}

// assertMatrixSwapped asserts the registered end-state of a
// review-matrix.yaml: gaze persona in the code/spec/triage/feedback
// advisor lists, generic persona removed from all four.
func assertMatrixSwapped(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading matrix: %v", err)
	}
	var matrix struct {
		Advisor struct {
			Code     []string `yaml:"code"`
			Spec     []string `yaml:"spec"`
			Triage   []string `yaml:"triage"`
			Feedback []string `yaml:"feedback"`
		} `yaml:"advisor"`
	}
	if err := yaml.Unmarshal(data, &matrix); err != nil {
		t.Fatalf("parsing registered matrix: %v\n%s", err, data)
	}
	for _, mode := range []struct {
		name string
		list []string
	}{
		{"code", matrix.Advisor.Code},
		{"spec", matrix.Advisor.Spec},
		{"triage", matrix.Advisor.Triage},
		{"feedback", matrix.Advisor.Feedback},
	} {
		if !containsString(mode.list, ufGazeTestingAgent) {
			t.Errorf("advisor.%s must contain %s:\n%s", mode.name, ufGazeTestingAgent, data)
		}
		if containsString(mode.list, ufGenericTestingAgent) {
			t.Errorf("advisor.%s must not contain %s:\n%s", mode.name, ufGenericTestingAgent, data)
		}
	}
}

// assertFileContent asserts a file's exact byte content.
func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", path, err)
	}
	if string(data) != want {
		t.Errorf("%s content changed:\ngot:  %q\nwant: %q", path, data, want)
	}
}

// assertStringSlicesEqual asserts two string slices are equal in
// order and content (nil equals empty).
func assertStringSlicesEqual(t *testing.T, what string, want, got []string) {
	t.Helper()
	if len(want) != len(got) {
		t.Errorf("%s: want %v, got %v", what, want, got)
		return
	}
	for i := range want {
		if want[i] != got[i] {
			t.Errorf("%s[%d]: want %q, got %q", what, i, want[i], got[i])
		}
	}
}

// containsString reports whether s is in the list.
func containsString(list []string, s string) bool {
	for _, v := range list {
		if v == s {
			return true
		}
	}
	return false
}

// readUFFiles reads the two .uf config files keyed by relative path.
func readUFFiles(dir string) (map[string][]byte, error) {
	out := map[string][]byte{}
	for _, rel := range []string{ufCapabilitiesRelPath, ufMatrixRelPath} {
		data, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(rel)))
		if err != nil {
			return nil, err
		}
		out[rel] = data
	}
	return out, nil
}
