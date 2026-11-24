package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// These tests pin the learnings-only PR skip wiring (design D8,
// CIW-FR-006 of skip-ci-learnings-only-prs): a `changes` job in both CI
// workflows classifies each PR, the heavy jobs gate on it, ci-ok
// aggregates the gated jobs for branch protection, and the duplicated
// classify step cannot drift between the two workflows.

type ciWorkflow struct {
	Jobs map[string]ciJob `yaml:"jobs"`
}

// ciJob decodes only the job fields the wiring contract pins.
type ciJob struct {
	Name        string            `yaml:"name"`
	Needs       ciNeeds           `yaml:"needs"`
	If          string            `yaml:"if"`
	Timeout     int               `yaml:"timeout-minutes"`
	Permissions map[string]string `yaml:"permissions"`
	Outputs     map[string]string `yaml:"outputs"`
	Steps       []ciStep          `yaml:"steps"`
}

// ciStep decodes only the step fields the wiring contract pins.
type ciStep struct {
	Name string            `yaml:"name"`
	ID   string            `yaml:"id"`
	Uses string            `yaml:"uses"`
	With map[string]any    `yaml:"with"`
	Env  map[string]string `yaml:"env"`
	Run  string            `yaml:"run"`
}

// ciNeeds accepts `needs:` in both spellings GitHub Actions allows: a
// single job name or a YAML list.
type ciNeeds []string

// UnmarshalYAML decodes the scalar and sequence forms of `needs:`.
func (n *ciNeeds) UnmarshalYAML(node *yaml.Node) error {
	if node.Tag == "!!null" {
		*n = nil
		return nil
	}
	if node.Kind == yaml.ScalarNode {
		*n = []string{node.Value}
		return nil
	}
	var list []string
	if err := node.Decode(&list); err != nil {
		return fmt.Errorf("needs: %w", err)
	}
	*n = list
	return nil
}

// loadCIWorkflow reads and parses a workflow under .github/workflows.
func loadCIWorkflow(t *testing.T, filename string) ciWorkflow {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", filename))
	if err != nil {
		t.Fatalf("read %s: %v", filename, err)
	}
	var wf ciWorkflow
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	return wf
}

// ciChangesJob returns the changes job, failing when it is absent.
func ciChangesJob(t *testing.T, filename string) ciJob {
	t.Helper()
	job, ok := loadCIWorkflow(t, filename).Jobs["changes"]
	if !ok {
		t.Fatalf("%s must contain the changes job", filename)
	}
	return job
}

// classifyRun returns the run script of the changes job's classify step.
func classifyRun(t *testing.T, filename string, job ciJob) string {
	t.Helper()
	for _, step := range job.Steps {
		if step.ID == "classify" {
			if step.Run == "" {
				t.Fatalf("%s classify step has an empty run", filename)
			}
			return step.Run
		}
	}
	t.Fatalf("%s changes job must contain a step with id classify", filename)
	return ""
}

// ciNeedsJob reports whether the job declares the given dependency.
func ciNeedsJob(job ciJob, dep string) bool {
	for _, need := range job.Needs {
		if need == dep {
			return true
		}
	}
	return false
}

// ciRunStep returns the sole step of a job that carries a run script,
// failing when the job has none or more than one.
func ciRunStep(t *testing.T, filename, jobID string) ciStep {
	t.Helper()
	job, ok := loadCIWorkflow(t, filename).Jobs[jobID]
	if !ok {
		t.Fatalf("%s must contain the %s job", filename, jobID)
	}
	var runs []ciStep
	for _, step := range job.Steps {
		if step.Run != "" {
			runs = append(runs, step)
		}
	}
	if len(runs) != 1 {
		t.Fatalf("%s %s must carry exactly one run step, found %d", filename, jobID, len(runs))
	}
	return runs[0]
}

// flattenRun collapses a run script's line continuations and whitespace
// runs into single spaces so a multi-line expression (the ci-ok jq
// pipeline) can be pinned as one substring.
func flattenRun(run string) string {
	continuations := regexp.MustCompile(`\\\s+`)
	spaces := regexp.MustCompile(`\s+`)
	return spaces.ReplaceAllString(continuations.ReplaceAllString(run, " "), " ")
}

// The changes job classifies each PR as code or learnings-only. Both CI
// workflows carry their own copy so their gated jobs can skip without a
// cross-workflow needs, and the checkout needs full history because the
// classifier diffs base...head itself.
func TestCIWorkflow_ChangesJobWiring(t *testing.T) {
	for _, filename := range []string{"test.yml", "mega-linter.yml"} {
		job := ciChangesJob(t, filename)

		if job.Name != "Changes" {
			t.Errorf("%s changes name = %q, want %q", filename, job.Name, "Changes")
		}
		if job.Timeout != 5 {
			t.Errorf("%s changes timeout-minutes = %d, want 5", filename, job.Timeout)
		}
		if len(job.Permissions) != 1 || job.Permissions["contents"] != "read" {
			t.Errorf("%s changes permissions = %v, want exactly contents: read", filename, job.Permissions)
		}
		wantOutput := "${{ steps.classify.outputs.code }}"
		if len(job.Outputs) != 1 || job.Outputs["code"] != wantOutput {
			t.Errorf("%s changes outputs = %v, want exactly code: %q", filename, job.Outputs, wantOutput)
		}

		var checkout *ciStep
		for i, step := range job.Steps {
			if strings.HasPrefix(step.Uses, "actions/checkout@") {
				checkout = &job.Steps[i]
				break
			}
		}
		if checkout == nil {
			t.Fatalf("%s changes job must check out the repo", filename)
		}
		const checkoutPin = "actions/checkout@11bd71901bbe5b1630ceea73d27597364c9af683"
		if checkout.Uses != checkoutPin {
			t.Errorf("%s changes checkout uses %q, want the exact SHA pin %q — a mutable tag would silently change what the classifier runs on (CI-001; precedent: release_workflow_test.go)", filename, checkout.Uses, checkoutPin)
		}
		fd, ok := checkout.With["fetch-depth"]
		if !ok || fmt.Sprintf("%v", fd) != "0" {
			t.Errorf("%s changes checkout fetch-depth = %v (present=%t), want 0", filename, fd, ok)
		}
	}
}

// unit-and-integration, e2e, and megalinter must skip on learnings-only
// PRs: each needs changes and carries exactly the != 'false' gate.
func TestCIWorkflow_GatedJobsSkipOnLearningsOnly(t *testing.T) {
	gated := []struct {
		workflow string
		job      string
	}{
		{"test.yml", "unit-and-integration"},
		{"test.yml", "e2e"},
		{"mega-linter.yml", "megalinter"},
	}
	wantIf := "needs.changes.outputs.code != 'false'"
	for _, g := range gated {
		job, ok := loadCIWorkflow(t, g.workflow).Jobs[g.job]
		if !ok {
			t.Errorf("%s must contain the %s job", g.workflow, g.job)
			continue
		}
		found := false
		for _, need := range job.Needs {
			if need == "changes" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("%s %s needs = %v, must include changes", g.workflow, g.job, job.Needs)
		}
		if job.If != wantIf {
			t.Errorf("%s %s if = %q, want %q", g.workflow, g.job, job.If, wantIf)
		}
	}
}

// ci-ok aggregates the gated jobs so branch protection can require one
// check name. It runs on !cancelled() so a failed or skipped dependency
// still reports instead of hanging the merge box.
func TestCIWorkflow_CIOKAggregatesGatedJobs(t *testing.T) {
	job, ok := loadCIWorkflow(t, "test.yml").Jobs["ci-ok"]
	if !ok {
		t.Fatal("test.yml must contain the ci-ok job")
	}
	if job.Name != "CI OK" {
		t.Errorf("ci-ok name = %q, want %q", job.Name, "CI OK")
	}
	if job.If != "${{ !cancelled() }}" {
		t.Errorf("ci-ok if = %q, want %q", job.If, "${{ !cancelled() }}")
	}
	if job.Timeout != 5 {
		t.Errorf("ci-ok timeout-minutes = %d, want 5", job.Timeout)
	}
	got := append([]string(nil), job.Needs...)
	sort.Strings(got)
	want := []string{"changes", "e2e", "unit-and-integration"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ci-ok needs = %v, want exactly %v", got, want)
	}
}

// Drift guard: a new test.yml job must make an explicit gate decision —
// join ci-ok's needs or be declared advisory. The test fails naming the
// job so the decision cannot be made by accident.
func TestCIWorkflow_EveryJobHasGateDecision(t *testing.T) {
	wf := loadCIWorkflow(t, "test.yml")
	ciOK, ok := wf.Jobs["ci-ok"]
	if !ok {
		t.Fatal("test.yml must contain the ci-ok job")
	}
	// Advisory jobs are deliberately outside ci-ok's needs: a provider
	// outage or rejected narrative cannot block a merge.
	advisory := map[string]bool{"gaze-ai-report": true}
	inNeeds := make(map[string]bool, len(ciOK.Needs))
	for _, need := range ciOK.Needs {
		inNeeds[need] = true
	}
	for name := range wf.Jobs {
		if name == "ci-ok" {
			continue
		}
		needed, allowed := inNeeds[name], advisory[name]
		switch {
		case needed && allowed:
			t.Errorf("job %q is on the advisory allowlist and must not be in ci-ok.needs", name)
		case !needed && !allowed:
			t.Errorf("job %q is neither in ci-ok.needs nor on the advisory allowlist [gaze-ai-report]; make an explicit gate decision for it", name)
		}
	}
}

// The classify step is duplicated so each workflow stays self-contained;
// any change must be applied to both copies in lockstep.
func TestCIWorkflow_ClassifyStepIdenticalAcrossWorkflows(t *testing.T) {
	testRun := classifyRun(t, "test.yml", ciChangesJob(t, "test.yml"))
	linterRun := classifyRun(t, "mega-linter.yml", ciChangesJob(t, "mega-linter.yml"))
	if testRun != linterRun {
		t.Errorf("classify step run differs between test.yml and mega-linter.yml:\n--- test.yml ---\n%s\n--- mega-linter.yml ---\n%s", testRun, linterRun)
	}
}

// A code file renamed into .uf/dewey/learnings/ must classify as code, not
// as a learnings-only change (CIW-FR-001 rename scenario, design D1): with
// rename collapsing disabled the name-only diff lists both the old and the
// new path, so the pre-rename code path still falls outside the learnings
// prefix. Both diff invocations — the --name-only and the --raw one — must
// carry --no-renames and diff the three-dot base...head range so the
// classification is the PR's net diff from its merge base.
func TestCIWorkflow_RenameIntoLearningsPathCountsAsCode(t *testing.T) {
	const threeDotRange = `"${base}...${head}"`
	for _, filename := range []string{"test.yml", "mega-linter.yml"} {
		run := classifyRun(t, filename, ciChangesJob(t, filename))
		for _, mode := range []string{"--name-only", "--raw"} {
			var diffLines []string
			for _, line := range strings.Split(run, "\n") {
				if strings.Contains(line, "git diff "+mode) {
					diffLines = append(diffLines, line)
				}
			}
			if len(diffLines) == 0 {
				t.Errorf("%s changes classify: no `git diff %s` invocation found in the run script", filename, mode)
				continue
			}
			for _, line := range diffLines {
				if !strings.Contains(line, "--no-renames") {
					t.Errorf("%s changes classify: `git diff %s` lacks --no-renames, so a code file renamed into .uf/dewey/learnings/ collapses to a single learnings path and skips CI: %q", filename, mode, strings.TrimSpace(line))
				}
				if !strings.Contains(line, threeDotRange) {
					t.Errorf("%s changes classify: `git diff %s` lacks the three-dot range %s, so the classification is not the PR's net diff from the merge base: %q", filename, mode, threeDotRange, strings.TrimSpace(line))
				}
			}
		}
	}
}

// A symlink under .uf/dewey/learnings/ must classify as code (CIW-FR-001
// symlink scenario, design D1): the learnings path prefix is a
// classification signal, not a trust boundary, so the classifier scans the
// raw diff for mode 120000 on either side of each change and feeds the
// symlink list into the fail-closed code=true verdict. The code=true
// branch must also label and print both the outside-file and symlink
// lists — CIW-FR-001's log-listing MUST — so those echo lines are pinned
// here too.
func TestCIWorkflow_SymlinkUnderLearningsCountsAsCode(t *testing.T) {
	for _, filename := range []string{"test.yml", "mega-linter.yml"} {
		run := classifyRun(t, filename, ciChangesJob(t, filename))
		if !strings.Contains(run, `printf '%s\n' "$raw"`) {
			t.Errorf("%s changes classify: the symlink scan must read the captured git diff --raw output (missing %q)", filename, `printf '%s\n' "$raw"`)
		}
		const modeCheck = `$1 ~ /120000/ || $2 ~ /120000/`
		if !strings.Contains(run, modeCheck) {
			t.Errorf("%s changes classify: the raw-diff scan must match symlink mode 120000 on either the old or the new side of a change (missing %q)", filename, modeCheck)
		}
		const linksInVerdict = `[ -n "$links" ]`
		if !strings.Contains(run, linksInVerdict) {
			t.Errorf("%s changes classify: the fail-closed code=true verdict must include the symlink list (missing %q), or a learnings-path symlink would classify as learnings-only", filename, linksInVerdict)
		}
		const outsideLabel = `echo "files outside .uf/dewey/learnings/:"`
		if !strings.Contains(run, outsideLabel) {
			t.Errorf("%s changes classify: the code=true branch must label the outside-file list (missing %q), per CIW-FR-001's log-listing MUST", filename, outsideLabel)
		}
		const outsideValue = `echo "$outside"`
		if !strings.Contains(run, outsideValue) {
			t.Errorf("%s changes classify: the code=true branch must log the outside-file list (missing %q), per CIW-FR-001's log-listing MUST", filename, outsideValue)
		}
		const linksLabel = `echo "symlinks:"`
		if !strings.Contains(run, linksLabel) {
			t.Errorf("%s changes classify: the code=true branch must label the symlink list (missing %q), per CIW-FR-001's log-listing MUST", filename, linksLabel)
		}
		const linksValue = `echo "$links"`
		if !strings.Contains(run, linksValue) {
			t.Errorf("%s changes classify: the code=true branch must log the symlink list (missing %q), per CIW-FR-001's log-listing MUST", filename, linksValue)
		}
	}
}

// A failing git diff must fail the changes job instead of being masked by
// the filtering pipelines (CIW-FR-001 diff-failure scenario, design D1):
// the classify script starts with set -euo pipefail and every git diff
// invocation is a standalone command substitution captured into a
// variable — no inline pipes or `|| true` — so a non-zero git exit
// propagates before any grep or awk runs.
func TestCIWorkflow_DiffFailureFailsClosed(t *testing.T) {
	capturedDiff := regexp.MustCompile(`^\s*\w+="\$\(\s*git diff [^)|]*\)"\s*$`)
	for _, filename := range []string{"test.yml", "mega-linter.yml"} {
		run := classifyRun(t, filename, ciChangesJob(t, filename))
		lines := strings.Split(run, "\n")
		if got := strings.TrimSpace(lines[0]); got != "set -euo pipefail" {
			t.Errorf("%s changes classify: first line = %q, want %q — without strict mode a failing git diff cannot fail the job", filename, got, "set -euo pipefail")
		}
		diffLines := 0
		for _, line := range lines {
			if !strings.Contains(line, "git diff") {
				continue
			}
			diffLines++
			if !capturedDiff.MatchString(line) {
				t.Errorf("%s changes classify: git diff must be captured into a variable before filtering, not piped or masked by a trailing || true (found: %q)", filename, strings.TrimSpace(line))
			}
		}
		if diffLines == 0 {
			t.Errorf("%s changes classify: the run script must invoke git diff at least once", filename)
		}
	}
}

// CI OK must fail when any needed job concluded failure or cancelled, and
// its log must name the job and its result (CIW-FR-003, design D3): the
// aggregate step reads needs as JSON from an env var, prints one
// `job: result` line per need, then enforces with jq -e that every result
// is success or skipped — anything else fails the job.
func TestCIWorkflow_CIOKFailsOnFailedOrCancelledNeed(t *testing.T) {
	step := ciRunStep(t, "test.yml", "ci-ok")
	if got := step.Env["NEEDS"]; got != "${{ toJSON(needs) }}" {
		t.Errorf("test.yml ci-ok run step env NEEDS = %q, want %q — needs JSON must arrive via an env var, not as script text", got, "${{ toJSON(needs) }}")
	}
	flat := flattenRun(step.Run)
	perNeedLine := `printf '%s\n' "$NEEDS" | jq -r 'to_entries[] | "\(.key): \(.value.result)"'`
	if !strings.Contains(flat, perNeedLine) {
		t.Errorf("test.yml ci-ok run step must print one `job: result` line per needed job before deciding (missing %q)", perNeedLine)
	}
	passPredicate := `jq -e 'to_entries | all(.value.result == "success" or .value.result == "skipped")'`
	if !strings.Contains(flat, passPredicate) {
		t.Errorf("test.yml ci-ok run step must enforce with jq -e that every needed job concluded success or skipped (missing %q)", passPredicate)
	}
	if printAt, verdictAt := strings.Index(flat, perNeedLine), strings.Index(flat, passPredicate); printAt >= 0 && verdictAt >= 0 && printAt > verdictAt {
		t.Errorf("test.yml ci-ok run step must print the per-need `job: result` lines before the jq -e verdict, so a red CI OK names the job that failed it")
	}
}

// A failed changes job must block the PR through CI OK (CIW-FR-003
// classification-failure scenario, design D2/D3): the gated jobs need
// changes so they are skipped when the classifier fails, ci-ok lists
// changes in its needs, and ci-ok runs on !cancelled() — never always(),
// which would convert a concurrency-cancelled run into a failed CI OK —
// so the changes failure is aggregated into a red CI OK instead of
// leaving the merge box waiting on a check that never reports.
func TestCIWorkflow_ClassificationFailureBlocksPR(t *testing.T) {
	wf := loadCIWorkflow(t, "test.yml")
	for _, jobID := range []string{"unit-and-integration", "e2e"} {
		job, ok := wf.Jobs[jobID]
		if !ok {
			t.Errorf("test.yml must contain the %s job", jobID)
			continue
		}
		if !ciNeedsJob(job, "changes") {
			t.Errorf("test.yml %s needs = %v, must include changes so a failed classifier skips (never runs) the heavy jobs", jobID, job.Needs)
		}
	}
	ciOK, ok := wf.Jobs["ci-ok"]
	if !ok {
		t.Fatal("test.yml must contain the ci-ok job")
	}
	if !ciNeedsJob(ciOK, "changes") {
		t.Errorf("test.yml ci-ok needs = %v, must include changes so a failed classifier fails CI OK", ciOK.Needs)
	}
	if ciOK.If != "${{ !cancelled() }}" {
		t.Errorf("test.yml ci-ok if = %q, want %q — with the default success() condition a failed changes job would skip CI OK and the PR would hang", ciOK.If, "${{ !cancelled() }}")
	}
	if strings.Contains(ciOK.If, "always()") {
		t.Errorf("test.yml ci-ok if = %q must not use always(), which converts a concurrency-cancelled run into a failed CI OK (design D3)", ciOK.If)
	}
}

// A changed path carrying a `..` segment must classify as code —
// defense-in-depth for the CIW-FR-001 learnings prefix check: git never
// emits `..` path segments in a diff, so a hypothetical traversal path
// (`.uf/dewey/learnings/../escape.go`) bypassing the prefix check cannot
// be exercised through a real git repository and the pin is necessarily
// textual, mirroring TestCIWorkflow_SymlinkUnderLearningsCountsAsCode.
// The classifier computes a dotdot list from the name-only diff, feeds it
// into the fail-closed code=true verdict, and logs it.
func TestCIWorkflow_DotdotSegmentFailsClosed(t *testing.T) {
	for _, filename := range []string{"test.yml", "mega-linter.yml"} {
		run := classifyRun(t, filename, ciChangesJob(t, filename))
		const nameOnlyFeed = `dotdot="$(printf '%s\n' "$files"`
		if !strings.Contains(run, nameOnlyFeed) {
			t.Errorf("%s changes classify: the dotdot scan must read the captured name-only diff, not the raw one (missing %q)", filename, nameOnlyFeed)
		}
		const dotdotScan = `grep -E '(^|/)\.\.(/|$)' || true`
		if !strings.Contains(run, dotdotScan) {
			t.Errorf("%s changes classify: the name-only diff must be scanned for `..` path segments (missing %q)", filename, dotdotScan)
		}
		const dotdotInVerdict = `[ -n "$dotdot" ]`
		if !strings.Contains(run, dotdotInVerdict) {
			t.Errorf("%s changes classify: the fail-closed code=true verdict must include the dotdot list (missing %q), or a `..` path would bypass the learnings prefix check", filename, dotdotInVerdict)
		}
		const dotdotLabel = `echo "path traversal segments:"`
		if !strings.Contains(run, dotdotLabel) {
			t.Errorf("%s changes classify: the code=true branch must label the traversal findings (missing %q)", filename, dotdotLabel)
		}
		const dotdotValue = `echo "$dotdot"`
		if !strings.Contains(run, dotdotValue) {
			t.Errorf("%s changes classify: the code=true branch must log the dotdot list (missing %q)", filename, dotdotValue)
		}
	}
}

// ciGit runs a git command in dir, failing the test on a non-zero exit.
func ciGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	var combined strings.Builder
	cmd.Stdout = &combined
	cmd.Stderr = &combined
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s in %s: %v\n%s", strings.Join(args, " "), dir, err, combined.String())
	}
}

// ciGitHead returns the HEAD commit SHA of the repository in dir.
func ciGitHead(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	var out strings.Builder
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		t.Fatalf("git rev-parse HEAD in %s: %v", dir, err)
	}
	return strings.TrimSpace(out.String())
}

// initClassifyRepo creates a temporary git repository whose sole base
// commit adds main.go, returning the repository directory and the base
// commit SHA for the classify script's diff range. Commit signing is
// disabled so a globally configured gpgsign cannot break the fixtures.
func initClassifyRepo(t *testing.T) (dir, base string) {
	t.Helper()
	dir = t.TempDir()
	ciGit(t, dir, "init", "-q")
	ciGit(t, dir, "config", "user.email", "gaze@example.com")
	ciGit(t, dir, "config", "user.name", "Gaze Test")
	ciGit(t, dir, "config", "commit.gpgsign", "false")
	writeClassifyFile(t, dir, "main.go", "package main\n")
	ciGit(t, dir, "add", "-A")
	ciGit(t, dir, "commit", "-q", "-m", "base")
	return dir, ciGitHead(t, dir)
}

// writeClassifyFile writes content at the repo-relative path inside dir,
// creating parent directories as needed.
func writeClassifyFile(t *testing.T, dir, path, content string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(path))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", filepath.Dir(full), err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", full, err)
	}
}

// readClassifyOutput reads the captured GITHUB_OUTPUT file; when the
// script died before writing anything it degrades to a placeholder so the
// original failure is not masked by a read error.
func readClassifyOutput(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return fmt.Sprintf("(no GITHUB_OUTPUT written: %v)", err)
	}
	return string(raw)
}

// The pins above check the classify script's text; this test executes the
// actual script extracted from test.yml — GitHub expression placeholders
// substituted — against real temporary git repositories, so the CIW-FR-001
// classification behavior is pinned end to end: a learnings-only diff
// classifies code=false, while mixed, renamed-into-learnings, and symlink
// diffs, the empty diff, and push events (CIW-FR-005) all fail closed to
// code=true (design D1).
func TestCIWorkflow_ClassifyScriptBehavior(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("skipping: bash not found in PATH")
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("skipping: git not found in PATH")
	}

	// commitLearningsOnly builds the learnings-only head commit shared by
	// the skip scenario and the push scenario (which must classify
	// code=true regardless of the diff).
	commitLearningsOnly := func(t *testing.T, dir, base string) string {
		t.Helper()
		writeClassifyFile(t, dir, ".uf/dewey/learnings/a.md", "# learning\n")
		ciGit(t, dir, "add", "-A")
		ciGit(t, dir, "commit", "-q", "-m", "learnings-only")
		return ciGitHead(t, dir)
	}

	scenarios := []struct {
		name      string
		eventName string // empty means pull_request
		head      func(t *testing.T, dir, base string) string
		want      string
		wantLog   []string // strings the script's stdout must contain
	}{
		{
			name:    "learnings-only pull request",
			head:    commitLearningsOnly,
			want:    "false",
			wantLog: []string{"learnings-only pull request"},
		},
		{
			name: "mixed learnings and code pull request",
			head: func(t *testing.T, dir, base string) string {
				writeClassifyFile(t, dir, ".uf/dewey/learnings/a.md", "# learning\n")
				writeClassifyFile(t, dir, "main.go", "package main\n\nfunc extra() {}\n")
				ciGit(t, dir, "add", "-A")
				ciGit(t, dir, "commit", "-q", "-m", "mixed")
				return ciGitHead(t, dir)
			},
			want:    "true",
			wantLog: []string{"files outside .uf/dewey/learnings/:", "main.go"},
		},
		{
			name: "code file renamed into learnings",
			head: func(t *testing.T, dir, base string) string {
				// git mv needs the destination directory to exist.
				if err := os.MkdirAll(filepath.Join(dir, ".uf", "dewey", "learnings"), 0o755); err != nil {
					t.Fatalf("mkdir learnings dir: %v", err)
				}
				ciGit(t, dir, "mv", "main.go", ".uf/dewey/learnings/main.go")
				ciGit(t, dir, "commit", "-q", "-m", "rename into learnings")
				return ciGitHead(t, dir)
			},
			want: "true",
		},
		{
			name: "symlink under learnings",
			head: func(t *testing.T, dir, base string) string {
				learnings := filepath.Join(dir, ".uf", "dewey", "learnings")
				if err := os.MkdirAll(learnings, 0o755); err != nil {
					t.Fatalf("mkdir %s: %v", learnings, err)
				}
				link := filepath.Join(learnings, "link")
				if err := os.Symlink("main.go", link); err != nil {
					t.Fatalf("symlink %s: %v", link, err)
				}
				ciGit(t, dir, "add", "-A")
				ciGit(t, dir, "commit", "-q", "-m", "symlink under learnings")
				return ciGitHead(t, dir)
			},
			want:    "true",
			wantLog: []string{"symlinks:", ".uf/dewey/learnings/link"},
		},
		{
			name: "empty diff fails closed",
			head: func(t *testing.T, dir, base string) string {
				return base // no head commit: the base...head diff is empty
			},
			want: "true",
		},
		{
			name:      "push event fails closed",
			eventName: "push",
			head:      commitLearningsOnly,
			want:      "true",
		},
	}

	scriptTemplate := classifyRun(t, "test.yml", ciChangesJob(t, "test.yml"))
	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			dir, base := initClassifyRepo(t)
			head := sc.head(t, dir, base)
			event := sc.eventName
			if event == "" {
				event = "pull_request"
			}
			script := scriptTemplate
			script = strings.ReplaceAll(script, "${{ github.event_name }}", event)
			script = strings.ReplaceAll(script, "${{ github.event.pull_request.base.sha }}", base)
			script = strings.ReplaceAll(script, "${{ github.event.pull_request.head.sha }}", head)
			if strings.Contains(script, "${{") {
				t.Fatalf("%s: classify script still carries an unsubstituted GitHub expression:\n%s", sc.name, script)
			}

			outPath := filepath.Join(t.TempDir(), "github-output")
			var stdout, stderr strings.Builder
			cmd := exec.Command("bash", "-c", script)
			cmd.Dir = dir
			cmd.Env = append(os.Environ(), "GITHUB_OUTPUT="+outPath)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr
			if err := cmd.Run(); err != nil {
				t.Fatalf("%s: classify script must exit 0, got %v\nstdout:\n%s\nstderr:\n%s\nGITHUB_OUTPUT:\n%s",
					sc.name, err, stdout.String(), stderr.String(), readClassifyOutput(outPath))
			}
			out := readClassifyOutput(outPath)
			var got string
			for _, line := range strings.Split(out, "\n") {
				if strings.HasPrefix(line, "code=") {
					got = strings.TrimPrefix(line, "code=")
				}
			}
			if got == "" {
				t.Fatalf("%s: GITHUB_OUTPUT must contain a code= line\nstdout:\n%s\nstderr:\n%s\nGITHUB_OUTPUT:\n%s",
					sc.name, stdout.String(), stderr.String(), out)
			}
			if got != sc.want {
				t.Errorf("%s: classified code=%s, want code=%s\nstdout:\n%s\nstderr:\n%s\nGITHUB_OUTPUT:\n%s",
					sc.name, got, sc.want, stdout.String(), stderr.String(), out)
			}
			// CIW-FR-001's log-listing MUST: the job log must name the
			// files and symlinks behind a code=true verdict, so stdout is
			// asserted, not just captured for failure diagnostics.
			for _, want := range sc.wantLog {
				if !strings.Contains(stdout.String(), want) {
					t.Errorf("%s: classify log must contain %q\nstdout:\n%s\nstderr:\n%s\nGITHUB_OUTPUT:\n%s",
						sc.name, want, stdout.String(), stderr.String(), out)
				}
			}
		})
	}
}

// The ci-ok jq verdict's text is pinned by
// TestCIWorkflow_CIOKFailsOnFailedOrCancelledNeed; this test extracts the
// program from the workflow and runs it against needs-shaped JSON, so the
// verdict's truth table is pinned by execution (CIW-FR-003): every needed
// job concluding success or skipped exits 0, and any failure or cancelled
// result exits non-zero.
func TestCIWorkflow_CIOKJqTruthTable(t *testing.T) {
	if _, err := exec.LookPath("jq"); err != nil {
		t.Skip("skipping: jq not found in PATH")
	}
	flat := flattenRun(ciRunStep(t, "test.yml", "ci-ok").Run)
	match := regexp.MustCompile(`jq -e\s+'([^']+)'`).FindStringSubmatch(flat)
	if match == nil {
		t.Fatal("test.yml ci-ok run step must carry a jq -e verdict program")
	}
	program := match[1]

	rows := []struct {
		name  string
		needs string
		pass  bool
	}{
		{
			name:  "all success",
			needs: `{"changes":{"result":"success"},"unit-and-integration":{"result":"success"},"e2e":{"result":"success"}}`,
			pass:  true,
		},
		{
			name:  "mixed success and skipped",
			needs: `{"changes":{"result":"success"},"unit-and-integration":{"result":"skipped"},"e2e":{"result":"success"}}`,
			pass:  true,
		},
		{
			name:  "all skipped",
			needs: `{"changes":{"result":"skipped"},"unit-and-integration":{"result":"skipped"},"e2e":{"result":"skipped"}}`,
			pass:  true,
		},
		{
			name:  "any failure",
			needs: `{"changes":{"result":"success"},"unit-and-integration":{"result":"failure"},"e2e":{"result":"success"}}`,
			pass:  false,
		},
		{
			name:  "any cancelled",
			needs: `{"changes":{"result":"cancelled"},"unit-and-integration":{"result":"success"},"e2e":{"result":"success"}}`,
			pass:  false,
		},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			cmd := exec.Command("jq", "-e", program)
			cmd.Stdin = strings.NewReader(row.needs)
			var combined strings.Builder
			cmd.Stdout = &combined
			cmd.Stderr = &combined
			err := cmd.Run()
			if row.pass && err != nil {
				t.Errorf("%s: jq -e verdict must exit 0 for needs %s, got %v\noutput:\n%s", row.name, row.needs, err, combined.String())
			}
			if !row.pass && err == nil {
				t.Errorf("%s: jq -e verdict must exit non-zero for needs %s, but it exited 0\noutput:\n%s", row.name, row.needs, combined.String())
			}
		})
	}
}
