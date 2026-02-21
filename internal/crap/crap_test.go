package crap

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go/token"

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
