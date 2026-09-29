package aireport

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/v2/internal/adapter"
	"github.com/unbound-force/gaze/v2/internal/analysis"
	"github.com/unbound-force/gaze/v2/internal/config"
	"github.com/unbound-force/gaze/v2/internal/quality"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// TestRunCRAPStep_RealPackage verifies that runCRAPStep successfully runs on
// a real package and returns a non-nil JSON payload.
// Guarded by testing.Short() — spawns the Go analysis pipeline.
func TestRunCRAPStep_RealPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs real CRAP analysis pipeline")
	}
	modRoot := findModuleRoot(t)
	res, err := runCRAPStep(
		[]string{"github.com/unbound-force/gaze/v2/internal/config"},
		modRoot,
		"", // no pre-generated profile — use internal generation
		io.Discard,
		nil,   // no contract coverage callback
		false, // short
	)
	if err != nil {
		t.Fatalf("runCRAPStep: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil crapStepResult")
	}
	if res.JSON == nil {
		t.Error("expected non-nil JSON from runCRAPStep")
	}
}

// TestRunDocscanStep_RealModuleDir verifies that RunDocscanStep runs without
// error on the module root and returns a non-nil JSON payload.
// Guarded by testing.Short().
func TestRunDocscanStep_RealModuleDir(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs real docscan pipeline")
	}
	modRoot := findModuleRoot(t)
	raw, err := RunDocscanStep(context.Background(), modRoot, nil, io.Discard)
	if err != nil {
		t.Fatalf("RunDocscanStep: %v", err)
	}
	if raw == nil {
		t.Error("expected non-nil JSON from RunDocscanStep")
	}
}

// TestRunCRAPStep_WithCoverProfile verifies that runCRAPStep accepts a
// pre-generated coverage profile and produces a non-nil JSON result (FR-001,
// FR-002). Uses the static fixture at testdata/sample.coverprofile, which
// records one covered statement in internal/crap/crap.go.
// Guarded by testing.Short() — calls crap.Analyze which loads Go packages.
func TestRunCRAPStep_WithCoverProfile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: calls crap.Analyze which loads Go packages")
	}
	// Locate the testdata fixture relative to this file's directory.
	_, thisFile, _, _ := runtime.Caller(0)
	fixture := filepath.Join(filepath.Dir(thisFile), "testdata", "sample.coverprofile")

	modRoot := findModuleRoot(t)
	res, err := runCRAPStep(
		[]string{"github.com/unbound-force/gaze/v2/internal/crap"},
		modRoot,
		fixture,
		io.Discard,
		nil,   // no contract coverage callback
		false, // short
	)
	if err != nil {
		t.Fatalf("runCRAPStep with coverprofile: %v", err)
	}
	if res == nil {
		t.Fatal("expected non-nil crapStepResult")
	}
	if res.JSON == nil {
		t.Error("expected non-nil JSON from runCRAPStep with pre-generated profile")
	}
}

// TestRunProductionPipeline_RealPackage verifies that runProductionPipeline
// returns a non-nil payload and exercises all four steps without panicking.
// Guarded by testing.Short() — runs the full four-step pipeline.
func TestRunProductionPipeline_RealPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs full four-step analysis pipeline")
	}
	modRoot := findModuleRoot(t)
	payload, err := runProductionPipeline(
		context.Background(),
		[]string{"github.com/unbound-force/gaze/v2/internal/config"},
		modRoot,
		"",    // no pre-generated profile — use internal generation
		false, // testShort
		io.Discard,
		pipelineStepFuncs{}, // zero value = real step functions
	)
	if err != nil {
		t.Fatalf("runProductionPipeline: %v", err)
	}
	if payload == nil {
		t.Fatal("expected non-nil ReportPayload")
	}
	// CRAP step must succeed for a real package.
	if payload.CRAP == nil && payload.Errors.CRAP == nil {
		t.Error("expected either CRAP JSON or CRAP error, got both nil")
	}
}

// ---------------------------------------------------------------------------
// Test helpers: synthetic data builders for DI tests
// ---------------------------------------------------------------------------

// syntheticAnalysisResult returns a minimal AnalysisResult with one side
// effect. When label is non-nil, the effect carries a Classification.
func syntheticAnalysisResult(pkg, fn string, label *taxonomy.ClassificationLabel) []taxonomy.AnalysisResult {
	se := taxonomy.SideEffect{
		ID:          "se-test0001",
		Type:        taxonomy.ReturnValue,
		Tier:        taxonomy.TierP0,
		Location:    "fake.go:1:1",
		Description: "returns int",
		Target:      "result",
	}
	if label != nil {
		se.Classification = &taxonomy.Classification{
			Label:      *label,
			Confidence: 90,
		}
	}
	return []taxonomy.AnalysisResult{{
		Target: taxonomy.FunctionTarget{
			Package:  pkg,
			Function: fn,
		},
		SideEffects: []taxonomy.SideEffect{se},
	}}
}

// syntheticQualityReport returns a minimal QualityReport for testing.
func syntheticQualityReport(fn string) taxonomy.QualityReport {
	return taxonomy.QualityReport{
		TestFunction: "Test_" + fn,
		TargetFunction: taxonomy.FunctionTarget{
			Function: fn,
		},
		ContractCoverage: taxonomy.ContractCoverage{Percentage: 80},
	}
}

// fakeDepsSuccess returns a qualityPipelineDeps where every function
// succeeds with minimal synthetic data. Individual fields can be
// overridden after construction.
func fakeDepsSuccess() qualityPipelineDeps {
	contractual := taxonomy.Contractual
	return qualityPipelineDeps{
		resolvePackagePaths: func(patterns []string, _ string) ([]string, error) {
			return patterns, nil
		},
		loadAndAnalyze: func(pattern string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
			return syntheticAnalysisResult(pattern, "Foo", &contractual), nil
		},
		classifyResults: func(results []taxonomy.AnalysisResult, _ string, _ *config.GazeConfig, _ []*packages.Package) ([]taxonomy.AnalysisResult, error) {
			return results, nil
		},
		loadTestPkg: func(_ string) (*packages.Package, error) {
			return &packages.Package{Name: "fake_test"}, nil
		},
		assess: func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
			return []taxonomy.QualityReport{syntheticQualityReport("Foo")}, &taxonomy.PackageSummary{
				TotalTests:              1,
				AverageContractCoverage: 80,
			}, nil
		},
		resolveModulePkgs: func(_ string) []*packages.Package {
			return []*packages.Package{{Name: "fake"}}
		},
		loadConfig: func(_ string, _ io.Writer) *config.GazeConfig {
			return config.DefaultConfig()
		},
	}
}

// ---------------------------------------------------------------------------
// Task 2.2: Unit tests for runQualityForPackage
// ---------------------------------------------------------------------------

func TestRunQualityForPackage_DI_Success(t *testing.T) {
	deps := fakeDepsSuccess()
	reports, degraded, skipped, _ := runQualityForPackage("fake/pkg", config.DefaultConfig(), nil, io.Discard, deps)
	if reports == nil {
		t.Fatal("expected non-nil reports on success path")
	}
	if len(reports) != 1 {
		t.Fatalf("expected 1 report, got %d", len(reports))
	}
	if degraded != "" {
		t.Errorf("expected empty degraded string, got %q", degraded)
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}
}

func TestRunQualityForPackage_DI_AnalysisError(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.loadAndAnalyze = func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		return nil, fmt.Errorf("analysis failed")
	}
	reports, degraded, skipped, _ := runQualityForPackage("fake/pkg", config.DefaultConfig(), nil, io.Discard, deps)
	if reports != nil {
		t.Errorf("expected nil reports on analysis error, got %d", len(reports))
	}
	if degraded != "" {
		t.Errorf("expected empty degraded string, got %q", degraded)
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}
}

func TestRunQualityForPackage_DI_EmptyResults(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.loadAndAnalyze = func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		return []taxonomy.AnalysisResult{}, nil
	}
	reports, degraded, skipped, _ := runQualityForPackage("fake/pkg", config.DefaultConfig(), nil, io.Discard, deps)
	if reports != nil {
		t.Errorf("expected nil reports on empty results, got %d", len(reports))
	}
	if degraded != "" {
		t.Errorf("expected empty degraded string, got %q", degraded)
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}
}

func TestRunQualityForPackage_DI_ClassifyError(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.classifyResults = func(_ []taxonomy.AnalysisResult, _ string, _ *config.GazeConfig, _ []*packages.Package) ([]taxonomy.AnalysisResult, error) {
		return nil, fmt.Errorf("classify failed")
	}
	reports, degraded, skipped, _ := runQualityForPackage("fake/pkg", config.DefaultConfig(), nil, io.Discard, deps)
	if reports != nil {
		t.Errorf("expected nil reports on classify error, got %d", len(reports))
	}
	if degraded != "" {
		t.Errorf("expected empty degraded string, got %q", degraded)
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}
}

func TestRunQualityForPackage_DI_LoadTestError(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.loadTestPkg = func(_ string) (*packages.Package, error) {
		return nil, fmt.Errorf("no test package found")
	}
	reports, degraded, skipped, _ := runQualityForPackage("fake/pkg", config.DefaultConfig(), nil, io.Discard, deps)
	if reports != nil {
		t.Errorf("expected nil reports on loadTestPkg error, got %d", len(reports))
	}
	if degraded != "" {
		t.Errorf("expected empty degraded string, got %q", degraded)
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}
}

func TestRunQualityForPackage_DI_AssessError(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
		return nil, nil, fmt.Errorf("assess failed")
	}
	reports, degraded, skipped, _ := runQualityForPackage("fake/pkg", config.DefaultConfig(), nil, io.Discard, deps)
	if reports != nil {
		t.Errorf("expected nil reports on assess error, got %d", len(reports))
	}
	if degraded != "" {
		t.Errorf("expected empty degraded string, got %q", degraded)
	}
	if skipped != 0 {
		t.Errorf("expected 0 skipped, got %d", skipped)
	}
}

func TestRunQualityForPackage_DI_SSADegraded(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
		return []taxonomy.QualityReport{syntheticQualityReport("Foo")}, &taxonomy.PackageSummary{
			TotalTests:              1,
			AverageContractCoverage: 50,
			SSADegraded:             true,
			SkippedTests:            2,
			SkippedTestNames:        []string{"TestSkipA", "TestSkipB"},
		}, nil
	}
	reports, degraded, skipped, skippedNames := runQualityForPackage("fake/pkg", config.DefaultConfig(), nil, io.Discard, deps)
	if reports == nil {
		t.Fatal("expected non-nil reports on SSA degradation")
	}
	if degraded != "fake/pkg" {
		t.Errorf("expected degraded=%q, got %q", "fake/pkg", degraded)
	}
	if skipped != 2 {
		t.Errorf("expected 2 skipped, got %d", skipped)
	}
	if len(skippedNames) != 2 {
		t.Errorf("expected 2 skipped names, got %d", len(skippedNames))
	}
}

// ---------------------------------------------------------------------------
// Task 2.3: Unit tests for runQualityStep
// ---------------------------------------------------------------------------

func TestRunQualityStep_DI_SinglePackage(t *testing.T) {
	deps := fakeDepsSuccess()
	result, err := runQualityStep([]string{"fake/pkg"}, "/tmp", io.Discard, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil qualityStepResult")
	}
	if result.JSON == nil {
		t.Error("expected non-nil JSON")
	}
	if result.SSADegraded {
		t.Error("expected SSADegraded=false")
	}
}

func TestRunQualityStep_DI_MultiplePackages(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return []string{"fake/pkg1", "fake/pkg2"}, nil
	}
	result, err := runQualityStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil qualityStepResult")
	}
	if result.JSON == nil {
		t.Error("expected non-nil JSON")
	}
	// AvgContractCoverage should be computed from 2 reports.
	// Both fake reports return 80% coverage, so the average should be 80.
	if result.AvgContractCoverage == nil || *result.AvgContractCoverage != 80 {
		t.Errorf("expected AvgContractCoverage=80, got %v", result.AvgContractCoverage)
	}
}

func TestRunQualityStep_DI_ResolveError(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return nil, fmt.Errorf("resolve failed")
	}
	_, err := runQualityStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err == nil {
		t.Fatal("expected error on resolve failure")
	}
	if !strings.Contains(err.Error(), "resolving packages") {
		t.Errorf("expected 'resolving packages' in error, got: %v", err)
	}
}

func TestRunQualityStep_DI_ResolveEmpty(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return []string{}, nil
	}
	_, err := runQualityStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err == nil {
		t.Fatal("expected error on empty resolve result")
	}
	if !strings.Contains(err.Error(), "no packages matched") {
		t.Errorf("expected 'no packages matched' in error, got: %v", err)
	}
}

func TestRunQualityStep_DI_SSADegradation(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
		return []taxonomy.QualityReport{syntheticQualityReport("Foo")}, &taxonomy.PackageSummary{
			TotalTests:              1,
			AverageContractCoverage: 50,
			SSADegraded:             true,
		}, nil
	}
	result, err := runQualityStep([]string{"fake/pkg"}, "/tmp", io.Discard, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.SSADegraded {
		t.Error("expected SSADegraded=true")
	}
	if len(result.SSADegradedPackages) != 1 || result.SSADegradedPackages[0] != "fake/pkg" {
		t.Errorf("expected SSADegradedPackages=[fake/pkg], got %v", result.SSADegradedPackages)
	}
}

func TestRunQualityStep_DI_SkippedTestsPropagation(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.assess = func(_ []taxonomy.AnalysisResult, _ *packages.Package, _ quality.Options) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
		return nil, &taxonomy.PackageSummary{
			SkippedTests:     3,
			SkippedTestNames: []string{"TestA", "TestB", "TestC"},
		}, nil
	}
	result, err := runQualityStep([]string{"fake/pkg"}, "/tmp", io.Discard, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.SkippedTests != 3 {
		t.Errorf("expected SkippedTests=3, got %d", result.SkippedTests)
	}
}

// ---------------------------------------------------------------------------
// Task 2.4: Unit tests for runClassifyStep
// ---------------------------------------------------------------------------

func TestRunClassifyStep_DI_Success(t *testing.T) {
	contractual := taxonomy.Contractual
	ambiguous := taxonomy.Ambiguous
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return []string{"fake/pkg"}, nil
	}
	deps.loadAndAnalyze = func(_ string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		// Return results with 2 contractual + 1 ambiguous effects.
		return []taxonomy.AnalysisResult{{
			Target: taxonomy.FunctionTarget{Package: "fake/pkg", Function: "Foo"},
			SideEffects: []taxonomy.SideEffect{
				{ID: "se-1", Type: taxonomy.ReturnValue, Classification: &taxonomy.Classification{Label: contractual, Confidence: 90}},
				{ID: "se-2", Type: taxonomy.ErrorReturn, Classification: &taxonomy.Classification{Label: contractual, Confidence: 85}},
				{ID: "se-3", Type: taxonomy.MapMutation, Classification: &taxonomy.Classification{Label: ambiguous, Confidence: 55}},
			},
		}}, nil
	}
	deps.classifyResults = func(results []taxonomy.AnalysisResult, _ string, _ *config.GazeConfig, _ []*packages.Package) ([]taxonomy.AnalysisResult, error) {
		return results, nil // passthrough — labels already set
	}
	result, err := runClassifyStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Contractual != 2 {
		t.Errorf("expected Contractual=2, got %d", result.Contractual)
	}
	if result.Ambiguous != 1 {
		t.Errorf("expected Ambiguous=1, got %d", result.Ambiguous)
	}
	if result.Incidental != 0 {
		t.Errorf("expected Incidental=0, got %d", result.Incidental)
	}
	if result.JSON == nil {
		t.Error("expected non-nil JSON")
	}
}

func TestRunClassifyStep_DI_ResolveError(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return nil, fmt.Errorf("resolve failed")
	}
	_, err := runClassifyStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err == nil {
		t.Fatal("expected error on resolve failure")
	}
	if !strings.Contains(err.Error(), "resolving packages for classification") {
		t.Errorf("expected 'resolving packages for classification' in error, got: %v", err)
	}
}

func TestRunClassifyStep_DI_ResolveEmpty(t *testing.T) {
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return []string{}, nil
	}
	_, err := runClassifyStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err == nil {
		t.Fatal("expected error on empty resolve result")
	}
	if !strings.Contains(err.Error(), "no packages matched") {
		t.Errorf("expected 'no packages matched' in error, got: %v", err)
	}
}

func TestRunClassifyStep_DI_AnalysisErrorSkip(t *testing.T) {
	contractual := taxonomy.Contractual
	callCount := 0
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return []string{"fake/pkg1", "fake/pkg2"}, nil
	}
	deps.loadAndAnalyze = func(pattern string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		callCount++
		if pattern == "fake/pkg1" {
			return nil, fmt.Errorf("analysis failed for pkg1")
		}
		return syntheticAnalysisResult(pattern, "Bar", &contractual), nil
	}
	result, err := runClassifyStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if callCount != 2 {
		t.Errorf("expected loadAndAnalyze called 2 times, got %d", callCount)
	}
	// Only pkg2's results should contribute (1 contractual effect).
	if result.Contractual != 1 {
		t.Errorf("expected Contractual=1 (from pkg2 only), got %d", result.Contractual)
	}
}

func TestRunClassifyStep_DI_EmptyResultsSkip(t *testing.T) {
	contractual := taxonomy.Contractual
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return []string{"fake/empty", "fake/notempty"}, nil
	}
	deps.loadAndAnalyze = func(pattern string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		if pattern == "fake/empty" {
			return []taxonomy.AnalysisResult{}, nil
		}
		return syntheticAnalysisResult(pattern, "Baz", &contractual), nil
	}
	result, err := runClassifyStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Contractual != 1 {
		t.Errorf("expected Contractual=1 (from notempty only), got %d", result.Contractual)
	}
}

func TestRunClassifyStep_DI_ClassifyErrorSkip(t *testing.T) {
	contractual := taxonomy.Contractual
	deps := fakeDepsSuccess()
	deps.resolvePackagePaths = func(_ []string, _ string) ([]string, error) {
		return []string{"fake/bad", "fake/good"}, nil
	}
	deps.loadAndAnalyze = func(pattern string, _ analysis.Options) ([]taxonomy.AnalysisResult, error) {
		return syntheticAnalysisResult(pattern, "Qux", &contractual), nil
	}
	deps.classifyResults = func(results []taxonomy.AnalysisResult, pkgPath string, _ *config.GazeConfig, _ []*packages.Package) ([]taxonomy.AnalysisResult, error) {
		if pkgPath == "fake/bad" {
			return nil, fmt.Errorf("classify failed for bad pkg")
		}
		return results, nil
	}
	result, err := runClassifyStep([]string{"./..."}, "/tmp", io.Discard, deps)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Only fake/good's results should contribute.
	if result.Contractual != 1 {
		t.Errorf("expected Contractual=1 (from good only), got %d", result.Contractual)
	}
}

var (
	docscanTestBinaryPath string
	docscanTestBuildOnce  sync.Once
	docscanTestBuildErr   error
)

// buildDocscanTestFakeAnalyzer lazily builds the fake analyzer binary and
// returns its path. The build is shared across tests via sync.Once.
func buildDocscanTestFakeAnalyzer(t *testing.T) string {
	t.Helper()
	docscanTestBuildOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "gaze-docscan-test-*")
		if err != nil {
			docscanTestBuildErr = err
			return
		}
		binPath := filepath.Join(tmpDir, "fake_analyzer")
		cmd := exec.Command("go", "build", "-o", binPath, "./testdata/fake_analyzer/")
		cmd.Dir = filepath.Join("..", "protocol")
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			docscanTestBuildErr = err
			return
		}
		docscanTestBinaryPath = binPath
	})
	if docscanTestBuildErr != nil {
		t.Fatalf("building fake_analyzer: %v", docscanTestBuildErr)
	}
	return docscanTestBinaryPath
}

// TestRunDocscanStep_WithSession verifies that a non-nil analyzer session
// yields a populated api_coverage section in the docscan envelope.
func TestRunDocscanStep_WithSession(t *testing.T) {
	bin := buildDocscanTestFakeAnalyzer(t)
	moduleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module fake\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	var stderr bytes.Buffer
	sess := adapter.NewSession(bin, []string{"--stdio"}, moduleDir, []string{"./..."}, &stderr, config.DefaultConfig())
	if _, err := sess.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	defer func() { _ = sess.Close() }()

	raw, err := RunDocscanStep(context.Background(), moduleDir, sess, &stderr)
	if err != nil {
		t.Fatalf("RunDocscanStep: %v", err)
	}

	var env map[string]json.RawMessage
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	var cov map[string]interface{}
	if err := json.Unmarshal(env["api_coverage"], &cov); err != nil {
		t.Fatalf("unmarshal api_coverage: %v", err)
	}
	if cov["source"] != "doc_coverage" {
		t.Errorf("api_coverage source = %v, want doc_coverage", cov["source"])
	}
	if cov["total_symbols"] != float64(3) {
		t.Errorf("api_coverage total_symbols = %v, want 3", cov["total_symbols"])
	}
	if cov["documented_symbols"] != float64(2) {
		t.Errorf("api_coverage documented_symbols = %v, want 2", cov["documented_symbols"])
	}
}

// TestRunDocscanStep_GracefulDegradation verifies that a doc_coverage fetch
// failure degrades to a null api_coverage with a warning, not a hard error.
func TestRunDocscanStep_GracefulDegradation(t *testing.T) {
	bin := buildDocscanTestFakeAnalyzer(t)
	moduleDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(moduleDir, "go.mod"), []byte("module fake\ngo 1.25\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	var stderr bytes.Buffer
	sess := adapter.NewSession(bin, []string{"--stdio", "--crash-after=doc_coverage"}, moduleDir, []string{"./..."}, &stderr, config.DefaultConfig())
	if _, err := sess.Initialize(); err != nil {
		t.Fatalf("Initialize: %v", err)
	}
	defer func() { _ = sess.Close() }()

	raw, err := RunDocscanStep(context.Background(), moduleDir, sess, &stderr)
	if err != nil {
		t.Fatalf("RunDocscanStep: %v", err)
	}

	var env map[string]json.RawMessage
	if err := json.Unmarshal(raw, &env); err != nil {
		t.Fatalf("unmarshal envelope: %v", err)
	}

	if string(env["api_coverage"]) != "null" {
		t.Errorf("api_coverage = %s, want null (graceful degradation)", string(env["api_coverage"]))
	}
	if !strings.Contains(stderr.String(), "skipping API coverage") {
		t.Errorf("stderr = %q, want it to contain %q", stderr.String(), "skipping API coverage")
	}
}
