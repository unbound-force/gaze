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

// These tests pin the chore-PR skip wiring: a `scope` job in both CI
// workflows calls jflowers/ship's pr-scope.yml, which outputs
// code == 'false' only for Dewey learnings PRs and gh-hud toolkit
// rollouts; the heavy jobs gate on it, and ci-ok aggregates the gated
// jobs into the one check branch protection requires. The classifier
// itself is ship's, and ship tests it.

type ciWorkflow struct {
	Jobs map[string]ciJob `yaml:"jobs"`
}

// ciJob decodes only the job fields the wiring contract pins.
type ciJob struct {
	Name        string            `yaml:"name"`
	Uses        string            `yaml:"uses"`
	Needs       ciNeeds           `yaml:"needs"`
	If          string            `yaml:"if"`
	Timeout     int               `yaml:"timeout-minutes"`
	Permissions map[string]string `yaml:"permissions"`
	Outputs     map[string]string `yaml:"outputs"`
	Steps       []ciStep          `yaml:"steps"`
}

// ciStep decodes only the step fields the wiring contract pins.
type ciStep struct {
	Name  string            `yaml:"name"`
	ID    string            `yaml:"id"`
	Uses  string            `yaml:"uses"`
	With  map[string]any    `yaml:"with"`
	Env   map[string]string `yaml:"env"`
	Run   string            `yaml:"run"`
	Shell string            `yaml:"shell"`
	If    string            `yaml:"if"`
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

// shipScopePinRe pins the scope job to ship's pr-scope.yml at a full
// commit SHA (CI-001). The SHA is format-pinned, not value-pinned:
// Dependabot bumps reusable-workflow pins, and a literal-SHA assertion
// would fail CI on those bumps and block the auto-merge behind them.
var shipScopePinRe = regexp.MustCompile(`^jflowers/ship/\.github/workflows/pr-scope\.yml@[0-9a-f]{40}$`)

// Both CI workflows carry their own scope job so their gated jobs can
// skip without a cross-workflow needs. It needs only contents: read.
func TestCIWorkflow_ScopeJobWiring(t *testing.T) {
	for _, filename := range []string{"test.yml", "mega-linter.yml"} {
		job, ok := loadCIWorkflow(t, filename).Jobs["scope"]
		if !ok {
			t.Fatalf("%s must contain the scope job", filename)
		}
		if !shipScopePinRe.MatchString(job.Uses) {
			t.Errorf("%s scope uses %q, want jflowers/ship's pr-scope.yml pinned to a full 40-hex commit SHA", filename, job.Uses)
		}
		if len(job.Permissions) != 1 || job.Permissions["contents"] != "read" {
			t.Errorf("%s scope permissions = %v, want exactly contents: read", filename, job.Permissions)
		}
		if len(job.Needs) != 0 || job.If != "" {
			t.Errorf("%s scope must run unconditionally (needs = %v, if = %q)", filename, job.Needs, job.If)
		}
	}
}

// unit-and-integration, e2e, and megalinter run only when scope says
// code: each needs scope and carries exactly the == 'true' gate, so a
// failed or missing scope output skips them (and fails CI OK) instead
// of running them by default.
func TestCIWorkflow_GatedJobsSkipOnChorePRs(t *testing.T) {
	gated := []struct {
		workflow string
		job      string
	}{
		{"test.yml", "unit-and-integration"},
		{"test.yml", "e2e"},
		{"mega-linter.yml", "megalinter"},
	}
	const wantIf = "needs.scope.outputs.code == 'true'"
	for _, g := range gated {
		job, ok := loadCIWorkflow(t, g.workflow).Jobs[g.job]
		if !ok {
			t.Errorf("%s must contain the %s job", g.workflow, g.job)
			continue
		}
		if !ciNeedsJob(job, "scope") {
			t.Errorf("%s %s needs = %v, must include scope", g.workflow, g.job, job.Needs)
		}
		if job.If != wantIf {
			t.Errorf("%s %s if = %q, want %q", g.workflow, g.job, job.If, wantIf)
		}
	}
}

// ci-ok aggregates the gated jobs so branch protection can require one
// check name. It runs on always(): a failed dependency still reports,
// and a cancelled run reports a red CI OK rather than a skip, which
// would count as passing.
func TestCIWorkflow_CIOKAggregatesGatedJobs(t *testing.T) {
	job, ok := loadCIWorkflow(t, "test.yml").Jobs["ci-ok"]
	if !ok {
		t.Fatal("test.yml must contain the ci-ok job")
	}
	if job.Name != "CI OK" {
		t.Errorf("ci-ok name = %q, want %q", job.Name, "CI OK")
	}
	if job.If != "always()" {
		t.Errorf("ci-ok if = %q, want %q", job.If, "always()")
	}
	if job.Timeout != 5 {
		t.Errorf("ci-ok timeout-minutes = %d, want 5", job.Timeout)
	}
	got := append([]string(nil), job.Needs...)
	sort.Strings(got)
	want := []string{"e2e", "gitleaks", "scope", "unit-and-integration"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("ci-ok needs = %v, want exactly %v", got, want)
	}
}

// Drift guard: a new test.yml job must make an explicit gate decision —
// join ci-ok's needs or be declared advisory. The test fails naming the
// job so the decision cannot be made by accident. Advisory jobs must
// still depend on a gated job, so they inherit the chore-PR skip.
func TestCIWorkflow_EveryJobHasGateDecision(t *testing.T) {
	wf := loadCIWorkflow(t, "test.yml")
	ciOK, ok := wf.Jobs["ci-ok"]
	if !ok {
		t.Fatal("test.yml must contain the ci-ok job")
	}
	// Advisory jobs are deliberately outside ci-ok's needs: a provider
	// outage or rejected narrative cannot block a merge.
	advisory := map[string]bool{"gaze-ai-report": true}
	gated := map[string]bool{"unit-and-integration": true, "e2e": true}
	inNeeds := make(map[string]bool, len(ciOK.Needs))
	for _, need := range ciOK.Needs {
		inNeeds[need] = true
	}
	for name, job := range wf.Jobs {
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
		if allowed {
			inherits := false
			for _, need := range job.Needs {
				inherits = inherits || gated[need]
			}
			if !inherits {
				t.Errorf("advisory job %q needs = %v, must include a gated job so it skips on chore PRs", name, job.Needs)
			}
		}
	}
}

// ciOKEnv is the env the ci-ok step must read its inputs from, so the
// run script itself carries no expressions.
var ciOKEnv = map[string]string{
	"SCOPE":    "${{ needs.scope.result }}",
	"CODE":     "${{ needs.scope.outputs.code }}",
	"UNIT":     "${{ needs.unit-and-integration.result }}",
	"E2E":      "${{ needs.e2e.result }}",
	"GITLEAKS": "${{ needs.gitleaks.result }}",
}

// The CI OK verdict is pinned by execution: the run script is taken
// from the workflow and run with bash against needs-shaped env values.
// It passes only when scope succeeded and the jobs it chose concluded
// as chosen — the code jobs success and gitleaks skipped for code,
// the reverse for chores — and fails on anything else, including a
// gated job that ran (or skipped) against the scope decision.
func TestCIWorkflow_CIOKTruthTable(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil {
		t.Skip("skipping: bash not found in PATH")
	}
	step := ciRunStep(t, "test.yml", "ci-ok")
	for name, want := range ciOKEnv {
		if got := step.Env[name]; got != want {
			t.Errorf("test.yml ci-ok env %s = %q, want %q", name, got, want)
		}
	}
	if strings.Contains(step.Run, "${{") {
		t.Fatalf("test.yml ci-ok run script must read its inputs from env, not expressions:\n%s", step.Run)
	}

	rows := []struct {
		name                             string
		scope, code, unit, e2e, gitleaks string
		pass                             bool
	}{
		{"code PR passes", "success", "true", "success", "success", "skipped", true},
		{"chore PR passes", "success", "false", "skipped", "skipped", "success", true},
		{"scope failed", "failure", "", "skipped", "skipped", "skipped", false},
		{"scope cancelled", "cancelled", "", "skipped", "skipped", "skipped", false},
		{"unexpected code output", "success", "", "skipped", "skipped", "skipped", false},
		{"unit failed", "success", "true", "failure", "success", "skipped", false},
		{"e2e cancelled", "success", "true", "success", "cancelled", "skipped", false},
		{"code job skipped on a code PR", "success", "true", "success", "skipped", "skipped", false},
		{"gitleaks ran on a code PR", "success", "true", "success", "success", "success", false},
		{"gitleaks failed on a chore PR", "success", "false", "skipped", "skipped", "failure", false},
		{"code job ran on a chore PR", "success", "false", "success", "skipped", "success", false},
	}
	for _, row := range rows {
		t.Run(row.name, func(t *testing.T) {
			cmd := exec.Command("bash", "-c", step.Run)
			cmd.Env = append(os.Environ(),
				"SCOPE="+row.scope, "CODE="+row.code, "UNIT="+row.unit,
				"E2E="+row.e2e, "GITLEAKS="+row.gitleaks)
			var combined strings.Builder
			cmd.Stdout = &combined
			cmd.Stderr = &combined
			err := cmd.Run()
			if row.pass && err != nil {
				t.Errorf("CI OK must pass, got %v\noutput:\n%s", err, combined.String())
			}
			if !row.pass && err == nil {
				t.Errorf("CI OK must fail, but it passed\noutput:\n%s", combined.String())
			}
		})
	}
}

// Chore PRs skip MegaLinter and with it gitleaks, so the gitleaks job
// scans their commits instead: it runs only when scope says the PR is a
// chore, verifies the pinned download checksum, scans base..head, and
// is aggregated by ci-ok so a found secret fails CI OK.
func TestCIWorkflow_GitleaksScansChorePRs(t *testing.T) {
	wf := loadCIWorkflow(t, "test.yml")
	job, ok := wf.Jobs["gitleaks"]
	if !ok {
		t.Fatal("test.yml must contain the gitleaks job")
	}
	if !ciNeedsJob(job, "scope") {
		t.Errorf("test.yml gitleaks needs = %v, must include scope", job.Needs)
	}
	const wantIf = "needs.scope.outputs.code == 'false'"
	if job.If != wantIf {
		t.Errorf("test.yml gitleaks if = %q, want %q — it must run on exactly the PRs that skip MegaLinter", job.If, wantIf)
	}
	if got := job.Permissions["contents"]; got != "read" || len(job.Permissions) != 1 {
		t.Errorf("test.yml gitleaks permissions = %v, want only contents: read", job.Permissions)
	}
	var install, scan string
	for _, step := range job.Steps {
		switch step.Name {
		case "Install gitleaks":
			install = step.Run
		case "Scan the pull request's commits":
			scan = step.Run
		}
	}
	if !strings.Contains(install, `sha256sum -c -`) {
		t.Errorf("test.yml gitleaks install must verify the download checksum (missing %q)", "sha256sum -c -")
	}
	if !strings.Contains(scan, `--log-opts="${BASE_SHA}..${HEAD_SHA}"`) {
		t.Errorf("test.yml gitleaks scan must cover the PR's commits (missing %q)", `--log-opts="${BASE_SHA}..${HEAD_SHA}"`)
	}
	if ciOK := wf.Jobs["ci-ok"]; !ciNeedsJob(ciOK, "gitleaks") {
		t.Errorf("test.yml ci-ok needs = %v, must include gitleaks so a found secret fails CI OK", ciOK.Needs)
	}
}

// --- ci-external-analyzer-support: reusable-workflow input contract ---

// reusableInput decodes one workflow_call input's contract fields.
type reusableInput struct {
	Type     string `yaml:"type"`
	Required bool   `yaml:"required"`
	Default  any    `yaml:"default"`
}

// reusableWorkflow decodes the workflow_call input surface of the
// reusable AI-report workflow. The gaze-version default's value is
// deliberately not pinned here: it is release-gate-managed by
// .github/scripts/validate-ai-report-default.sh, which forces it to
// equal the release tag shipping the file.
type reusableWorkflow struct {
	On struct {
		WorkflowCall struct {
			Inputs map[string]reusableInput `yaml:"inputs"`
		} `yaml:"workflow_call"`
	} `yaml:"on"`
	Jobs map[string]ciJob `yaml:"jobs"`
}

// loadReusableWorkflow reads and parses the reusable AI-report
// workflow under .github/workflows.
func loadReusableWorkflow(t *testing.T) reusableWorkflow {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", ".github", "workflows", "reusable_gaze_ai_report.yml"))
	if err != nil {
		t.Fatalf("read reusable_gaze_ai_report.yml: %v", err)
	}
	var wf reusableWorkflow
	if err := yaml.Unmarshal(raw, &wf); err != nil {
		t.Fatalf("parse reusable_gaze_ai_report.yml: %v", err)
	}
	return wf
}

// reusableStep returns the named step of the reusable workflow's
// gaze-ai-report job, failing when absent.
func reusableStep(t *testing.T, wf reusableWorkflow, name string) ciStep {
	t.Helper()
	job, ok := wf.Jobs["gaze-ai-report"]
	if !ok {
		t.Fatal("reusable workflow must contain the gaze-ai-report job")
	}
	for _, step := range job.Steps {
		if step.Name == name {
			return step
		}
	}
	t.Fatalf("reusable workflow gaze-ai-report job must contain a step named %q", name)
	return ciStep{}
}

// reusableStepIndex returns the position of the named step within the
// gaze-ai-report job's steps list, failing when absent. Resolving
// steps by name alone cannot detect a reorder, and this job's order
// is load-bearing (see the ordering assertions at the end of
// TestCIWorkflow_ReusableExternalAnalyzerInputs).
func reusableStepIndex(t *testing.T, wf reusableWorkflow, name string) int {
	t.Helper()
	job, ok := wf.Jobs["gaze-ai-report"]
	if !ok {
		t.Fatal("reusable workflow must contain the gaze-ai-report job")
	}
	for i, step := range job.Steps {
		if step.Name == name {
			return i
		}
	}
	t.Fatalf("reusable workflow gaze-ai-report job must contain a step named %q", name)
	return -1
}

// setupActionPinRe pins the toolchain setup actions to full commit
// SHAs, format-pinned rather than value-pinned so Dependabot's
// actions group can bump them (the shipScopePinRe precedent).
var setupActionPinRe = regexp.MustCompile(`^actions/setup-(node|python)@[0-9a-f]{40}$`)

// TestCIWorkflow_ReusableExternalAnalyzerInputs pins the
// external-analyzer input contract of the reusable AI-report workflow
// (ci-external-analyzer-support): the six optional inputs exist with
// Go-preserving defaults, the pre-existing Go inputs keep their
// contract, the toolchain setup steps are gated on their inputs, and
// the report and skipped-report steps branch on analyzer presence.
func TestCIWorkflow_ReusableExternalAnalyzerInputs(t *testing.T) {
	wf := loadReusableWorkflow(t)
	inputs := wf.On.WorkflowCall.Inputs

	// New optional inputs: Go callers that pin the workflow and pass
	// none of them must observe identical behavior.
	for name, wantDefault := range map[string]string{
		"analyzer":       "",
		"language":       "",
		"packages":       "./...",
		"setup":          "",
		"node-version":   "",
		"python-version": "",
	} {
		input, ok := inputs[name]
		if !ok {
			t.Errorf("reusable workflow must declare the %q input", name)
			continue
		}
		if input.Type != "string" {
			t.Errorf("%s input type = %q, want string", name, input.Type)
		}
		if input.Required {
			t.Errorf("%s input must be optional", name)
		}
		if got := fmt.Sprintf("%v", input.Default); got != wantDefault {
			t.Errorf("%s input default = %q, want %q", name, got, wantDefault)
		}
	}

	// Pre-existing Go inputs must keep their contract.
	goContract := []struct {
		name         string
		wantType     string
		wantRequired bool
		wantDefault  string
	}{
		{name: "coverage-artifact", wantType: "string", wantRequired: true},
		{name: "coverprofile", wantType: "string", wantDefault: "coverage.out"},
		{name: "opencode-version", wantType: "string", wantDefault: "1.19.0"},
		{name: "go-version", wantType: "string", wantDefault: "1.25"},
		{name: "build-from-source", wantType: "boolean", wantDefault: "false"},
		{name: "gaze-version", wantType: "string"},
		{name: "model", wantType: "string"},
		{name: "max-crapload", wantType: "number"},
		{name: "max-gaze-crapload", wantType: "number"},
		{name: "min-contract-coverage", wantType: "number"},
	}
	for _, want := range goContract {
		input, ok := inputs[want.name]
		if !ok {
			t.Errorf("reusable workflow must keep the %q input", want.name)
			continue
		}
		if input.Type != want.wantType {
			t.Errorf("%s input type = %q, want %s", want.name, input.Type, want.wantType)
		}
		if input.Required != want.wantRequired {
			t.Errorf("%s input required = %v, want %v", want.name, input.Required, want.wantRequired)
		}
		if want.wantDefault != "" {
			if got := fmt.Sprintf("%v", input.Default); got != want.wantDefault {
				t.Errorf("%s input default = %q, want %q", want.name, got, want.wantDefault)
			}
		}
	}

	// Toolchain setup steps are gated on their inputs so Go callers
	// never pay for them.
	node := reusableStep(t, wf, "Set up Node")
	if node.If != "inputs.node-version != ''" {
		t.Errorf("Set up Node if = %q, want inputs.node-version != ''", node.If)
	}
	if !setupActionPinRe.MatchString(node.Uses) {
		t.Errorf("Set up Node uses = %q, want a full-SHA actions/setup-node pin", node.Uses)
	}
	if got, _ := node.With["node-version"].(string); got != "${{ inputs.node-version }}" {
		t.Errorf("Set up Node with.node-version = %q, want ${{ inputs.node-version }}", got)
	}
	python := reusableStep(t, wf, "Set up Python")
	if python.If != "inputs.python-version != ''" {
		t.Errorf("Set up Python if = %q, want inputs.python-version != ''", python.If)
	}
	if !setupActionPinRe.MatchString(python.Uses) {
		t.Errorf("Set up Python uses = %q, want a full-SHA actions/setup-python pin", python.Uses)
	}
	if got, _ := python.With["python-version"].(string); got != "${{ inputs.python-version }}" {
		t.Errorf("Set up Python with.python-version = %q, want ${{ inputs.python-version }}", got)
	}

	// The download step's id backs the skipped-report arrival check,
	// which reads its conclusion.
	download := reusableStep(t, wf, "Download coverage")
	if download.ID != "download" {
		t.Errorf("Download coverage step id = %q, want \"download\" (the skipped-report arrival check reads its conclusion)", download.ID)
	}

	// The analyzer setup snippet runs only when the secret is present
	// and the caller supplied one, and arrives via env indirection —
	// never interpolated directly into the run script.
	setup := reusableStep(t, wf, "Setup analyzer")
	if setup.If != "env.HAS_OPENCODE_CONFIG == 'true' && inputs.setup != ''" {
		t.Errorf("Setup analyzer if = %q, want the secret-and-setup gate", setup.If)
	}
	if got := setup.Env["ANALYZER_SETUP"]; got != "${{ inputs.setup }}" {
		t.Errorf("Setup analyzer env ANALYZER_SETUP = %q, want ${{ inputs.setup }} (env indirection)", got)
	}
	if setup.Run != `eval "$ANALYZER_SETUP"` {
		t.Errorf("Setup analyzer run = %q, want eval \"$ANALYZER_SETUP\"", setup.Run)
	}
	if setup.Shell != "bash" {
		t.Errorf("Setup analyzer shell = %q, want bash", setup.Shell)
	}

	// Report step: the analyzer/language/coverprofile arguments
	// assemble conditionally, the positional packages argument replaces
	// the hard-coded ./... of the pre-analyzer era, and every assembled
	// variable must actually reach the gaze report invocation —
	// dropping one from the command line would silently degrade the
	// report (Go mode re-running the full test suite without
	// --coverprofile, analyzer mode falling back to Go analysis).
	report := reusableStep(t, wf, "Run gaze AI report")
	for _, want := range []string{
		// Assembly guards, not just the assembled values: dropping an
		// `if [ -n ... ]` guard would pass `--analyzer=` /
		// `--language=` on every Go-mode run, degrading to Go-native
		// analysis only because gaze happens to treat empty flags as
		// absent — an unpinned cross-component coupling.
		`if [ -n "$ANALYZER" ]; then`,
		`if [ -n "$LANGUAGE" ]; then`,
		`ANALYZER_ARG="--analyzer=$ANALYZER"`,
		`LANGUAGE_ARG="--language=$LANGUAGE"`,
		`COVERPROFILE_ARG="--coverprofile=$COVERPROFILE"`,
		`if [ -z "$ANALYZER" ] && [ -n "$COVERPROFILE" ]; then`,
		`gaze report $PACKAGES`,
		"--ai=opencode \\",
		"$MODEL_ARG \\",
		"$ANALYZER_ARG \\",
		"$LANGUAGE_ARG \\",
		"$COVERPROFILE_ARG \\",
		"$THRESHOLDS\n",
	} {
		if !strings.Contains(report.Run, want) {
			t.Errorf("Run gaze AI report must contain %q, got:\n%s", want, report.Run)
		}
	}

	// Skipped-report step: both modes validate that the coverage
	// artifact arrived via the download step's conclusion, and the
	// coverprofile existence check applies only in Go mode.
	skipped := reusableStep(t, wf, "Report skipped")
	if got := skipped.Env["DOWNLOAD_CONCLUSION"]; got != "${{ steps.download.conclusion }}" {
		t.Errorf("Report skipped env DOWNLOAD_CONCLUSION = %q, want ${{ steps.download.conclusion }}", got)
	}
	for _, want := range []string{
		`test "$DOWNLOAD_CONCLUSION" = "success"`,
		`if [ -z "$ANALYZER" ]; then`,
		`test -f "$COVERPROFILE"`,
		// Enforcement, not just detection: each check's ::error::
		// message is pinned together with the exit that makes failing
		// it fail the job — dropping an exit 1 would convert the gate
		// into an annotation-only silent pass (the ci-gate-integrity
		// failure class, #101/#108/#116).
		`coverage artifact '$ARTIFACT_NAME' did not arrive (download conclusion: ${DOWNLOAD_CONCLUSION:-none})`,
		`not found after downloading artifact '$ARTIFACT_NAME'`,
		`exit 1`,
	} {
		if !strings.Contains(skipped.Run, want) {
			t.Errorf("Report skipped must contain %q, got:\n%s", want, skipped.Run)
		}
	}

	// Step order is load-bearing in this job and per-step content
	// assertions cannot detect a reorder: the toolchain steps must
	// precede Setup analyzer (the caller's setup snippet may invoke
	// npm/pip), the analyzer must be installed before the report step
	// spawns it, the coverage artifact must be downloaded before
	// either report step reads it (Report skipped reads
	// steps.download.conclusion), and the report must precede its
	// skipped fallback. The gate-order precedent is
	// release_workflow_test.go.
	nodeIdx := reusableStepIndex(t, wf, "Set up Node")
	pythonIdx := reusableStepIndex(t, wf, "Set up Python")
	setupIdx := reusableStepIndex(t, wf, "Setup analyzer")
	downloadIdx := reusableStepIndex(t, wf, "Download coverage")
	reportIdx := reusableStepIndex(t, wf, "Run gaze AI report")
	skippedIdx := reusableStepIndex(t, wf, "Report skipped")
	if nodeIdx >= setupIdx || pythonIdx >= setupIdx {
		t.Errorf("toolchain steps (Node %d, Python %d) must precede Setup analyzer (%d)", nodeIdx, pythonIdx, setupIdx)
	}
	if setupIdx >= reportIdx {
		t.Errorf("Setup analyzer (%d) must precede Run gaze AI report (%d)", setupIdx, reportIdx)
	}
	if downloadIdx >= reportIdx || downloadIdx >= skippedIdx {
		t.Errorf("Download coverage (%d) must precede the report steps (report %d, skipped %d)", downloadIdx, reportIdx, skippedIdx)
	}
	if reportIdx >= skippedIdx {
		t.Errorf("Run gaze AI report (%d) must precede Report skipped (%d)", reportIdx, skippedIdx)
	}
}
