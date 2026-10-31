package scaffold

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// CI-scaffolding tests for tasks 3.1–3.6 of the init-ci-scaffold
// change: version resolution, go.mod directive parsing, template
// rendering (golden + YAML structure), skip-if-present semantics,
// flag gating, and the CI next-steps section.

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
func TestRenderCIWorkflowTemplate_GoldenFile(t *testing.T) {
	template, err := assets.ReadFile(ciWorkflowTemplatePath)
	if err != nil {
		t.Fatalf("reading embedded CI workflow template: %v", err)
	}

	rendered := renderCIWorkflowTemplate(template, goldenGazeVersion, goldenGoVersion)

	golden, err := os.ReadFile(filepath.Join("testdata", "golden-gaze-ai-report.yml"))
	if err != nil {
		t.Fatalf("reading golden file: %v", err)
	}
	if !bytes.Equal(rendered, golden) {
		t.Errorf("rendered workflow does not match golden file.\nRun: go run ./cmd/gaze init --ci # in a temp module, or regenerate the golden from the template\n--- rendered ---\n%s\n--- golden ---\n%s", rendered, golden)
	}

	for _, placeholder := range []string{gazeVersionPlaceholder, goVersionPlaceholder} {
		if strings.Contains(string(rendered), placeholder) {
			t.Errorf("rendered workflow still contains placeholder %s", placeholder)
		}
	}
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

	rendered := string(renderCIWorkflowTemplate(template, goldenGazeVersion, goldenGoVersion))

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
