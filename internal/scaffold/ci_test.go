package scaffold

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// CI-scaffolding tests for tasks 3.1–3.6 of the init-ci-scaffold
// change: version resolution, go.mod directive parsing, template
// rendering (golden + YAML structure), skip-if-present semantics,
// flag gating, and the CI next-steps section. The
// ci-external-analyzer-support change adds variant detection and
// Node/Python caller-workflow tests (detection matrix, golden files,
// YAML structure, end-to-end scaffolding).

// goldenGazeVersion and goldenGoVersion are the fixed render inputs
// used by both golden-file tests. They must match the committed
// testdata/golden-gaze-ai-report.yml exactly.
const (
	goldenGazeVersion = "v2.4.0"
	goldenGoVersion   = "1.24"
)

// writeGoMod writes a go.mod with the given `go` directive line (or
// none, when directive is empty) into dir.
func writeGoMod(t *testing.T, dir, directive string) {
	t.Helper()
	content := "module test\n"
	if directive != "" {
		content += "\n" + directive + "\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(content), 0o644); err != nil {
		t.Fatalf("creating go.mod: %v", err)
	}
}

// ciWorkflowDiskPath returns the generated CI workflow's on-disk
// location under dir.
func ciWorkflowDiskPath(dir string) string {
	return filepath.Join(dir, filepath.FromSlash(ciWorkflowPath))
}

// assertVersionGuidance fails the test when err's message does not
// carry the fail-closed guidance: the floor value, the release-tag
// remedy, and the --gaze-version escape hatch (task 3.1).
func assertVersionGuidance(t *testing.T, err error) {
	t.Helper()
	msg := err.Error()
	for _, want := range []string{"v2.3.0", "release tag", "--gaze-version"} {
		if !strings.Contains(msg, want) {
			t.Errorf("version error should mention %q, got: %s", want, msg)
		}
	}
}

// TestResolveGazeVersion exercises the version-resolution matrix
// (task 3.1, design D4) against the pure resolver: valid releases,
// the inclusive floor, pre-releases above and of the floor,
// pre-floor releases, dev/empty/malformed strings, and override
// precedence.
func TestResolveGazeVersion(t *testing.T) {
	tests := []struct {
		name        string
		opts        Options
		wantVersion string
		wantErr     bool
	}{
		{name: "valid release", opts: Options{Version: "v2.4.0"}, wantVersion: "v2.4.0"},
		{name: "floor is inclusive", opts: Options{Version: "v2.3.0"}, wantVersion: "v2.3.0"},
		{name: "pre-release above floor", opts: Options{Version: "v2.4.0-rc1"}, wantVersion: "v2.4.0-rc1"},
		{name: "pre-release of floor sorts below", opts: Options{Version: "v2.3.0-rc1"}, wantErr: true},
		{name: "pre-floor release", opts: Options{Version: "v2.2.0"}, wantErr: true},
		{name: "pre-floor patch release", opts: Options{Version: "v2.2.9"}, wantErr: true},
		{name: "dev build", opts: Options{Version: "dev"}, wantErr: true},
		{name: "empty version", opts: Options{Version: ""}, wantErr: true},
		{name: "missing v prefix", opts: Options{Version: "2.3.0"}, wantErr: true},
		{name: "doubled v prefix", opts: Options{Version: "vv2.3.0"}, wantErr: true},
		{name: "partial version passes bare semver but not strict form", opts: Options{Version: "v2.3"}, wantErr: true},
		{name: "build metadata rejected", opts: Options{Version: "v2.4.0+build.1"}, wantErr: true},
		{name: "override takes precedence over dev build", opts: Options{Version: "dev", GazeVersion: "v2.5.0"}, wantVersion: "v2.5.0"},
		{name: "invalid override fails even with valid build-time version", opts: Options{Version: "v9.9.9", GazeVersion: "banana"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveGazeVersion(tt.opts)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("resolveGazeVersion(%+v) = %q, want error", tt.opts, got)
				}
				assertVersionGuidance(t, err)
				return
			}
			if err != nil {
				t.Fatalf("resolveGazeVersion(%+v) returned error: %v", tt.opts, err)
			}
			if got != tt.wantVersion {
				t.Errorf("resolveGazeVersion(%+v) = %q, want %q", tt.opts, got, tt.wantVersion)
			}
		})
	}
}

// TestParseGoModVersion exercises the go.mod directive parser matrix
// (task 3.5, design D5) against the pure parser: valid forms pass
// through, trailing whitespace and CRLF carriage returns are
// stripped, and absent or garbage directives fall back to
// defaultGoVersion.
func TestParseGoModVersion(t *testing.T) {
	tests := []struct {
		name         string
		content      string
		wantVersion  string
		wantFellBack bool
	}{
		{name: "major.minor", content: "module test\n\ngo 1.25\n", wantVersion: "1.25"},
		{name: "major.minor.patch", content: "module test\n\ngo 1.25.0\n", wantVersion: "1.25.0"},
		{name: "major.minor.patch pre-release", content: "go 1.25.0-rc1\n", wantVersion: "1.25.0-rc1"},
		{name: "trailing whitespace stripped", content: "go 1.25   \n", wantVersion: "1.25"},
		{name: "CRLF carriage return stripped", content: "go 1.24\r\n", wantVersion: "1.24"},
		{name: "no go directive", content: "module test\n", wantVersion: "1.25", wantFellBack: true},
		{name: "garbage directive value", content: "go banana\n", wantVersion: "1.25", wantFellBack: true},
		{name: "directive value with trailing comment", content: "go 1.25\" # note\n", wantVersion: "1.25", wantFellBack: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, fellBack := parseGoModVersion(tt.content)
			if got != tt.wantVersion {
				t.Errorf("parseGoModVersion(%q) = %q, want %q", tt.content, got, tt.wantVersion)
			}
			if fellBack != tt.wantFellBack {
				t.Errorf("parseGoModVersion(%q) fellBack = %v, want %v", tt.content, fellBack, tt.wantFellBack)
			}
		})
	}
}

// TestReadGoModVersion_MissingFileFallsBack verifies an unreadable
// go.mod reports the fallback rather than an error (design D5: the
// file I/O wrapper keeps failures as unreadable-directive signals).
func TestReadGoModVersion_MissingFileFallsBack(t *testing.T) {
	version, fellBack := readGoModVersion(filepath.Join(t.TempDir(), "does-not-exist", "go.mod"))
	if version != defaultGoVersion {
		t.Errorf("readGoModVersion(missing) = %q, want %q", version, defaultGoVersion)
	}
	if !fellBack {
		t.Error("readGoModVersion(missing) fellBack = false, want true")
	}
}

// TestRenderCIWorkflowTemplate_GoldenFile verifies the rendered
// workflow matches the committed golden file byte-for-byte with both
// placeholders substituted and none remaining (task 3.2, design D9).
// It renders through renderCIWorkflowTemplateForVariant — the same
// seam the write path uses for every variant, Go included — so the
// golden guards the production renderer rather than a test-only twin.
func TestRenderCIWorkflowTemplate_GoldenFile(t *testing.T) {
	template, err := assets.ReadFile(ciWorkflowTemplatePath)
	if err != nil {
		t.Fatalf("reading embedded CI workflow template: %v", err)
	}

	rendered := renderCIWorkflowTemplateForVariant(template, ciContext{
		gazeVersion: goldenGazeVersion,
		goVersion:   goldenGoVersion,
		variant:     ciVariantGo,
	})

	golden, err := os.ReadFile(filepath.Join("testdata", "golden-gaze-ai-report.yml"))
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	if !bytes.Equal(rendered, golden) {
		t.Errorf("rendered workflow does not match golden file.\nRun: go run ./cmd/gaze init --ci # in a temp module, or regenerate the golden from the template\n--- rendered ---\n%s\n--- golden ---\n%s", rendered, golden)
	}

	assertNoPlaceholders(t, string(rendered))
}

// TestRenderCIWorkflowTemplate_YAMLStructure verifies the rendered
// workflow parses as YAML and carries the structural contract of
// design D9 / task 3.2: workflow name, least-privilege permissions
// at both levels, CI-012 concurrency, the reusable-workflow call
// with its two live inputs, and the self-contained test job.
func TestRenderCIWorkflowTemplate_YAMLStructure(t *testing.T) {
	template, err := assets.ReadFile(ciWorkflowTemplatePath)
	if err != nil {
		t.Fatalf("reading embedded CI workflow template: %v", err)
	}

	rendered := string(renderCIWorkflowTemplateForVariant(template, ciContext{
		gazeVersion: goldenGazeVersion,
		goVersion:   goldenGoVersion,
		variant:     ciVariantGo,
	}))

	var doc map[string]any
	if err := yaml.Unmarshal([]byte(rendered), &doc); err != nil {
		t.Fatalf("rendered workflow does not parse as YAML: %v", err)
	}

	if doc["name"] != "Gaze AI Report" {
		t.Errorf("workflow name = %v, want \"Gaze AI Report\"", doc["name"])
	}

	// Workflow-level least privilege.
	perms, ok := doc["permissions"].(map[string]any)
	if !ok || perms["contents"] != "read" {
		t.Errorf("workflow-level permissions = %v, want contents: read", doc["permissions"])
	}

	// CI-012 concurrency: superseded PR runs cancel, main runs queue.
	conc, ok := doc["concurrency"].(map[string]any)
	if !ok {
		t.Fatalf("concurrency block missing or not a mapping: %v", doc["concurrency"])
	}
	if conc["group"] != "${{ github.ref }}-${{ github.workflow }}" {
		t.Errorf("concurrency group = %v, unexpected", conc["group"])
	}
	if conc["cancel-in-progress"] != "${{ github.ref != 'refs/heads/main' }}" {
		t.Errorf("cancel-in-progress = %v, unexpected", conc["cancel-in-progress"])
	}

	jobs, ok := doc["jobs"].(map[string]any)
	if !ok {
		t.Fatalf("jobs missing or not a mapping: %v", doc["jobs"])
	}

	// Self-contained test job.
	testJob, ok := jobs["test"].(map[string]any)
	if !ok {
		t.Fatalf("test job missing: %v", jobs)
	}
	if testJob["runs-on"] != "ubuntu-latest" {
		t.Errorf("test job runs-on = %v, want ubuntu-latest", testJob["runs-on"])
	}
	if testJob["timeout-minutes"] != 30 {
		t.Errorf("test job timeout-minutes = %v, want 30", testJob["timeout-minutes"])
	}

	// The reusable-workflow caller job.
	report, ok := jobs["gaze-ai-report"].(map[string]any)
	if !ok {
		t.Fatalf("gaze-ai-report job missing: %v", jobs)
	}
	if report["needs"] != "test" {
		t.Errorf("gaze-ai-report needs = %v, want test", report["needs"])
	}
	rperms, ok := report["permissions"].(map[string]any)
	if !ok || rperms["contents"] != "read" {
		t.Errorf("gaze-ai-report job-level permissions = %v, want contents: read", report["permissions"])
	}

	wantUses := "unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@" + goldenGazeVersion
	if report["uses"] != wantUses {
		t.Errorf("gaze-ai-report uses = %v, want %v", report["uses"], wantUses)
	}

	with, ok := report["with"].(map[string]any)
	if !ok {
		t.Fatalf("gaze-ai-report with block missing: %v", report["with"])
	}
	if len(with) != 2 {
		t.Errorf("gaze-ai-report with block has %d inputs, want exactly 2 (thresholds stay commented out): %v", len(with), with)
	}
	if with["coverage-artifact"] != "coverage-out" {
		t.Errorf("coverage-artifact = %v, want coverage-out", with["coverage-artifact"])
	}
	if with["gaze-version"] != goldenGazeVersion {
		t.Errorf("gaze-version input = %v, want %v (identical to the uses: pin)", with["gaze-version"], goldenGazeVersion)
	}

	secrets, ok := report["secrets"].(map[string]any)
	if !ok || secrets["opencode-config"] != "${{ secrets.OPENCODE_CONFIG_CONTENT }}" {
		t.Errorf("secrets = %v, want opencode-config from OPENCODE_CONFIG_CONTENT", report["secrets"])
	}

	// Raw-text contract: SHA-pinned actions, the coverage test
	// command, and the comment-only threshold lines.
	rawChecks := []string{
		"actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683",
		"actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff",
		"actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02",
		"go test -race -count=1 -short -coverprofile=coverage.out ./...",
		"# max-crapload: <N>",
		"# max-gaze-crapload: <N>",
		"# min-contract-coverage: <N>",
	}
	for _, want := range rawChecks {
		if !strings.Contains(rendered, want) {
			t.Errorf("rendered workflow missing %q", want)
		}
	}
}

// TestCIWorkflowTemplate_AntiDrift pins the template's reusable-
// workflow reference and the floor constant to literals (task 3.2):
// a workflow rename or floor change cannot silently diverge from
// the scaffolding contract without failing this test.
func TestCIWorkflowTemplate_AntiDrift(t *testing.T) {
	template, err := assets.ReadFile(ciWorkflowTemplatePath)
	if err != nil {
		t.Fatalf("reading embedded CI workflow template: %v", err)
	}
	s := string(template)

	wantUses := "unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@" + gazeVersionPlaceholder
	if !strings.Contains(s, wantUses) {
		t.Errorf("template uses: reference = want %q — a reusable-workflow rename must update the template and golden file", wantUses)
	}

	// The floor must stay pinned to the first release tag containing
	// the reusable workflow; changing it is a deliberate contract
	// change that must touch this test.
	if minWorkflowTag != "v2.3.0" {
		t.Errorf("minWorkflowTag = %q, want \"v2.3.0\" (first release containing the reusable workflow)", minWorkflowTag)
	}
}

// TestRun_CI_SkipIfPresent verifies skip-if-present semantics for
// the user-owned CI workflow (task 3.3, design D7): a rerun without
// --force leaves user edits byte-identical and reports the file as
// skipped; --force regenerates it.
func TestRun_CI_SkipIfPresent(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "go 1.24")

	var buf1 bytes.Buffer
	result1, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf1})
	if err != nil {
		t.Fatalf("first Run() returned error: %v", err)
	}
	if len(result1.Created) != 10 {
		t.Fatalf("expected 10 created files (9 .opencode/ + workflow), got %d: %v", len(result1.Created), result1.Created)
	}

	workflow := ciWorkflowDiskPath(dir)

	// Hand-edit the generated file, as a user adopting thresholds
	// would: uncomment the max-crapload input with a real value.
	original, err := os.ReadFile(workflow)
	if err != nil {
		t.Fatalf("reading generated workflow: %v", err)
	}
	edited := strings.Replace(string(original),
		"# max-crapload: <N>  # TODO: fail when the CRAPload exceeds this value",
		"max-crapload: 38", 1)
	if edited == string(original) {
		t.Fatal("threshold edit did not apply — comment line not found in generated workflow")
	}
	if err := os.WriteFile(workflow, []byte(edited), 0o644); err != nil {
		t.Fatalf("writing edited workflow: %v", err)
	}

	// Rerun without --force: the file must survive byte-identical
	// and be reported as skipped.
	var buf2 bytes.Buffer
	result2, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf2})
	if err != nil {
		t.Fatalf("second Run() returned error: %v", err)
	}
	after, err := os.ReadFile(workflow)
	if err != nil {
		t.Fatalf("re-reading workflow after rerun: %v", err)
	}
	if !bytes.Equal(after, []byte(edited)) {
		t.Error("rerun without --force clobbered user edits in the CI workflow")
	}
	if !containsString(result2.Skipped, ciWorkflowPath) {
		t.Errorf("expected %s in Skipped, got: %v", ciWorkflowPath, result2.Skipped)
	}
	if !strings.Contains(buf2.String(), "skipped: "+ciWorkflowPath) {
		t.Errorf("summary should report the workflow as skipped, got:\n%s", buf2.String())
	}
	// Pins the --force hint's only surviving trigger: since
	// commands/gaze.md became tool-owned, the CI workflow is the
	// one user-owned file a re-run can skip, so the summary must
	// print the hint naming exactly one file.
	if !strings.Contains(buf2.String(), "1 file skipped (use --force to overwrite)") {
		t.Errorf("summary should print the --force hint for the skipped CI workflow, got:\n%s", buf2.String())
	}

	// Rerun with --force: the file is regenerated (edits gone) and
	// reported as overwritten.
	var buf3 bytes.Buffer
	result3, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Force: true, Stdout: &buf3})
	if err != nil {
		t.Fatalf("third Run() with force returned error: %v", err)
	}
	regen, err := os.ReadFile(workflow)
	if err != nil {
		t.Fatalf("re-reading workflow after force rerun: %v", err)
	}
	if !bytes.Equal(regen, original) {
		t.Error("--force rerun did not regenerate the pristine workflow")
	}
	if !containsString(result3.Overwritten, ciWorkflowPath) {
		t.Errorf("expected %s in Overwritten, got: %v", ciWorkflowPath, result3.Overwritten)
	}
}

// TestRun_PlainInit_NoCIWorkflow verifies plain init (no --ci)
// writes no workflow file and prints no CI guidance (task 3.4).
func TestRun_PlainInit_NoCIWorkflow(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "go 1.24")

	var buf bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "v2.4.0", Stdout: &buf})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(result.Created) != 9 {
		t.Errorf("expected 9 created files, got %d: %v", len(result.Created), result.Created)
	}
	if _, err := os.Stat(ciWorkflowDiskPath(dir)); !os.IsNotExist(err) {
		t.Error("plain init (no --ci) must not write the CI workflow")
	}
	if strings.Contains(buf.String(), "CI next steps") {
		t.Errorf("plain init must not print CI next steps, got:\n%s", buf.String())
	}
}

// TestRun_CI_OptInEmitsWorkflow verifies gaze init --ci in a
// Go-module-at-root directory writes both the .opencode/ assets and
// the workflow (task 3.4).
func TestRun_CI_OptInEmitsWorkflow(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "go 1.24")

	var buf bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(result.Created) != 10 {
		t.Fatalf("expected 10 created files (9 .opencode/ + workflow), got %d: %v", len(result.Created), result.Created)
	}
	if !containsString(result.Created, ciWorkflowPath) {
		t.Errorf("expected %s in Created, got: %v", ciWorkflowPath, result.Created)
	}

	content, err := os.ReadFile(ciWorkflowDiskPath(dir))
	if err != nil {
		t.Fatalf("reading generated workflow: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "reusable_gaze_ai_report.yml@v2.4.0") {
		t.Errorf("generated workflow missing uses: pin for v2.4.0:\n%s", s)
	}
	if !strings.Contains(s, "gaze-version: v2.4.0") {
		t.Errorf("generated workflow missing gaze-version input v2.4.0:\n%s", s)
	}
}

// TestRun_CI_MissingGoMod_SkipsCI verifies --ci without go.mod
// prints both warnings, writes no workflow, and completes the
// .opencode/ scaffolding (task 3.4, design D6).
func TestRun_CI_MissingGoMod_SkipsCI(t *testing.T) {
	dir := t.TempDir()
	// Deliberately do NOT create go.mod.

	var buf bytes.Buffer
	result, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf})
	if err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(result.Created) != 9 {
		t.Errorf("expected 9 created files (.opencode/ scaffolding proceeds), got %d: %v", len(result.Created), result.Created)
	}
	if _, err := os.Stat(ciWorkflowDiskPath(dir)); !os.IsNotExist(err) {
		t.Error("--ci without go.mod must not write the CI workflow")
	}

	output := buf.String()
	if !strings.Contains(output, "Warning: no go.mod found") {
		t.Errorf("expected generic missing-go.mod warning, got:\n%s", output)
	}
	if !strings.Contains(output, "Warning: skipping CI scaffolding") {
		t.Errorf("expected CI-skip warning, got:\n%s", output)
	}
	if strings.Contains(output, "CI next steps") {
		t.Errorf("go.mod-skip run must suppress the CI next-steps section, got:\n%s", output)
	}
}

// TestRun_CI_DevBuildFailsClosed verifies failed version validation
// writes nothing at all (task 3.4, design D4): the version check
// runs before the .opencode/ asset walk and before the go.mod
// check, so even the combined dev-build-without-go.mod case fails
// on the version error with no skip warning.
func TestRun_CI_DevBuildFailsClosed(t *testing.T) {
	tests := []struct {
		name        string
		goMod       bool
		version     string
		gazeVersion string
	}{
		{name: "dev build with go.mod", goMod: true, version: "dev"},
		{name: "dev build without go.mod", goMod: false, version: "dev"},
		{name: "invalid override", goMod: true, version: "v2.4.0", gazeVersion: "banana"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			if tt.goMod {
				writeGoMod(t, dir, "go 1.24")
			}

			var buf bytes.Buffer
			result, err := Run(Options{
				TargetDir:   dir,
				Version:     tt.version,
				GazeVersion: tt.gazeVersion,
				CI:          true,
				Stdout:      &buf,
			})
			if err == nil {
				t.Fatalf("Run() succeeded, want fail-closed version error (result: %+v)", result)
			}
			if result != nil {
				t.Errorf("Run() returned a result alongside the error: %+v", result)
			}
			assertVersionGuidance(t, err)

			// Fail-closed before the walk: nothing written, and the
			// go.mod skip path was never taken (no skip warning).
			for _, written := range []string{".opencode", ".github"} {
				if _, err := os.Stat(filepath.Join(dir, written)); !os.IsNotExist(err) {
					t.Errorf("fail-closed run wrote %s", written)
				}
			}
			if output := buf.String(); output != "" {
				t.Errorf("fail-closed run printed output before failing, got:\n%s", output)
			}
		})
	}
}

// TestRun_CI_GoModDirectiveSubstituted verifies end-to-end directive
// substitution: a go.mod declaring `go 1.24` yields a generated
// workflow whose setup-go step pins that version (task 3.4).
func TestRun_CI_GoModDirectiveSubstituted(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "go 1.24")

	var buf bytes.Buffer
	if _, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf}); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	content, err := os.ReadFile(ciWorkflowDiskPath(dir))
	if err != nil {
		t.Fatalf("reading generated workflow: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "go-version: \"1.24\"") {
		t.Errorf("generated workflow missing go-version \"1.24\" from go.mod directive:\n%s", s)
	}
	if strings.Contains(buf.String(), "no readable `go` directive") {
		t.Errorf("valid directive must not print the fallback warning, got:\n%s", buf.String())
	}
}

// TestRun_CI_UnparseableDirectiveFallsBack verifies a go.mod whose
// directive is outside the accepted grammar falls back to
// defaultGoVersion in the generated workflow with the fallback
// warning (task 3.4, design D5).
func TestRun_CI_UnparseableDirectiveFallsBack(t *testing.T) {
	dir := t.TempDir()
	writeGoMod(t, dir, "go banana")

	var buf bytes.Buffer
	if _, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf}); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}

	content, err := os.ReadFile(ciWorkflowDiskPath(dir))
	if err != nil {
		t.Fatalf("reading generated workflow: %v", err)
	}
	if s := string(content); !strings.Contains(s, "go-version: \"1.25\"") {
		t.Errorf("generated workflow missing fallback go-version \"1.25\":\n%s", s)
	}
	if output := buf.String(); !strings.Contains(output, "Warning: no readable `go` directive in go.mod") {
		t.Errorf("expected fallback warning, got:\n%s", output)
	}
}

// TestRun_CI_NextSteps verifies the CI next-steps section (task
// 3.6, design D10): all four guidance items print on a successful
// --ci run and on a rerun that skips the existing workflow, and the
// section is suppressed when CI scaffolding was skipped for a
// missing go.mod.
func TestRun_CI_NextSteps(t *testing.T) {
	assertAllItems := func(t *testing.T, output string) {
		t.Helper()
		items := []string{
			"CI next steps for " + ciWorkflowPath,
			"gh secret set OPENCODE_CONFIG_CONTENT < opencode-config.json",
			"`jq -c .`",
			"OPENCODE_API_KEY",
			"gaze report ./... --format=json",
		}
		for _, want := range items {
			if !strings.Contains(output, want) {
				t.Errorf("CI next steps missing %q, got:\n%s", want, output)
			}
		}
	}

	t.Run("successful run prints all four items", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "go 1.24")

		var buf bytes.Buffer
		if _, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf}); err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		assertAllItems(t, buf.String())
	})

	t.Run("rerun reporting file as skipped still prints the section", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "go 1.24")

		var buf1 bytes.Buffer
		if _, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf1}); err != nil {
			t.Fatalf("first Run() returned error: %v", err)
		}

		var buf2 bytes.Buffer
		result, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf2})
		if err != nil {
			t.Fatalf("second Run() returned error: %v", err)
		}
		if !containsString(result.Skipped, ciWorkflowPath) {
			t.Fatalf("expected workflow reported skipped on rerun, got: %v", result.Skipped)
		}
		assertAllItems(t, buf2.String())
	})

	t.Run("go.mod skip suppresses the section", func(t *testing.T) {
		dir := t.TempDir()
		// Deliberately do NOT create go.mod.

		var buf bytes.Buffer
		if _, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf}); err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if strings.Contains(buf.String(), "CI next steps") {
			t.Errorf("go.mod-skip run must not print CI next steps, got:\n%s", buf.String())
		}
	})
}

// renderVariantTemplate loads and renders one variant caller workflow
// and asserts it matches the committed golden file byte-for-byte. The
// toolchain versions deliberately render from the
// default{Node,Python}Version constants rather than local literals:
// bumping a default must fail the golden comparison and force
// regenerating the committed golden alongside the documented version
// instead of silently diverging from it.
func renderVariantTemplate(t *testing.T, templatePath, goldenPath string, ci ciContext) string {
	t.Helper()
	template, err := assets.ReadFile(templatePath)
	if err != nil {
		t.Fatalf("reading embedded CI workflow template %s: %v", templatePath, err)
	}
	rendered := renderCIWorkflowTemplateForVariant(template, ci)
	golden, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("reading golden file %s: %v", goldenPath, err)
	}
	if !bytes.Equal(rendered, golden) {
		t.Errorf("rendered workflow does not match golden file %s.\n--- rendered ---\n%s\n--- golden ---\n%s", goldenPath, rendered, golden)
	}
	return string(rendered)
}

// TestDetectCIVariant exercises the variant-detection matrix
// (ci-external-analyzer-support, design D5): the helper runs only for
// directories whose go.mod is already established to be absent, so
// package.json selects Node, pyproject.toml selects Python, and a
// directory with neither marker returns the Go variant whose skip
// semantics the caller owns.
func TestDetectCIVariant(t *testing.T) {
	tests := []struct {
		name  string
		files []string
		want  ciVariant
	}{
		{name: "package.json only", files: []string{"package.json"}, want: ciVariantNode},
		{name: "pyproject.toml only", files: []string{"pyproject.toml"}, want: ciVariantPython},
		{name: "package.json wins over pyproject.toml", files: []string{"package.json", "pyproject.toml"}, want: ciVariantNode},
		{name: "no recognized marker defaults to go", files: nil, want: ciVariantGo},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			for _, file := range tt.files {
				if err := os.WriteFile(filepath.Join(dir, file), []byte("{}\n"), 0o644); err != nil {
					t.Fatalf("creating %s: %v", file, err)
				}
			}
			if got := detectCIVariant(Options{TargetDir: dir}); got != tt.want {
				t.Errorf("detectCIVariant(files=%v) = %q, want %q", tt.files, got, tt.want)
			}
		})
	}
}

// TestRenderCIWorkflowTemplate_NodeGoldenFile verifies the rendered
// Node caller workflow matches its committed golden file byte-for-byte
// with all placeholders substituted and none remaining
// (ci-external-analyzer-support, task 2.4).
func TestRenderCIWorkflowTemplate_NodeGoldenFile(t *testing.T) {
	rendered := renderVariantTemplate(t, ciWorkflowNodeTemplatePath,
		filepath.Join("testdata", "golden-gaze-ai-report-node.yml"),
		ciContext{gazeVersion: goldenGazeVersion, nodeVersion: defaultNodeVersion, variant: ciVariantNode})
	assertNoPlaceholders(t, rendered)
}

// TestRenderCIWorkflowTemplate_PythonGoldenFile verifies the rendered
// Python caller workflow matches its committed golden file
// byte-for-byte with all placeholders substituted and none remaining
// (ci-external-analyzer-support, task 2.4).
func TestRenderCIWorkflowTemplate_PythonGoldenFile(t *testing.T) {
	rendered := renderVariantTemplate(t, ciWorkflowPythonTemplatePath,
		filepath.Join("testdata", "golden-gaze-ai-report-python.yml"),
		ciContext{gazeVersion: goldenGazeVersion, pythonVersion: defaultPythonVersion, variant: ciVariantPython})
	assertNoPlaceholders(t, rendered)
}

// placeholderResidueRe matches any remaining template placeholder
// token regardless of variant. The renderer substitutes every
// variant's placeholders, so a cross-variant placeholder (e.g.
// __GO_VERSION__ copy-pasted into the Node template) substitutes to
// the ciContext value — empty in the golden fixtures — and would
// vanish silently; a generic dunder-token scan catches it where the
// per-variant lists the golden tests previously enumerated could not.
var placeholderResidueRe = regexp.MustCompile(`__[A-Z][A-Z0-9_]*__`)

// assertNoPlaceholders fails when a rendered workflow still contains
// any placeholder token.
func assertNoPlaceholders(t *testing.T, rendered string) {
	t.Helper()
	if residue := placeholderResidueRe.FindString(rendered); residue != "" {
		t.Errorf("rendered workflow still contains placeholder %s", residue)
	}
}

// workflowStep returns the named step of a decoded workflow job,
// failing when absent.
func workflowStep(t *testing.T, job map[string]any, name string) map[string]any {
	t.Helper()
	steps, ok := job["steps"].([]any)
	if !ok {
		t.Fatalf("job has no steps list: %v", job)
	}
	for _, s := range steps {
		step, ok := s.(map[string]any)
		if ok && step["name"] == name {
			return step
		}
	}
	t.Fatalf("job has no step named %q", name)
	return nil
}

// TestRenderCIWorkflowTemplate_VariantYAMLStructure verifies the
// rendered Node and Python caller workflows parse as YAML and carry
// the ci-scaffolding spec's structural contract: least-privilege
// permissions at both levels, CI-012 concurrency, the variant's
// self-contained test job (toolchain setup pinned to the substituted
// version, install, coverage run, upload), and the
// reusable-workflow call passing analyzer, packages, and the variant
// toolchain input with thresholds commented out. The golden-file
// comparison rides along through renderVariantTemplate.
func TestRenderCIWorkflowTemplate_VariantYAMLStructure(t *testing.T) {
	tests := []struct {
		name           string
		templatePath   string
		goldenPath     string
		ci             ciContext
		analyzer       string
		setupStep      string
		setupUses      string
		versionInput   string
		testJobVersion string // quoted string in the test job's with block
		withVersion    string // unquoted scalar in the report job's with block
		installRun     string
		testRun        string
		stageRun       string // coverage staging step; empty = no staging step
		uploadPath     string
		setupInput     string // analyzer-install snippet passed via `setup:`
	}{
		{
			name:           "node",
			templatePath:   ciWorkflowNodeTemplatePath,
			goldenPath:     filepath.Join("testdata", "golden-gaze-ai-report-node.yml"),
			ci:             ciContext{gazeVersion: goldenGazeVersion, nodeVersion: defaultNodeVersion, variant: ciVariantNode},
			analyzer:       "gaze-analyzer-typescript",
			setupStep:      "Set up Node",
			setupUses:      "actions/setup-node@49933ea5288caeca8642d1e84afbd3f7d6820020",
			versionInput:   "node-version",
			testJobVersion: "24",
			withVersion:    "24",
			installRun:     "npm ci",
			testRun:        "npm test -- --coverage",
			stageRun:       "mkdir -p .gaze-artifact && cp -r coverage .gaze-artifact/",
			uploadPath:     ".gaze-artifact",
			setupInput:     "npm ci && npm run build && npm link",
		},
		{
			name:           "python",
			templatePath:   ciWorkflowPythonTemplatePath,
			goldenPath:     filepath.Join("testdata", "golden-gaze-ai-report-python.yml"),
			ci:             ciContext{gazeVersion: goldenGazeVersion, pythonVersion: defaultPythonVersion, variant: ciVariantPython},
			analyzer:       "gaze-analyzer-python",
			setupStep:      "Set up Python",
			setupUses:      "actions/setup-python@a26af69be951a213d495a4c3e4e4022e16d87065",
			versionInput:   "python-version",
			testJobVersion: "3.12",
			withVersion:    "3.12",
			installRun:     "pip install -e . && pip install pytest pytest-cov",
			testRun:        "pytest --cov --cov-report=xml --cov-report=term",
			stageRun:       "", // coverage.xml is a single file: no LCA stripping to compensate for
			uploadPath:     "coverage.xml",
			setupInput:     "pip install -e .",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rendered := renderVariantTemplate(t, tt.templatePath, tt.goldenPath, tt.ci)

			var doc map[string]any
			if err := yaml.Unmarshal([]byte(rendered), &doc); err != nil {
				t.Fatalf("rendered %s workflow does not parse as YAML: %v", tt.name, err)
			}

			if doc["name"] != "Gaze AI Report" {
				t.Errorf("%s workflow name = %v, want \"Gaze AI Report\"", tt.name, doc["name"])
			}

			// Workflow-level least privilege.
			perms, ok := doc["permissions"].(map[string]any)
			if !ok || perms["contents"] != "read" {
				t.Errorf("%s workflow-level permissions = %v, want contents: read", tt.name, doc["permissions"])
			}

			// CI-012 concurrency: superseded PR runs cancel, main runs queue.
			conc, ok := doc["concurrency"].(map[string]any)
			if !ok {
				t.Fatalf("%s concurrency block missing or not a mapping: %v", tt.name, doc["concurrency"])
			}
			if conc["group"] != "${{ github.ref }}-${{ github.workflow }}" {
				t.Errorf("%s concurrency group = %v, unexpected", tt.name, conc["group"])
			}
			if conc["cancel-in-progress"] != "${{ github.ref != 'refs/heads/main' }}" {
				t.Errorf("%s cancel-in-progress = %v, unexpected", tt.name, conc["cancel-in-progress"])
			}

			jobs, ok := doc["jobs"].(map[string]any)
			if !ok {
				t.Fatalf("%s jobs missing or not a mapping: %v", tt.name, doc["jobs"])
			}

			// Self-contained variant test job.
			testJob, ok := jobs["test"].(map[string]any)
			if !ok {
				t.Fatalf("%s test job missing: %v", tt.name, jobs)
			}
			if testJob["runs-on"] != "ubuntu-latest" {
				t.Errorf("%s test job runs-on = %v, want ubuntu-latest", tt.name, testJob["runs-on"])
			}
			if testJob["timeout-minutes"] != 30 {
				t.Errorf("%s test job timeout-minutes = %v, want 30", tt.name, testJob["timeout-minutes"])
			}
			setup := workflowStep(t, testJob, tt.setupStep)
			if setup["uses"] != tt.setupUses {
				t.Errorf("%s setup step uses = %v, want %s", tt.name, setup["uses"], tt.setupUses)
			}
			setupWith, ok := setup["with"].(map[string]any)
			if !ok || fmt.Sprint(setupWith[tt.versionInput]) != tt.testJobVersion {
				t.Errorf("%s setup step with[%s] = %v, want %q", tt.name, tt.versionInput, setup["with"], tt.testJobVersion)
			}
			if got := workflowStep(t, testJob, "Install dependencies")["run"]; got != tt.installRun {
				t.Errorf("%s install step run = %v, want %q", tt.name, got, tt.installRun)
			}
			if got := workflowStep(t, testJob, "Test with coverage")["run"]; got != tt.testRun {
				t.Errorf("%s test step run = %v, want %q", tt.name, got, tt.testRun)
			}
			if tt.stageRun != "" {
				if got := workflowStep(t, testJob, "Stage coverage for upload")["run"]; got != tt.stageRun {
					t.Errorf("%s staging step run = %v, want %q", tt.name, got, tt.stageRun)
				}
			}
			upload := workflowStep(t, testJob, "Upload coverage")
			uploadWith, ok := upload["with"].(map[string]any)
			if !ok || uploadWith["name"] != "coverage-out" || uploadWith["path"] != tt.uploadPath {
				t.Errorf("%s upload step with = %v, want name coverage-out and path %s", tt.name, upload["with"], tt.uploadPath)
			}

			// The reusable-workflow caller job.
			report, ok := jobs["gaze-ai-report"].(map[string]any)
			if !ok {
				t.Fatalf("%s gaze-ai-report job missing: %v", tt.name, jobs)
			}
			if report["needs"] != "test" {
				t.Errorf("%s gaze-ai-report needs = %v, want test", tt.name, report["needs"])
			}
			rperms, ok := report["permissions"].(map[string]any)
			if !ok || rperms["contents"] != "read" {
				t.Errorf("%s gaze-ai-report job-level permissions = %v, want contents: read", tt.name, report["permissions"])
			}
			wantUses := "unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@" + goldenGazeVersion
			if report["uses"] != wantUses {
				t.Errorf("%s gaze-ai-report uses = %v, want %v", tt.name, report["uses"], wantUses)
			}

			with, ok := report["with"].(map[string]any)
			if !ok {
				t.Fatalf("%s gaze-ai-report with block missing: %v", tt.name, report["with"])
			}
			if len(with) != 6 {
				t.Errorf("%s gaze-ai-report with block has %d inputs, want exactly 6 (thresholds stay commented out): %v", tt.name, len(with), with)
			}
			wantWith := map[string]string{
				"coverage-artifact": "coverage-out",
				"gaze-version":      goldenGazeVersion,
				"analyzer":          tt.analyzer,
				"packages":          "./src/...",
				tt.versionInput:     tt.withVersion,
				"setup":             tt.setupInput,
			}
			for key, want := range wantWith {
				if got := fmt.Sprint(with[key]); got != want {
					t.Errorf("%s gaze-ai-report with[%s] = %q, want %q", tt.name, key, got, want)
				}
			}
			// Tie the invoked analyzer to its installer: a template
			// that passes `analyzer:` without a `setup:` input
			// scaffolds a workflow whose report step cannot find the
			// analyzer binary on a fresh runner — and it fails only
			// after the consumer sets the opencode-config secret, the
			// deferred-failure class this repository has fixed before.
			if with["analyzer"] != nil && fmt.Sprint(with["setup"]) == "" {
				t.Errorf("%s template passes analyzer %q but no setup input installs it — the report job cannot run", tt.name, tt.analyzer)
			}
			secrets, ok := report["secrets"].(map[string]any)
			if !ok || secrets["opencode-config"] != "${{ secrets.OPENCODE_CONFIG_CONTENT }}" {
				t.Errorf("%s secrets = %v, want opencode-config from OPENCODE_CONFIG_CONTENT", tt.name, report["secrets"])
			}

			// Raw-text contract: SHA-pinned actions and the comment-only
			// threshold lines, mirroring the Go caller workflow — plus
			// variant isolation from setup-go.
			rawChecks := []string{
				"actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683",
				"actions/upload-artifact@ea165f8d65b6e75b540449e92b4886f43607fa02",
				"# max-crapload: <N>",
				"# max-gaze-crapload: <N>",
				"# min-contract-coverage: <N>",
			}
			for _, want := range rawChecks {
				if !strings.Contains(rendered, want) {
					t.Errorf("%s rendered workflow missing %q", tt.name, want)
				}
			}
			if strings.Contains(rendered, "actions/setup-go") {
				t.Errorf("%s rendered workflow must not use setup-go", tt.name)
			}
		})
	}
}

// TestCIWorkflowTemplate_VariantPinParity pins the Node/Python
// templates' copied constants to the reusable workflow they call
// (review finding, ci-external-analyzer-support): the setup-node and
// setup-python action pins embedded in the templates must match the
// reusable workflow's own steps, and all three templates must agree on
// the reusable-workflow uses: reference. Golden files alone would
// happily record stale pins, so a rotation or rename applied to one
// side must fail here rather than diverge silently.
func TestCIWorkflowTemplate_VariantPinParity(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "reusable_gaze_ai_report.yml"))
	if err != nil {
		t.Fatalf("reading reusable workflow: %v", err)
	}
	var wf struct {
		Jobs struct {
			Report struct {
				Steps []struct {
					Name string `yaml:"name"`
					Uses string `yaml:"uses"`
				} `yaml:"steps"`
			} `yaml:"gaze-ai-report"`
		} `yaml:"jobs"`
	}
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parsing reusable workflow: %v", err)
	}
	reusablePins := map[string]string{}
	for _, step := range wf.Jobs.Report.Steps {
		reusablePins[step.Name] = step.Uses
	}

	wantUses := "unbound-force/gaze/.github/workflows/reusable_gaze_ai_report.yml@" + gazeVersionPlaceholder
	templates := []struct {
		name      string
		path      string
		setupStep string
	}{
		{name: "node", path: ciWorkflowNodeTemplatePath, setupStep: "Set up Node"},
		{name: "python", path: ciWorkflowPythonTemplatePath, setupStep: "Set up Python"},
	}
	for _, tt := range templates {
		t.Run(tt.name, func(t *testing.T) {
			template, err := assets.ReadFile(tt.path)
			if err != nil {
				t.Fatalf("reading embedded template: %v", err)
			}
			s := string(template)
			if !strings.Contains(s, wantUses) {
				t.Errorf("%s template must reference the reusable workflow via %q", tt.name, wantUses)
			}
			wantPin, ok := reusablePins[tt.setupStep]
			if !ok || wantPin == "" {
				t.Fatalf("reusable workflow has no %q step with a uses: pin", tt.setupStep)
			}
			if !strings.Contains(s, wantPin) {
				t.Errorf("%s template's %s pin %q does not match the reusable workflow's — rotate both together", tt.name, tt.setupStep, wantPin)
			}
		})
	}

	goTemplate, err := assets.ReadFile(ciWorkflowTemplatePath)
	if err != nil {
		t.Fatalf("reading Go template: %v", err)
	}
	if !strings.Contains(string(goTemplate), wantUses) {
		t.Error("Go template must reference the reusable workflow via the same placeholder pin")
	}
}

// TestRun_CI_VariantScaffolding verifies end-to-end variant
// scaffolding (ci-external-analyzer-support): a package.json-only
// directory gets the Node caller workflow and the analyzer-aware CI
// next steps, a pyproject.toml-only directory gets the Python caller,
// and a directory carrying both go.mod and package.json scaffolds the
// Go workflow — go.mod keeps precedence.
func TestRun_CI_VariantScaffolding(t *testing.T) {
	writeMarker := func(t *testing.T, dir, name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("creating %s: %v", name, err)
		}
	}

	t.Run("node project scaffolds the node workflow", func(t *testing.T) {
		dir := t.TempDir()
		writeMarker(t, dir, "package.json", "{\"name\":\"demo\"}\n")

		var buf bytes.Buffer
		result, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf})
		if err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}
		if len(result.Created) != 10 {
			t.Fatalf("expected 10 created files (9 .opencode/ + workflow), got %d: %v", len(result.Created), result.Created)
		}
		if !containsString(result.Created, ciWorkflowPath) {
			t.Errorf("expected %s in Created, got: %v", ciWorkflowPath, result.Created)
		}

		content, err := os.ReadFile(ciWorkflowDiskPath(dir))
		if err != nil {
			t.Fatalf("reading generated workflow: %v", err)
		}
		s := string(content)
		for _, want := range []string{
			"analyzer: gaze-analyzer-typescript",
			"packages: ./src/...",
			"node-version: 24",
			"setup: npm ci && npm run build && npm link",
			"cp -r coverage .gaze-artifact/",
			"reusable_gaze_ai_report.yml@v2.4.0",
		} {
			if !strings.Contains(s, want) {
				t.Errorf("node workflow missing %q:\n%s", want, s)
			}
		}
		if strings.Contains(s, "actions/setup-go") {
			t.Error("node workflow must not use setup-go")
		}

		output := buf.String()
		if strings.Contains(output, "Warning: skipping CI scaffolding") {
			t.Errorf("node project must not print the CI-skip warning, got:\n%s", output)
		}
		if !strings.Contains(output, "gaze report ./src/... --analyzer=gaze-analyzer-typescript --format=json") {
			t.Errorf("CI next steps must carry the analyzer report command, got:\n%s", output)
		}
	})

	t.Run("python project scaffolds the python workflow", func(t *testing.T) {
		dir := t.TempDir()
		writeMarker(t, dir, "pyproject.toml", "[project]\nname = \"demo\"\n")

		var buf bytes.Buffer
		if _, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf}); err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		content, err := os.ReadFile(ciWorkflowDiskPath(dir))
		if err != nil {
			t.Fatalf("reading generated workflow: %v", err)
		}
		s := string(content)
		for _, want := range []string{
			"analyzer: gaze-analyzer-python",
			"python-version: 3.12",
			"setup: pip install -e .",
			"pytest --cov",
		} {
			if !strings.Contains(s, want) {
				t.Errorf("python workflow missing %q:\n%s", want, s)
			}
		}
		if strings.Contains(s, "actions/setup-go") || strings.Contains(s, "gaze-analyzer-typescript") {
			t.Error("python workflow must not reference setup-go or the TypeScript analyzer")
		}
		if output := buf.String(); !strings.Contains(output, "gaze report ./src/... --analyzer=gaze-analyzer-python --format=json") {
			t.Errorf("CI next steps must carry the analyzer report command, got:\n%s", output)
		}
	})

	t.Run("go.mod wins over package.json", func(t *testing.T) {
		dir := t.TempDir()
		writeGoMod(t, dir, "go 1.24")
		writeMarker(t, dir, "package.json", "{\"name\":\"demo\"}\n")

		var buf bytes.Buffer
		if _, err := Run(Options{TargetDir: dir, Version: "v2.4.0", CI: true, Stdout: &buf}); err != nil {
			t.Fatalf("Run() returned error: %v", err)
		}

		content, err := os.ReadFile(ciWorkflowDiskPath(dir))
		if err != nil {
			t.Fatalf("reading generated workflow: %v", err)
		}
		s := string(content)
		if !strings.Contains(s, "actions/setup-go") || !strings.Contains(s, "go-version: \"1.24\"") {
			t.Errorf("go.mod must win: workflow missing setup-go with go-version \"1.24\":\n%s", s)
		}
		if strings.Contains(s, "analyzer:") {
			t.Errorf("go workflow must not pass an analyzer input:\n%s", s)
		}
	})
}
