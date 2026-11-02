package aireport

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"
)

// Shared golden-test helpers for spec 040 (T005).
//
// The golden files under goldenDir are the normative byte-level
// contract for the Gaze-rendered markdown report
// (contracts/markdown-report.md). They are read by the renderer tests
// in this package AND by the CLI tests in cmd/gaze (SC-003). Because
// cmd/gaze cannot import _test.go files, cmd/gaze duplicates the path
// as aireportGoldenDir (cmd/gaze/golden_paths_test.go), and a drift
// guard there asserts the goldenDir declaration below is unchanged.
// Design choice: duplicating one constant was preferred over a new
// exported test-support package, which would import aireport and so
// could not be imported by this package's own in-package tests
// (import cycle).

// goldenDir is the golden directory, relative to this package. The
// cmd/gaze drift guard matches this exact declaration line; update
// cmd/gaze/golden_paths_test.go if it ever changes.
const goldenDir = "testdata/golden"

// goldenUpdateHint tells a developer how to regenerate golden files.
const goldenUpdateHint = "run `go test ./internal/aireport -run Golden -update` " +
	"to regenerate, then review the golden diff by hand"

// goldenDiffContext is the number of unchanged lines printed before
// the first difference, and goldenDiffWindow caps the number of
// removed/added lines printed from each side.
const (
	goldenDiffContext = 3
	goldenDiffWindow  = 8
)

// update rewrites golden files instead of comparing against them. It
// is test-only; CI never passes it (contracts/markdown-report.md).
var update = flag.Bool("update", false,
	"rewrite golden files under "+goldenDir+" instead of comparing against them")

// aiSlotRE matches one complete AI slot block plus its trailing blank
// line (research R6). The research pattern
//
//	(?s)<!-- gaze:ai-(summary|recommendations) -->\n.*?<!-- /gaze:ai-\1 -->\n\n
//
// uses a backreference, which Go's RE2 engine does not support. It is
// expanded here into explicit alternation with one fully spelled
// branch per slot name, so each opening marker only pairs with its
// own closing marker — the same language the backreference describes.
//
// A recommendations slot that ends the report carries the report's
// single trailing newline instead of a blank line (markdown-report rule
// 11), so the trailing separator also accepts end of input.
var aiSlotRE = regexp.MustCompile(
	`(?s)(?:<!-- gaze:ai-summary -->\n.*?<!-- /gaze:ai-summary -->` +
		`|<!-- gaze:ai-recommendations -->\n.*?<!-- /gaze:ai-recommendations -->)` +
		`(?:\n\n|\n\z)`)

// stripAISlots removes every AI slot block (markers, content, and the
// single trailing blank line) from a rendered report, leaving only
// the deterministic Gaze-rendered bytes (SC-002 byte identity).
func stripAISlots(s string) string {
	return aiSlotRE.ReplaceAllString(s, "")
}

// goldenPath returns the path of the named file in goldenDir.
func goldenPath(name string) string {
	return filepath.Join(goldenDir, name)
}

// fixtureSummary mirrors ReportSummary with snake_case JSON tags.
// ReportSummary is json:"-" in production output, so payload fixtures
// carry it under a separate top-level "summary" key that only this
// test decoder reads. Keep the fields in sync with ReportSummary.
type fixtureSummary struct {
	TotalFunctions      int      `json:"total_functions"`
	CRAPload            *int     `json:"crapload"`
	GazeCRAPload        *int     `json:"gaze_crapload"`
	AvgContractCoverage *int     `json:"avg_contract_coverage"`
	SSADegraded         bool     `json:"ssa_degraded"`
	SSADegradedPackages []string `json:"ssa_degraded_packages"`
	Contractual         int      `json:"contractual"`
	Ambiguous           int      `json:"ambiguous"`
	Incidental          int      `json:"incidental"`
	SkippedTests        int      `json:"skipped_tests"`
	SkippedTestNames    []string `json:"skipped_test_names"`
}

// reportSummary converts the fixture mirror into the production type.
func (f fixtureSummary) reportSummary() ReportSummary {
	// The mirror has the same fields in the same order as ReportSummary
	// (only the struct tags differ), so a direct conversion is valid and
	// fails to compile if the two types ever drift apart.
	return ReportSummary(f)
}

// decodePayloadFixture decodes fixture bytes: the top-level keys are
// the ReportPayload JSON keys, plus an optional "summary" object with
// fixtureSummary keys. Unknown keys inside "summary" are an error so a
// typo cannot silently zero a metric. Unknown top-level keys are
// tolerated so fixtures may carry keys added by later tasks (for
// example metadata, T008) before the struct fields exist.
func decodePayloadFixture(data []byte) (*ReportPayload, error) {
	var p ReportPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("decoding payload: %w", err)
	}
	var wrapper struct {
		Summary json.RawMessage `json:"summary"`
	}
	if err := json.Unmarshal(data, &wrapper); err != nil {
		return nil, fmt.Errorf("decoding summary wrapper: %w", err)
	}
	if len(wrapper.Summary) == 0 {
		return &p, nil
	}
	var fixture fixtureSummary
	dec := json.NewDecoder(bytes.NewReader(wrapper.Summary))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&fixture); err != nil {
		return nil, fmt.Errorf("decoding summary: %w", err)
	}
	p.Summary = fixture.reportSummary()
	return &p, nil
}

// loadPayloadFixture reads and decodes the named payload fixture from
// goldenDir (for example "payload-full.json"). It fails the test on
// any read or decode error.
func loadPayloadFixture(t *testing.T, name string) *ReportPayload {
	t.Helper()
	data, err := os.ReadFile(goldenPath(name))
	if err != nil {
		t.Fatalf("reading payload fixture %s: %v", name, err)
	}
	p, err := decodePayloadFixture(data)
	if err != nil {
		t.Fatalf("payload fixture %s: %v", name, err)
	}
	return p
}

// assertGolden compares got with the named golden file in goldenDir,
// or rewrites the file when -update is set.
func assertGolden(t *testing.T, name string, got string) {
	t.Helper()
	assertGoldenFile(t, goldenPath(name), got, *update)
}

// assertGoldenFile is the path-explicit core of assertGolden. With
// update set it writes got to path; otherwise it compares bytes and
// reports a unified-diff hint on mismatch.
func assertGoldenFile(t *testing.T, path, got string, update bool) {
	t.Helper()
	if update {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("creating golden dir for %s: %v", path, err)
		}
		if err := os.WriteFile(path, []byte(got), 0o644); err != nil {
			t.Fatalf("writing golden %s: %v", path, err)
		}
		t.Logf("updated golden %s", path)
		return
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading golden %s: %v; %s", path, err, goldenUpdateHint)
	}
	if msg := goldenDiff(string(want), got); msg != "" {
		t.Errorf("%s: %s", path, msg)
	}
}

// goldenDiff returns "" when want equals got. Otherwise it returns a
// unified-diff style hint: byte counts, the first differing line, up
// to goldenDiffContext unchanged lines before it, a bounded window of
// removed (want) and added (got) lines, and the regeneration hint.
func goldenDiff(want, got string) string {
	if want == got {
		return ""
	}
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")
	first := firstDiffLine(wantLines, gotLines)
	start := max(0, first-goldenDiffContext)

	var b strings.Builder
	fmt.Fprintf(&b, "golden mismatch (want %d bytes, got %d bytes), first difference at line %d\n",
		len(want), len(got), first+1)
	fmt.Fprintf(&b, "--- want\n+++ got\n@@ -%d +%d @@\n", start+1, start+1)
	for _, line := range wantLines[start:first] {
		b.WriteString(" " + line + "\n")
	}
	writeDiffLines(&b, "-", wantLines, first)
	writeDiffLines(&b, "+", gotLines, first)
	b.WriteString(goldenUpdateHint)
	return b.String()
}

// firstDiffLine returns the index of the first line that differs
// between a and b, or the shorter length when one is a prefix.
func firstDiffLine(a, b []string) int {
	n := min(len(a), len(b))
	for i := 0; i < n; i++ {
		if a[i] != b[i] {
			return i
		}
	}
	return n
}

// writeDiffLines writes up to goldenDiffWindow lines from lines[from:]
// with the given prefix ("-" or "+").
func writeDiffLines(b *strings.Builder, prefix string, lines []string, from int) {
	end := min(len(lines), from+goldenDiffWindow)
	for _, line := range lines[from:end] {
		b.WriteString(prefix + line + "\n")
	}
}

// --- Self-tests for the helpers -------------------------------------

func TestStripAISlots(t *testing.T) {
	const summary = "<!-- gaze:ai-summary -->\n### Summary\n\nAI text.\n<!-- /gaze:ai-summary -->\n\n"
	const recs = "<!-- gaze:ai-recommendations -->\n### AI-Prioritized Recommendations\n\n1. Do it.\n<!-- /gaze:ai-recommendations -->\n\n"
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"no slots", "# Title\n\nbody\n", "# Title\n\nbody\n"},
		{"summary slot", "# T\n\n" + summary + "## A\n", "# T\n\n## A\n"},
		{"recommendations slot", "## H\n\n" + recs + "end\n", "## H\n\nend\n"},
		{"both slots", "# T\n\n" + summary + "mid\n\n" + recs + "end\n", "# T\n\nmid\n\nend\n"},
		{
			"mismatched close marker kept",
			"<!-- gaze:ai-summary -->\nx\n<!-- /gaze:ai-recommendations -->\n\nend\n",
			"<!-- gaze:ai-summary -->\nx\n<!-- /gaze:ai-recommendations -->\n\nend\n",
		},
		{
			"slot at end of report stripped",
			"# T\n\n" + strings.TrimSuffix(recs, "\n"),
			"# T\n\n",
		},
		{
			"missing trailing blank line kept",
			"<!-- gaze:ai-summary -->\nx\n<!-- /gaze:ai-summary -->\nend\n",
			"<!-- gaze:ai-summary -->\nx\n<!-- /gaze:ai-summary -->\nend\n",
		},
		{
			"non-greedy stops at first close",
			summary + "keep\n\n" + summary + "end\n",
			"keep\n\nend\n",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := stripAISlots(tc.in); got != tc.want {
				t.Errorf("stripAISlots() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestGoldenDiff(t *testing.T) {
	if got := goldenDiff("same\n", "same\n"); got != "" {
		t.Errorf("goldenDiff(equal) = %q, want empty", got)
	}

	got := goldenDiff("a\nb\nc\nd\ne\n", "a\nb\nc\nd\nX\n")
	for _, want := range []string{
		"want 10 bytes, got 10 bytes",
		"first difference at line 5",
		"--- want\n+++ got\n@@ -2 +2 @@\n",
		" b\n c\n d\n-e\n",
		"+X\n",
		goldenUpdateHint,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("goldenDiff() missing %q in:\n%s", want, got)
		}
	}

	// A trailing-newline-only difference must still be reported.
	got = goldenDiff("a\n", "a\n\n")
	if !strings.Contains(got, "want 2 bytes, got 3 bytes") ||
		!strings.Contains(got, "first difference at line 3") {
		t.Errorf("goldenDiff(trailing newline) = %q", got)
	}

	// The removed/added window is bounded.
	long := strings.Repeat("x\n", 50)
	got = goldenDiff("", long)
	if n := strings.Count(got, "+x\n"); n != goldenDiffWindow {
		t.Errorf("goldenDiff() printed %d added lines, want %d", n, goldenDiffWindow)
	}
}

func TestAssertGoldenFile_UpdateThenCompare(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "report.md")
	const content = "# 🔍 Gaze CRAP Report\n\n| a \\| b |\n"

	assertGoldenFile(t, path, content, true)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading written golden: %v", err)
	}
	if string(data) != content {
		t.Errorf("written golden = %q, want %q", data, content)
	}
	// Compare mode passes against the bytes just written.
	assertGoldenFile(t, path, content, false)
}

func TestDecodePayloadFixture(t *testing.T) {
	data := []byte(`{
		"crap": {"summary": {"total_functions": 3}},
		"quality": null,
		"classify": null,
		"docscan": null,
		"errors": {"crap": null, "quality": "boom", "classify": null, "docscan": null},
		"summary": {
			"total_functions": 3,
			"crapload": 2,
			"ssa_degraded": true,
			"ssa_degraded_packages": ["example.com/p"],
			"ambiguous": 7,
			"skipped_tests": 1,
			"skipped_test_names": ["TestX"]
		}
	}`)
	p, err := decodePayloadFixture(data)
	if err != nil {
		t.Fatalf("decodePayloadFixture() error = %v", err)
	}
	if string(p.CRAP) != `{"summary": {"total_functions": 3}}` {
		t.Errorf("CRAP = %s", p.CRAP)
	}
	if p.Quality != nil && string(p.Quality) != "null" {
		t.Errorf("Quality = %s, want null", p.Quality)
	}
	if p.Errors.Quality == nil || *p.Errors.Quality != "boom" {
		t.Errorf("Errors.Quality = %v, want boom", p.Errors.Quality)
	}
	s := p.Summary
	if s.TotalFunctions != 3 || s.CRAPload == nil || *s.CRAPload != 2 {
		t.Errorf("Summary totals = %d/%v, want 3/2", s.TotalFunctions, s.CRAPload)
	}
	if s.GazeCRAPload != nil {
		t.Errorf("Summary.GazeCRAPload = %d, want nil (absent)", *s.GazeCRAPload)
	}
	if s.AvgContractCoverage != nil {
		t.Errorf("Summary.AvgContractCoverage = %d, want nil", *s.AvgContractCoverage)
	}
	if !s.SSADegraded || len(s.SSADegradedPackages) != 1 || s.Ambiguous != 7 ||
		s.SkippedTests != 1 || len(s.SkippedTestNames) != 1 {
		t.Errorf("Summary = %+v", s)
	}
}

func TestDecodePayloadFixture_NoSummary(t *testing.T) {
	p, err := decodePayloadFixture([]byte(`{"crap": null, "errors": {}}`))
	if err != nil {
		t.Fatalf("decodePayloadFixture() error = %v", err)
	}
	if p.Summary.TotalFunctions != 0 || p.Summary.CRAPload != nil {
		t.Errorf("Summary = %+v, want zero value", p.Summary)
	}
}

func TestDecodePayloadFixture_Errors(t *testing.T) {
	tests := []struct {
		name    string
		data    string
		wantErr string
	}{
		{"invalid json", `{`, "decoding payload"},
		{"unknown summary key", `{"summary": {"gaze_crap_load": 1}}`, "decoding summary"},
		{"wrong summary type", `{"summary": {"total_functions": "3"}}`, "decoding summary"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decodePayloadFixture([]byte(tc.data))
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("decodePayloadFixture() error = %v, want containing %q", err, tc.wantErr)
			}
		})
	}
}

// TestGoldenHelpers_SharedFiles exercises loadPayloadFixture and
// assertGolden against whatever shared golden files exist. Payload
// fixtures must decode, and every golden markdown file must match
// itself through the goldenDir path resolution the renderer tests
// use. It skips until the fixtures exist (T002/T003/T026) and under
// -update so it never rewrites files.
func TestGoldenHelpers_SharedFiles(t *testing.T) {
	if *update {
		t.Skip("-update set; golden self-check skipped")
	}
	entries, err := os.ReadDir(goldenDir)
	if errors.Is(err, fs.ErrNotExist) {
		t.Skipf("%s does not exist yet", goldenDir)
	}
	if err != nil {
		t.Fatalf("reading %s: %v", goldenDir, err)
	}
	for _, e := range entries {
		name := e.Name()
		switch filepath.Ext(name) {
		case ".json":
			if p := loadPayloadFixture(t, name); p == nil {
				t.Errorf("loadPayloadFixture(%s) = nil", name)
			}
		case ".md":
			// Every golden report on disk must be pinned by a case in
			// TestRenderMarkdown_Golden; an unpinned file would never be
			// regenerated or reviewed.
			if strings.HasPrefix(name, "report-") && !slices.Contains(markdownGoldenNames, name) {
				t.Errorf("%s has no TestRenderMarkdown_Golden case", name)
			}
		}
	}
}
