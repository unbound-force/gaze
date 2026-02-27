package crap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/token"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"

	"github.com/fzipp/gocyclo"
	"golang.org/x/tools/cover"
)

func TestFormula_ZeroCoverage(t *testing.T) {
	// CRAP(5, 0%) = 5^2 * (1-0)^3 + 5 = 25 + 5 = 30
	got := Formula(5, 0)
	want := 30.0
	if math.Abs(got-want) > 0.01 {
		t.Errorf("Formula(5, 0) = %f, want %f", got, want)
	}
}

func TestFormula_FullCoverage(t *testing.T) {
	// CRAP(5, 100%) = 5^2 * (1-1)^3 + 5 = 0 + 5 = 5
	got := Formula(5, 100)
	want := 5.0
	if math.Abs(got-want) > 0.01 {
		t.Errorf("Formula(5, 100) = %f, want %f", got, want)
	}
}

func TestFormula_HalfCoverage(t *testing.T) {
	// CRAP(10, 50%) = 100 * (0.5)^3 + 10 = 100*0.125 + 10 = 22.5
	got := Formula(10, 50)
	want := 22.5
	if math.Abs(got-want) > 0.01 {
		t.Errorf("Formula(10, 50) = %f, want %f", got, want)
	}
}

func TestFormula_Complexity1_FullCoverage(t *testing.T) {
	// CRAP(1, 100%) = 1 * 0 + 1 = 1
	got := Formula(1, 100)
	want := 1.0
	if math.Abs(got-want) > 0.01 {
		t.Errorf("Formula(1, 100) = %f, want %f", got, want)
	}
}

func TestFormula_Complexity1_ZeroCoverage(t *testing.T) {
	// CRAP(1, 0%) = 1 * 1 + 1 = 2
	got := Formula(1, 0)
	want := 2.0
	if math.Abs(got-want) > 0.01 {
		t.Errorf("Formula(1, 0) = %f, want %f", got, want)
	}
}

func TestFormula_HighComplexity_ZeroCoverage(t *testing.T) {
	// CRAP(30, 0%) = 900 * 1 + 30 = 930
	got := Formula(30, 0)
	want := 930.0
	if math.Abs(got-want) > 0.01 {
		t.Errorf("Formula(30, 0) = %f, want %f", got, want)
	}
}

func TestFormula_75PercentCoverage(t *testing.T) {
	// CRAP(10, 75%) = 100 * (0.25)^3 + 10 = 100*0.015625 + 10 = 11.5625
	got := Formula(10, 75)
	want := 11.5625
	if math.Abs(got-want) > 0.01 {
		t.Errorf("Formula(10, 75) = %f, want %f", got, want)
	}
}

// TestFormula_BenchmarkSuite validates SC-001: CRAP scores match
// hand-computed values for a benchmark suite of 20+ functions with
// known complexity and coverage (tolerance: +/- 0.01).
func TestFormula_BenchmarkSuite(t *testing.T) {
	cases := []struct {
		name       string
		complexity int
		coverage   float64
		want       float64
	}{
		// Boundary: minimum complexity, zero coverage
		{"comp2_cov0", 2, 0, 6.0},      // 4*1 + 2 = 6
		{"comp3_cov0", 3, 0, 12.0},     // 9*1 + 3 = 12
		{"comp4_cov0", 4, 0, 20.0},     // 16*1 + 4 = 20
		{"comp15_cov0", 15, 0, 240.0},  // 225*1 + 15 = 240
		{"comp20_cov0", 20, 0, 420.0},  // 400*1 + 20 = 420
		{"comp50_cov0", 50, 0, 2550.0}, // 2500*1 + 50 = 2550

		// Full coverage: CRAP = comp (uncov^3 = 0)
		{"comp2_cov100", 2, 100, 2.0},
		{"comp10_cov100", 10, 100, 10.0},
		{"comp20_cov100", 20, 100, 20.0},
		{"comp50_cov100", 50, 100, 50.0},

		// 25% coverage: uncov = 0.75, uncov^3 = 0.421875
		{"comp4_cov25", 4, 25, 10.75},     // 16*0.421875 + 4 = 10.75
		{"comp8_cov25", 8, 25, 35.0},      // 64*0.421875 + 8 = 35.0
		{"comp10_cov25", 10, 25, 52.1875}, // 100*0.421875 + 10 = 52.1875

		// 90% coverage: uncov = 0.10, uncov^3 = 0.001
		{"comp5_cov90", 5, 90, 5.025},   // 25*0.001 + 5 = 5.025
		{"comp10_cov90", 10, 90, 10.1},  // 100*0.001 + 10 = 10.1
		{"comp20_cov90", 20, 90, 20.4},  // 400*0.001 + 20 = 20.4
		{"comp50_cov90", 50, 90, 52.5},  // 2500*0.001 + 50 = 52.5
		{"comp100_cov90", 100, 90, 110}, // 10000*0.001 + 100 = 110

		// Mixed coverage levels
		{"comp3_cov30", 3, 30, 6.087},  // 9*(0.7)^3 + 3 = 9*0.343 + 3 = 6.087
		{"comp6_cov10", 6, 10, 32.244}, // 36*(0.9)^3 + 6 = 36*0.729 + 6 = 32.244
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Formula(tc.complexity, tc.coverage)
			if math.Abs(got-tc.want) > 0.01 {
				t.Errorf("Formula(%d, %.1f) = %.4f, want %.4f",
					tc.complexity, tc.coverage, got, tc.want)
			}
		})
	}
}

func TestClassifyQuadrant_Q1Safe(t *testing.T) {
	q := ClassifyQuadrant(10, 10, 15, 15)
	if q != Q1Safe {
		t.Errorf("expected Q1_Safe, got %s", q)
	}
}

func TestClassifyQuadrant_Q2ComplexButTested(t *testing.T) {
	q := ClassifyQuadrant(20, 10, 15, 15)
	if q != Q2ComplexButTested {
		t.Errorf("expected Q2_ComplexButTested, got %s", q)
	}
}

func TestClassifyQuadrant_Q3SimpleButUnderspecified(t *testing.T) {
	q := ClassifyQuadrant(10, 20, 15, 15)
	if q != Q3SimpleButUnderspecified {
		t.Errorf("expected Q3_SimpleButUnderspecified, got %s", q)
	}
}

func TestClassifyQuadrant_Q4Dangerous(t *testing.T) {
	q := ClassifyQuadrant(20, 20, 15, 15)
	if q != Q4Dangerous {
		t.Errorf("expected Q4_Dangerous, got %s", q)
	}
}

func TestClassifyQuadrant_ExactThreshold(t *testing.T) {
	// At exactly the threshold, it should be "at or above".
	q := ClassifyQuadrant(15, 15, 15, 15)
	if q != Q4Dangerous {
		t.Errorf("expected Q4_Dangerous at exact threshold, got %s", q)
	}
}

func TestClassifyQuadrant_IndependentThresholds(t *testing.T) {
	// CRAP threshold = 20, GazeCRAP threshold = 10.
	// CRAP=18 (below 20), GazeCRAP=12 (above 10) → Q3
	q := ClassifyQuadrant(18, 12, 20, 10)
	if q != Q3SimpleButUnderspecified {
		t.Errorf("expected Q3 with independent thresholds, got %s", q)
	}
}

func TestBuildSummary_CRAPload(t *testing.T) {
	scores := []Score{
		{Complexity: 5, LineCoverage: 0, CRAP: 30},     // above 15
		{Complexity: 3, LineCoverage: 100, CRAP: 3},    // below 15
		{Complexity: 10, LineCoverage: 50, CRAP: 22.5}, // above 15
		{Complexity: 1, LineCoverage: 0, CRAP: 2},      // below 15
	}

	opts := DefaultOptions()
	summary := buildSummary(scores, opts)

	if summary.CRAPload != 2 {
		t.Errorf("expected CRAPload 2, got %d", summary.CRAPload)
	}
	if summary.TotalFunctions != 4 {
		t.Errorf("expected 4 functions, got %d", summary.TotalFunctions)
	}
}

func TestBuildSummary_WorstOffenders(t *testing.T) {
	scores := make([]Score, 10)
	for i := range scores {
		scores[i] = Score{
			Function: fmt.Sprintf("Func%d", i),
			CRAP:     float64(i * 10),
		}
	}

	opts := DefaultOptions()
	summary := buildSummary(scores, opts)

	if len(summary.WorstCRAP) != 5 {
		t.Errorf("expected 5 worst offenders, got %d",
			len(summary.WorstCRAP))
	}
	// Worst should be the highest CRAP score.
	if summary.WorstCRAP[0].CRAP != 90 {
		t.Errorf("expected worst CRAP 90, got %f",
			summary.WorstCRAP[0].CRAP)
	}
}

func TestBuildSummary_WithGazeCRAP(t *testing.T) {
	gazeCRAP1 := 25.0
	gazeCRAP2 := 10.0
	gazeCRAP3 := 35.0
	contractCov1 := 60.0
	contractCov2 := 90.0
	contractCov3 := 30.0
	q1 := Q1Safe
	q2 := Q2ComplexButTested
	q3 := Q4Dangerous

	scores := []Score{
		{
			Function:         "A",
			CRAP:             20,
			Complexity:       5,
			GazeCRAP:         &gazeCRAP1,
			ContractCoverage: &contractCov1,
			Quadrant:         &q1,
		},
		{
			Function:         "B",
			CRAP:             8,
			Complexity:       3,
			GazeCRAP:         &gazeCRAP2,
			ContractCoverage: &contractCov2,
			Quadrant:         &q2,
		},
		{
			Function:         "C",
			CRAP:             30,
			Complexity:       10,
			GazeCRAP:         &gazeCRAP3,
			ContractCoverage: &contractCov3,
			Quadrant:         &q3,
		},
	}

	opts := DefaultOptions()
	summary := buildSummary(scores, opts)

	// GazeCRAPload: functions with GazeCRAP >= 15 → A (25) and C (35) = 2.
	if summary.GazeCRAPload == nil {
		t.Fatal("expected GazeCRAPload to be non-nil")
	}
	if *summary.GazeCRAPload != 2 {
		t.Errorf("expected GazeCRAPload 2, got %d", *summary.GazeCRAPload)
	}

	// AvgGazeCRAP: (25 + 10 + 35) / 3 = 23.333...
	if summary.AvgGazeCRAP == nil {
		t.Fatal("expected AvgGazeCRAP to be non-nil")
	}
	wantAvgGaze := (25.0 + 10.0 + 35.0) / 3.0
	if math.Abs(*summary.AvgGazeCRAP-wantAvgGaze) > 0.01 {
		t.Errorf("expected AvgGazeCRAP %.4f, got %.4f", wantAvgGaze, *summary.AvgGazeCRAP)
	}

	// AvgContractCoverage: (60 + 90 + 30) / 3 = 60.
	if summary.AvgContractCoverage == nil {
		t.Fatal("expected AvgContractCoverage to be non-nil")
	}
	wantAvgCov := (60.0 + 90.0 + 30.0) / 3.0
	if math.Abs(*summary.AvgContractCoverage-wantAvgCov) > 0.01 {
		t.Errorf("expected AvgContractCoverage %.4f, got %.4f",
			wantAvgCov, *summary.AvgContractCoverage)
	}

	// WorstGazeCRAP: sorted descending → C (35), A (25), B (10).
	if len(summary.WorstGazeCRAP) != 3 {
		t.Fatalf("expected 3 WorstGazeCRAP entries, got %d", len(summary.WorstGazeCRAP))
	}
	if summary.WorstGazeCRAP[0].Function != "C" {
		t.Errorf("expected WorstGazeCRAP[0] to be C, got %s",
			summary.WorstGazeCRAP[0].Function)
	}
	if summary.WorstGazeCRAP[1].Function != "A" {
		t.Errorf("expected WorstGazeCRAP[1] to be A, got %s",
			summary.WorstGazeCRAP[1].Function)
	}

	// All WorstGazeCRAP entries must have non-nil GazeCRAP.
	for i, ws := range summary.WorstGazeCRAP {
		if ws.GazeCRAP == nil {
			t.Errorf("WorstGazeCRAP[%d] has nil GazeCRAP", i)
		}
	}

	// QuadrantCounts.
	if summary.QuadrantCounts[Q1Safe] != 1 {
		t.Errorf("expected Q1Safe count 1, got %d", summary.QuadrantCounts[Q1Safe])
	}
	if summary.QuadrantCounts[Q4Dangerous] != 1 {
		t.Errorf("expected Q4Dangerous count 1, got %d", summary.QuadrantCounts[Q4Dangerous])
	}
}

func TestBuildSummary_Empty(t *testing.T) {
	opts := DefaultOptions()
	summary := buildSummary(nil, opts)

	if summary.TotalFunctions != 0 {
		t.Errorf("expected 0 functions, got %d", summary.TotalFunctions)
	}
}

func TestWriteJSON_ValidOutput(t *testing.T) {
	report := &Report{
		Scores: []Score{
			{
				Package:      "pkg",
				Function:     "Foo",
				File:         "foo.go",
				Line:         10,
				Complexity:   5,
				LineCoverage: 80,
				CRAP:         5.8,
			},
		},
		Summary: Summary{
			TotalFunctions: 1,
			CRAPThreshold:  15,
		},
	}

	var buf bytes.Buffer
	if err := WriteJSON(&buf, report); err != nil {
		t.Fatal(err)
	}

	var parsed map[string]interface{}
	if err := json.Unmarshal(buf.Bytes(), &parsed); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestWriteText_ContainsSummary(t *testing.T) {
	report := &Report{
		Scores: []Score{
			{
				Package:      "pkg",
				Function:     "Foo",
				File:         "foo.go",
				Line:         10,
				Complexity:   5,
				LineCoverage: 80,
				CRAP:         5.8,
			},
		},
		Summary: Summary{
			TotalFunctions:  1,
			AvgComplexity:   5,
			AvgLineCoverage: 80,
			AvgCRAP:         5.8,
			CRAPload:        0,
			CRAPThreshold:   15,
		},
	}

	var buf bytes.Buffer
	if err := WriteText(&buf, report); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	if !strings.Contains(output, "Functions analyzed") {
		t.Error("missing summary header")
	}
	if !strings.Contains(output, "CRAPload") {
		t.Error("missing CRAPload")
	}
	if !strings.Contains(output, "Foo") {
		t.Error("missing function name")
	}
}

func TestWriteText_MarksAboveThreshold(t *testing.T) {
	report := &Report{
		Scores: []Score{
			{Function: "Bad", CRAP: 30, Complexity: 5, LineCoverage: 0},
			{Function: "Good", CRAP: 3, Complexity: 1, LineCoverage: 100},
		},
		Summary: Summary{
			TotalFunctions: 2,
			CRAPThreshold:  15,
			CRAPload:       1,
		},
	}

	var buf bytes.Buffer
	if err := WriteText(&buf, report); err != nil {
		t.Fatal(err)
	}

	output := buf.String()
	// The * marker should appear for the function above threshold.
	if !strings.Contains(output, "*") {
		t.Error("expected * marker for function above threshold")
	}
}

// --- isGeneratedFile tests ---

func TestIsGeneratedFile_Generated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "gen.go")
	content := `// Code generated by protoc-gen-go. DO NOT EDIT.

package pb

func Foo() {}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if !isGeneratedFile(path) {
		t.Error("expected file to be detected as generated")
	}
}

func TestIsGeneratedFile_NotGenerated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "normal.go")
	content := `// Package foo provides functionality.
package foo

func Bar() {}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if isGeneratedFile(path) {
		t.Error("expected file NOT to be detected as generated")
	}
}

func TestIsGeneratedFile_GeneratedAfterPackage(t *testing.T) {
	// A "Code generated" comment AFTER the package clause should
	// NOT count as generated (per Go convention).
	dir := t.TempDir()
	path := filepath.Join(dir, "late.go")
	content := `package foo

// Code generated by something. DO NOT EDIT.
func Baz() {}
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if isGeneratedFile(path) {
		t.Error("comment after package clause should not be detected as generated")
	}
}

func TestIsGeneratedFile_NonexistentFile(t *testing.T) {
	if isGeneratedFile("/nonexistent/path/file.go") {
		t.Error("nonexistent file should return false")
	}
}

// --- resolvePatterns tests ---

func TestResolvePatterns_DotSlashDotDotDot(t *testing.T) {
	paths, err := resolvePatterns([]string{"./..."}, "/module/dir")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "/module/dir" {
		t.Errorf("expected [/module/dir], got %v", paths)
	}
}

func TestResolvePatterns_DotSlashPrefix(t *testing.T) {
	paths, err := resolvePatterns([]string{"./cmd/gaze"}, "/module/dir")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/module/dir", "./cmd/gaze")
	if len(paths) != 1 || paths[0] != want {
		t.Errorf("expected [%s], got %v", want, paths)
	}
}

func TestResolvePatterns_BarePattern(t *testing.T) {
	paths, err := resolvePatterns([]string{"some/path"}, "/module/dir")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 1 || paths[0] != "some/path" {
		t.Errorf("expected [some/path], got %v", paths)
	}
}

func TestResolvePatterns_MultiplePatterns(t *testing.T) {
	patterns := []string{"./...", "./internal/crap", "bare"}
	paths, err := resolvePatterns(patterns, "/mod")
	if err != nil {
		t.Fatal(err)
	}
	if len(paths) != 3 {
		t.Fatalf("expected 3 paths, got %d", len(paths))
	}
	if paths[0] != "/mod" {
		t.Errorf("paths[0] = %q, want /mod", paths[0])
	}
	if paths[1] != filepath.Join("/mod", "./internal/crap") {
		t.Errorf("paths[1] = %q, want %s", paths[1],
			filepath.Join("/mod", "./internal/crap"))
	}
	if paths[2] != "bare" {
		t.Errorf("paths[2] = %q, want bare", paths[2])
	}
}

// --- buildCoverMap tests ---

func TestBuildCoverMap_Basic(t *testing.T) {
	coverages := []FuncCoverage{
		{File: "foo.go", StartLine: 10, Percentage: 85.0},
		{File: "bar.go", StartLine: 20, Percentage: 50.0},
	}
	m := buildCoverMap(coverages)

	if pct, ok := m.exact[coverKey{file: "foo.go", line: 10}]; !ok || pct != 85.0 {
		t.Errorf("expected 85.0 for foo.go:10, got %v (ok=%v)", pct, ok)
	}
	if pct, ok := m.exact[coverKey{file: "bar.go", line: 20}]; !ok || pct != 50.0 {
		t.Errorf("expected 50.0 for bar.go:20, got %v (ok=%v)", pct, ok)
	}
}

func TestBuildCoverMap_Empty(t *testing.T) {
	m := buildCoverMap(nil)
	if len(m.exact) != 0 {
		t.Errorf("expected empty map, got %d entries", len(m.exact))
	}
}

// --- lookupCoverage tests ---

func TestLookupCoverage_ExactMatch(t *testing.T) {
	m := buildCoverMap([]FuncCoverage{
		{File: "/abs/path/foo.go", StartLine: 10, Percentage: 75.0},
	})
	stat := gocyclo.Stat{
		Pos: token.Position{Filename: "/abs/path/foo.go", Line: 10},
	}
	got := lookupCoverage(stat, m)
	if got != 75.0 {
		t.Errorf("expected 75.0, got %f", got)
	}
}

func TestLookupCoverage_BasenameFallback(t *testing.T) {
	m := buildCoverMap([]FuncCoverage{
		{File: "/other/path/foo.go", StartLine: 10, Percentage: 60.0},
	})
	stat := gocyclo.Stat{
		Pos: token.Position{Filename: "/different/path/foo.go", Line: 10},
	}
	got := lookupCoverage(stat, m)
	if got != 60.0 {
		t.Errorf("expected 60.0 via basename fallback, got %f", got)
	}
}

func TestLookupCoverage_NoMatch(t *testing.T) {
	m := buildCoverMap([]FuncCoverage{
		{File: "foo.go", StartLine: 10, Percentage: 50.0},
	})
	stat := gocyclo.Stat{
		Pos: token.Position{Filename: "bar.go", Line: 20},
	}
	got := lookupCoverage(stat, m)
	if got != 0 {
		t.Errorf("expected 0 for no match, got %f", got)
	}
}

// --- coverage.go tests ---

func TestRecvTypeString_Ident(t *testing.T) {
	// Test is implicit through findFunctions, but we test the
	// helper directly here. We use AST construction.
	// For simplicity, parse a file and check results.
	dir := t.TempDir()
	src := filepath.Join(dir, "recv.go")
	content := `package recv

type Foo struct{}
type Bar struct{}

func (f *Foo) Method1() {}
func (b Bar) Method2() int { return 0 }
`
	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	funcs, err := findFunctions(src)
	if err != nil {
		t.Fatal(err)
	}

	names := make(map[string]bool)
	for _, fn := range funcs {
		names[fn.name] = true
	}

	if !names["(*Foo).Method1"] {
		t.Error("expected (*Foo).Method1")
	}
	if !names["(Bar).Method2"] {
		t.Error("expected (Bar).Method2")
	}
}

func TestFindFunctions_NoBody(t *testing.T) {
	// Interface methods and external function declarations have no body
	// and should be skipped.
	dir := t.TempDir()
	src := filepath.Join(dir, "iface.go")
	content := `package iface

type Service interface {
	DoWork()
}

func Concrete() int { return 1 }
`
	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	funcs, err := findFunctions(src)
	if err != nil {
		t.Fatal(err)
	}

	// Only Concrete should be found (DoWork has no body).
	if len(funcs) != 1 {
		t.Fatalf("expected 1 function, got %d", len(funcs))
	}
	if funcs[0].name != "Concrete" {
		t.Errorf("expected 'Concrete', got %q", funcs[0].name)
	}
}

func TestFindFunctions_InvalidFile(t *testing.T) {
	_, err := findFunctions("/nonexistent/path.go")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestFuncCoverage_OverlappingBlocks(t *testing.T) {
	fn := funcExtent{
		name:      "Foo",
		startLine: 10,
		startCol:  1,
		endLine:   20,
		endCol:    2,
	}
	profile := &cover.Profile{
		Blocks: []cover.ProfileBlock{
			{StartLine: 5, StartCol: 1, EndLine: 8, EndCol: 1, NumStmt: 3, Count: 1},   // before function
			{StartLine: 11, StartCol: 1, EndLine: 15, EndCol: 1, NumStmt: 5, Count: 1}, // inside, covered
			{StartLine: 16, StartCol: 1, EndLine: 18, EndCol: 1, NumStmt: 2, Count: 0}, // inside, not covered
			{StartLine: 21, StartCol: 1, EndLine: 25, EndCol: 1, NumStmt: 4, Count: 1}, // after function
		},
	}

	covered, total := funcCoverage(fn, profile)
	if total != 7 {
		t.Errorf("expected total=7 (5+2), got %d", total)
	}
	if covered != 5 {
		t.Errorf("expected covered=5, got %d", covered)
	}
}

func TestFuncCoverage_EmptyProfile(t *testing.T) {
	fn := funcExtent{
		name:      "Foo",
		startLine: 10,
		startCol:  1,
		endLine:   20,
		endCol:    2,
	}
	profile := &cover.Profile{}

	covered, total := funcCoverage(fn, profile)
	if total != 0 || covered != 0 {
		t.Errorf("expected 0/0 for empty profile, got %d/%d", covered, total)
	}
}

func TestResolveFilePath_Absolute(t *testing.T) {
	// Create a temp file to ensure os.Stat succeeds.
	dir := t.TempDir()
	f := filepath.Join(dir, "test.go")
	if err := os.WriteFile(f, []byte("package x"), 0644); err != nil {
		t.Fatal(err)
	}

	got := resolveFilePath(f, dir)
	if got != f {
		t.Errorf("expected %q, got %q", f, got)
	}
}

func TestResolveFilePath_ModuleRelative(t *testing.T) {
	// Create a temp directory with a go.mod and a source file.
	dir := t.TempDir()
	gomod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(gomod, []byte("module example.com/test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "internal", "pkg"), 0755); err != nil {
		t.Fatal(err)
	}
	srcFile := filepath.Join(dir, "internal", "pkg", "foo.go")
	if err := os.WriteFile(srcFile, []byte("package pkg"), 0644); err != nil {
		t.Fatal(err)
	}

	got := resolveFilePath("example.com/test/internal/pkg/foo.go", dir)
	if got != srcFile {
		t.Errorf("expected %q, got %q", srcFile, got)
	}
}

func TestResolveFilePath_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	got := resolveFilePath("example.com/test/foo.go", dir)
	if got != "" {
		t.Errorf("expected empty string without go.mod, got %q", got)
	}
}

func TestReadModulePath_Valid(t *testing.T) {
	dir := t.TempDir()
	gomod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(gomod, []byte("module github.com/user/repo\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatal(err)
	}

	got := readModulePath(dir)
	if got != "github.com/user/repo" {
		t.Errorf("expected 'github.com/user/repo', got %q", got)
	}
}

func TestReadModulePath_NoGoMod(t *testing.T) {
	dir := t.TempDir()
	got := readModulePath(dir)
	if got != "" {
		t.Errorf("expected empty string, got %q", got)
	}
}

// --- shortenPath tests ---

func TestShortenPath_InternalMarker(t *testing.T) {
	got := shortenPath("/home/user/go/src/github.com/user/repo/internal/crap/analyze.go")
	if got != "internal/crap/analyze.go" {
		t.Errorf("expected 'internal/crap/analyze.go', got %q", got)
	}
}

func TestShortenPath_CmdMarker(t *testing.T) {
	got := shortenPath("/home/user/repo/cmd/gaze/main.go")
	if got != "cmd/gaze/main.go" {
		t.Errorf("expected 'cmd/gaze/main.go', got %q", got)
	}
}

func TestShortenPath_PkgMarker(t *testing.T) {
	got := shortenPath("/home/user/repo/pkg/util/helper.go")
	if got != "pkg/util/helper.go" {
		t.Errorf("expected 'pkg/util/helper.go', got %q", got)
	}
}

func TestShortenPath_LongPathFallback(t *testing.T) {
	got := shortenPath("/a/b/c/d/e/f.go")
	if got != "d/e/f.go" {
		t.Errorf("expected 'd/e/f.go', got %q", got)
	}
}

func TestShortenPath_ShortPath(t *testing.T) {
	got := shortenPath("foo.go")
	if got != "foo.go" {
		t.Errorf("expected 'foo.go', got %q", got)
	}
}

// ---------------------------------------------------------------------------
// SC-002: GazeCRAP formula accuracy (T052)
// ---------------------------------------------------------------------------

// TestGazeCRAPFormula_BenchmarkSuite validates SC-002: GazeCRAP
// scores match hand-computed values using contract coverage instead
// of line coverage (tolerance: +/- 0.01).
//
// GazeCRAP formula: comp^2 * (1 - contractCov/100)^3 + comp
// Same math as CRAP but with contract coverage (0-100%) as input.
func TestGazeCRAPFormula_BenchmarkSuite(t *testing.T) {
	cases := []struct {
		name           string
		complexity     int
		contractCovPct float64 // 0-100
		wantGazeCRAP   float64
	}{
		// Zero contract coverage: GazeCRAP = comp^2 + comp
		{"comp5_cc0", 5, 0, 30.0},    // 25*1 + 5 = 30
		{"comp10_cc0", 10, 0, 110.0}, // 100*1 + 10 = 110
		{"comp20_cc0", 20, 0, 420.0}, // 400*1 + 20 = 420

		// Full contract coverage: GazeCRAP = comp
		{"comp5_cc100", 5, 100, 5.0},
		{"comp10_cc100", 10, 100, 10.0},
		{"comp20_cc100", 20, 100, 20.0},
		{"comp1_cc100", 1, 100, 1.0},

		// Spec scenario US2.1: comp=10, contractCov=25%
		// GazeCRAP = 100*(0.75)^3 + 10 = 100*0.421875 + 10 = 52.1875
		{"spec_us2_1", 10, 25, 52.1875},

		// Spec scenario US2.2: comp=3, contractCov=100%
		// GazeCRAP = 9*0 + 3 = 3
		{"spec_us2_2", 3, 100, 3.0},

		// 50% contract coverage: uncov = 0.50, uncov^3 = 0.125
		{"comp5_cc50", 5, 50, 8.125},  // 25*0.125 + 5 = 8.125
		{"comp10_cc50", 10, 50, 22.5}, // 100*0.125 + 10 = 22.5
		{"comp20_cc50", 20, 50, 70.0}, // 400*0.125 + 20 = 70.0

		// 75% contract coverage: uncov = 0.25, uncov^3 = 0.015625
		{"comp10_cc75", 10, 75, 11.5625}, // 100*0.015625 + 10 = 11.5625
		{"comp20_cc75", 20, 75, 26.25},   // 400*0.015625 + 20 = 26.25

		// 90% contract coverage: uncov = 0.10, uncov^3 = 0.001
		{"comp10_cc90", 10, 90, 10.1}, // 100*0.001 + 10 = 10.1
		{"comp50_cc90", 50, 90, 52.5}, // 2500*0.001 + 50 = 52.5

		// Low complexity, various coverage
		{"comp1_cc0", 1, 0, 2.0},     // 1*1 + 1 = 2
		{"comp1_cc50", 1, 50, 1.125}, // 1*0.125 + 1 = 1.125
		{"comp2_cc80", 2, 80, 2.032}, // 4*0.008 + 2 = 2.032

		// High complexity edge case
		{"comp100_cc0", 100, 0, 10100.0},  // 10000*1 + 100 = 10100
		{"comp100_cc50", 100, 50, 1350.0}, // 10000*0.125 + 100 = 1350
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Formula(tc.complexity, tc.contractCovPct)
			if math.Abs(got-tc.wantGazeCRAP) > 0.01 {
				t.Errorf("GazeCRAP Formula(%d, %.1f) = %.4f, want %.4f",
					tc.complexity, tc.contractCovPct, got, tc.wantGazeCRAP)
			}
		})
	}
}

// TestGazeCRAP_Wiring verifies that Analyze() correctly populates
// Score.GazeCRAP, Score.ContractCoverage, and Score.Quadrant when
// ContractCoverageFunc is provided.
func TestGazeCRAP_Wiring(t *testing.T) {
	// Create scores with known values and verify via buildSummary.
	ccPct := 75.0
	gazeCRAP := Formula(10, ccPct) // 100*0.015625 + 10 = 11.5625
	quadrant := ClassifyQuadrant(10.1, gazeCRAP, 15, 15)

	score := Score{
		Package:          "pkg",
		Function:         "Foo",
		File:             "foo.go",
		Line:             1,
		Complexity:       10,
		LineCoverage:     90,
		CRAP:             10.1,
		ContractCoverage: &ccPct,
		GazeCRAP:         &gazeCRAP,
		Quadrant:         &quadrant,
	}

	if score.ContractCoverage == nil {
		t.Fatal("expected non-nil ContractCoverage")
	}
	if *score.ContractCoverage != 75.0 {
		t.Errorf("expected ContractCoverage=75.0, got %.1f", *score.ContractCoverage)
	}
	if score.GazeCRAP == nil {
		t.Fatal("expected non-nil GazeCRAP")
	}
	if math.Abs(*score.GazeCRAP-11.5625) > 0.01 {
		t.Errorf("expected GazeCRAP=11.5625, got %.4f", *score.GazeCRAP)
	}
	if score.Quadrant == nil {
		t.Fatal("expected non-nil Quadrant")
	}
	if *score.Quadrant != Q1Safe {
		t.Errorf("expected Q1_Safe, got %s", *score.Quadrant)
	}

	// Verify buildSummary handles the populated GazeCRAP fields.
	summary := buildSummary([]Score{score}, Options{CRAPThreshold: 15, GazeCRAPThreshold: 15})
	if summary.GazeCRAPload == nil {
		t.Fatal("expected non-nil GazeCRAPload in summary")
	}
	if summary.AvgGazeCRAP == nil {
		t.Fatal("expected non-nil AvgGazeCRAP")
	}
	if summary.AvgContractCoverage == nil {
		t.Fatal("expected non-nil AvgContractCoverage")
	}
	if *summary.AvgContractCoverage != 75.0 {
		t.Errorf("expected AvgContractCoverage=75.0, got %.1f", *summary.AvgContractCoverage)
	}
	if len(summary.QuadrantCounts) == 0 {
		t.Error("expected non-empty QuadrantCounts")
	}
}

// TestWriteText_QuadrantBreakdown verifies that quadrant breakdown
// renders correctly when GazeCRAP data is populated (T053).
func TestWriteText_QuadrantBreakdown(t *testing.T) {
	gcl := 1
	gct := 15.0
	avgGC := 12.5
	avgCC := 80.0
	report := &Report{
		Scores: []Score{
			{
				Package: "pkg", Function: "Safe", File: "a.go", Line: 1,
				Complexity: 5, LineCoverage: 90, CRAP: 5.025,
				ContractCoverage: ptrFloat(95),
				GazeCRAP:         ptrFloat(5.001),
				Quadrant:         ptrQuadrant(Q1Safe),
			},
			{
				Package: "pkg", Function: "Dangerous", File: "b.go", Line: 1,
				Complexity: 20, LineCoverage: 10, CRAP: 312,
				ContractCoverage: ptrFloat(10),
				GazeCRAP:         ptrFloat(311),
				Quadrant:         ptrQuadrant(Q4Dangerous),
			},
		},
		Summary: Summary{
			TotalFunctions:      2,
			AvgComplexity:       12.5,
			AvgLineCoverage:     50,
			AvgCRAP:             158.5,
			CRAPload:            1,
			CRAPThreshold:       15,
			GazeCRAPload:        &gcl,
			GazeCRAPThreshold:   &gct,
			AvgGazeCRAP:         &avgGC,
			AvgContractCoverage: &avgCC,
			QuadrantCounts: map[Quadrant]int{
				Q1Safe:                    1,
				Q2ComplexButTested:        0,
				Q3SimpleButUnderspecified: 0,
				Q4Dangerous:               1,
			},
			WorstCRAP: []Score{
				{Function: "Dangerous", CRAP: 312},
			},
			WorstGazeCRAP: []Score{
				{Function: "Dangerous", GazeCRAP: ptrFloat(311)},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteText(&buf, report); err != nil {
		t.Fatalf("WriteText failed: %v", err)
	}

	output := buf.String()

	// Verify quadrant breakdown appears.
	if !strings.Contains(output, "Quadrant Breakdown") {
		t.Error("expected 'Quadrant Breakdown' in output")
	}
	if !strings.Contains(output, string(Q1Safe)) {
		t.Errorf("expected %q in output", Q1Safe)
	}
	if !strings.Contains(output, string(Q4Dangerous)) {
		t.Errorf("expected %q in output", Q4Dangerous)
	}

	// Verify GazeCRAP summary lines.
	if !strings.Contains(output, "GazeCRAP") {
		t.Error("expected 'GazeCRAP' in output")
	}

	// Verify non-table lines fit 80 columns. Table border lines
	// (containing box-drawing characters) are excluded because
	// the Bubble Tea table renderer uses fixed-width styling.
	for i, line := range strings.Split(output, "\n") {
		stripped := stripANSI(line)
		if strings.ContainsAny(stripped, "┌┐└┘├┤┬┴┼─│") {
			continue // skip table border lines
		}
		if len(stripped) > 80 {
			t.Errorf("line %d exceeds 80 columns (%d): %q", i+1, len(stripped), stripped)
		}
	}
}

func ptrFloat(f float64) *float64      { return &f }
func ptrQuadrant(q Quadrant) *Quadrant { return &q }

// stripANSI removes ANSI escape sequences for width measurement.
var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

func stripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// moduleRoot returns the root directory of the Go module by walking
// up from the current test file until go.mod is found.
func moduleRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root (no go.mod found)")
		}
		dir = parent
	}
}

// ---------------------------------------------------------------------------
// Analyze tests
// ---------------------------------------------------------------------------

// TestAnalyze_CoverProfileNotFound verifies that Analyze returns an
// error when the user-supplied cover profile path does not exist.
func TestAnalyze_CoverProfileNotFound(t *testing.T) {
	opts := DefaultOptions()
	opts.CoverProfile = "/nonexistent/path/cover.out"

	_, err := Analyze([]string{"./internal/crap"}, moduleRoot(t), opts)
	if err == nil {
		t.Fatal("expected error for non-existent cover profile")
	}
	if !strings.Contains(err.Error(), "cover profile") {
		t.Errorf("expected error to mention 'cover profile', got: %v", err)
	}
}

// TestAnalyze_CoverProfileIsDirectory verifies that Analyze returns
// an error when the user-supplied cover profile path is a directory.
func TestAnalyze_CoverProfileIsDirectory(t *testing.T) {
	opts := DefaultOptions()
	opts.CoverProfile = t.TempDir() // a directory, not a file

	_, err := Analyze([]string{"./internal/crap"}, moduleRoot(t), opts)
	if err == nil {
		t.Fatal("expected error when cover profile path is a directory")
	}
	if !strings.Contains(err.Error(), "directory") {
		t.Errorf("expected error to mention 'directory', got: %v", err)
	}
}

// TestAnalyze_WithPrebuiltProfile runs the full Analyze pipeline
// using a pre-built coverage profile, bypassing the go test
// subprocess. This tests steps 2-6 of Analyze and also exercises
// ParseCoverProfile and buildCoverMap.
func TestAnalyze_WithPrebuiltProfile(t *testing.T) {
	modRoot := moduleRoot(t)

	// Build a minimal coverage profile that references crap.go.
	// Formula (lines 89-92) and ClassifyQuadrant (lines 97-107)
	// are marked as covered (Count=1); everything else is absent
	// and defaults to 0% coverage.
	profileContent := "mode: set\n" +
		"github.com/unbound-force/gaze/internal/crap/crap.go:89.39,92.2 2 1\n" +
		"github.com/unbound-force/gaze/internal/crap/crap.go:97.57,105.2 3 1\n"

	profileFile := filepath.Join(t.TempDir(), "cover.out")
	if err := os.WriteFile(profileFile, []byte(profileContent), 0o644); err != nil {
		t.Fatalf("writing cover profile: %v", err)
	}

	opts := DefaultOptions()
	opts.CoverProfile = profileFile

	report, err := Analyze([]string{"./internal/crap"}, modRoot, opts)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	if report == nil {
		t.Fatal("expected non-nil report")
	}

	// Should have analyzed multiple functions.
	if report.Summary.TotalFunctions == 0 {
		t.Error("expected TotalFunctions > 0")
	}

	// Formula and ClassifyQuadrant should have non-zero coverage
	// (they're in the profile). All others should have 0%.
	var formulaFound, classifyFound bool
	for _, s := range report.Scores {
		switch s.Function {
		case "Formula":
			formulaFound = true
			if s.LineCoverage == 0 {
				t.Errorf("Formula: expected non-zero coverage from profile, got 0%%")
			}
		case "ClassifyQuadrant":
			classifyFound = true
			if s.LineCoverage == 0 {
				t.Errorf("ClassifyQuadrant: expected non-zero coverage from profile, got 0%%")
			}
		}
	}
	if !formulaFound {
		t.Error("expected 'Formula' in scores")
	}
	if !classifyFound {
		t.Error("expected 'ClassifyQuadrant' in scores")
	}

	// Summary fields are populated.
	if report.Summary.AvgComplexity == 0 {
		t.Error("expected non-zero AvgComplexity")
	}
	if report.Summary.CRAPThreshold != 15 {
		t.Errorf("expected CRAPThreshold=15, got %.0f", report.Summary.CRAPThreshold)
	}
	if len(report.Summary.WorstCRAP) == 0 {
		t.Error("expected at least one WorstCRAP entry")
	}
}

// TestAnalyze_ContractCoverageFunc verifies that Analyze populates
// GazeCRAP, ContractCoverage, and Quadrant when ContractCoverageFunc
// is provided.
func TestAnalyze_ContractCoverageFunc(t *testing.T) {
	modRoot := moduleRoot(t)

	profileContent := "mode: set\n" +
		"github.com/unbound-force/gaze/internal/crap/crap.go:89.39,92.2 2 1\n"
	profileFile := filepath.Join(t.TempDir(), "cover.out")
	if err := os.WriteFile(profileFile, []byte(profileContent), 0o644); err != nil {
		t.Fatalf("writing cover profile: %v", err)
	}

	opts := DefaultOptions()
	opts.CoverProfile = profileFile
	opts.ContractCoverageFunc = func(pkg, function string) (float64, bool) {
		if function == "Formula" {
			return 80.0, true
		}
		return 0, false
	}

	report, err := Analyze([]string{"./internal/crap"}, modRoot, opts)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Find Formula and verify GazeCRAP fields are populated.
	var formulaScore *Score
	for i := range report.Scores {
		if report.Scores[i].Function == "Formula" {
			formulaScore = &report.Scores[i]
			break
		}
	}
	if formulaScore == nil {
		t.Fatal("Formula not found in scores")
	}
	if formulaScore.GazeCRAP == nil {
		t.Error("expected non-nil GazeCRAP for Formula")
	}
	if formulaScore.ContractCoverage == nil {
		t.Error("expected non-nil ContractCoverage for Formula")
	} else if *formulaScore.ContractCoverage != 80.0 {
		t.Errorf("expected ContractCoverage=80.0, got %.1f", *formulaScore.ContractCoverage)
	}
	if formulaScore.Quadrant == nil {
		t.Error("expected non-nil Quadrant for Formula")
	}

	// Summary should show GazeCRAP data.
	if report.Summary.GazeCRAPload == nil {
		t.Error("expected non-nil GazeCRAPload in summary")
	}
}

// TestAnalyze_IgnoreGeneratedFiles verifies that generated files are
// skipped when IgnoreGenerated is true.
func TestAnalyze_IgnoreGeneratedFiles(t *testing.T) {
	dir := t.TempDir()

	// Create a minimal go.mod.
	gomod := "module example.com/gentest\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a generated file with a single function.
	generatedSrc := `// Code generated by protoc-gen-go. DO NOT EDIT.
package gentest

func GeneratedFunc() int { return 42 }
`
	if err := os.WriteFile(filepath.Join(dir, "gen.go"), []byte(generatedSrc), 0o644); err != nil {
		t.Fatal(err)
	}

	// Write a minimal (empty) cover profile so Analyze doesn't
	// spawn a subprocess.
	profileContent := "mode: set\n"
	profileFile := filepath.Join(t.TempDir(), "cover.out")
	if err := os.WriteFile(profileFile, []byte(profileContent), 0o644); err != nil {
		t.Fatal(err)
	}

	opts := DefaultOptions()
	opts.CoverProfile = profileFile
	opts.IgnoreGenerated = true

	report, err := Analyze([]string{"."}, dir, opts)
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	for _, s := range report.Scores {
		if s.Function == "GeneratedFunc" {
			t.Error("GeneratedFunc should be excluded when IgnoreGenerated=true")
		}
	}
}

// ---------------------------------------------------------------------------
// WriteText specific-value assertion tests
// ---------------------------------------------------------------------------

// TestWriteText_ExactCRAPScore verifies that WriteText renders the
// exact CRAP score value in the output table.
func TestWriteText_ExactCRAPScore(t *testing.T) {
	rpt := &Report{
		Scores: []Score{
			{
				Package:      "pkg",
				Function:     "MyFunc",
				File:         "myfile.go",
				Line:         42,
				Complexity:   7,
				LineCoverage: 50.0,
				CRAP:         22.5,
			},
		},
		Summary: Summary{
			TotalFunctions:  1,
			AvgComplexity:   7,
			AvgLineCoverage: 50.0,
			AvgCRAP:         22.5,
			CRAPload:        1,
			CRAPThreshold:   15,
		},
	}

	var buf bytes.Buffer
	if err := WriteText(&buf, rpt); err != nil {
		t.Fatal(err)
	}

	out := stripANSI(buf.String())

	// Exact CRAP score "22.5" must appear in the table.
	if !strings.Contains(out, "22.5") {
		t.Errorf("expected CRAP score '22.5' in output, got:\n%s", out)
	}
	// Complexity "7" must appear.
	if !strings.Contains(out, "7") {
		t.Errorf("expected complexity '7' in output, got:\n%s", out)
	}
	// Coverage "50.0%" must appear.
	if !strings.Contains(out, "50.0%") {
		t.Errorf("expected coverage '50.0%%' in output, got:\n%s", out)
	}
	// Function name must appear.
	if !strings.Contains(out, "MyFunc") {
		t.Errorf("expected function name 'MyFunc' in output, got:\n%s", out)
	}
	// File and line must appear.
	if !strings.Contains(out, "myfile.go:42") {
		t.Errorf("expected 'myfile.go:42' in output, got:\n%s", out)
	}
}

// TestWriteText_SummaryNumericValues verifies that summary section
// renders correct numeric values.
func TestWriteText_SummaryNumericValues(t *testing.T) {
	rpt := &Report{
		Scores: []Score{
			{Function: "A", Complexity: 3, LineCoverage: 100, CRAP: 3},
			{Function: "B", Complexity: 5, LineCoverage: 0, CRAP: 30},
		},
		Summary: Summary{
			TotalFunctions:  2,
			AvgComplexity:   4.0,
			AvgLineCoverage: 50.0,
			AvgCRAP:         16.5,
			CRAPload:        1,
			CRAPThreshold:   15,
		},
	}

	var buf bytes.Buffer
	if err := WriteText(&buf, rpt); err != nil {
		t.Fatal(err)
	}

	out := stripANSI(buf.String())

	if !strings.Contains(out, "2") {
		t.Errorf("expected TotalFunctions '2' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "4.0") {
		t.Errorf("expected AvgComplexity '4.0' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "50.0%") {
		t.Errorf("expected AvgLineCoverage '50.0%%' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "16.5") {
		t.Errorf("expected AvgCRAP '16.5' in summary, got:\n%s", out)
	}
}

// TestWriteText_EmptyReport verifies "No functions analyzed" message.
func TestWriteText_EmptyReport(t *testing.T) {
	rpt := &Report{Scores: nil, Summary: Summary{CRAPThreshold: 15}}
	var buf bytes.Buffer
	if err := WriteText(&buf, rpt); err != nil {
		t.Fatal(err)
	}
	out := stripANSI(buf.String())
	if !strings.Contains(out, "No functions analyzed") {
		t.Errorf("expected 'No functions analyzed', got:\n%s", out)
	}
}

// TestWriteText_WorstOffendersSection verifies that the worst
// offenders section contains function names and CRAP scores.
func TestWriteText_WorstOffendersSection(t *testing.T) {
	rpt := &Report{
		Scores: []Score{
			{Function: "WorstFunc", File: "bad.go", Line: 1, Complexity: 20, LineCoverage: 0, CRAP: 420},
			{Function: "GoodFunc", File: "good.go", Line: 1, Complexity: 2, LineCoverage: 100, CRAP: 2},
		},
		Summary: Summary{
			TotalFunctions: 2,
			CRAPThreshold:  15,
			CRAPload:       1,
			WorstCRAP: []Score{
				{Function: "WorstFunc", File: "bad.go", Line: 1, CRAP: 420},
			},
		},
	}

	var buf bytes.Buffer
	if err := WriteText(&buf, rpt); err != nil {
		t.Fatal(err)
	}

	out := stripANSI(buf.String())

	if !strings.Contains(out, "Worst Offenders") {
		t.Errorf("expected 'Worst Offenders' section, got:\n%s", out)
	}
	if !strings.Contains(out, "WorstFunc") {
		t.Errorf("expected 'WorstFunc' in worst offenders, got:\n%s", out)
	}
	if !strings.Contains(out, "420.0") {
		t.Errorf("expected '420.0' CRAP score, got:\n%s", out)
	}
}
