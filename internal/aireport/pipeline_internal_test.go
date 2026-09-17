package aireport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"testing"

	"github.com/unbound-force/gaze/internal/adapter"
	"github.com/unbound-force/gaze/internal/crap"
)

// fakeSteps returns a pipelineStepFuncs with all four steps returning
// synthetic success results. Individual steps can be overridden after.
func fakeSteps() pipelineStepFuncs {
	return pipelineStepFuncs{
		crapStep: func(_ []string, _ string, _ string, _ io.Writer, _ crap.ContractCoverageProvider, _ bool) (*crapStepResult, error) {
			return &crapStepResult{
				JSON:           json.RawMessage(`{"crap":"ok"}`),
				CRAPload:       intPtr(5),
				GazeCRAPload:   intPtr(3),
				TotalFunctions: 20,
			}, nil
		},
		qualityStep: func(_ []string, _ string, _ io.Writer, _ ...qualityPipelineDeps) (*qualityStepResult, error) {
			return &qualityStepResult{
				JSON:                json.RawMessage(`{"quality":"ok"}`),
				AvgContractCoverage: intPtr(85),
			}, nil
		},
		classifyStep: func(_ []string, _ string, _ io.Writer, _ ...qualityPipelineDeps) (*classifyStepResult, error) {
			return &classifyStepResult{
				JSON:        json.RawMessage(`{"classify":"ok"}`),
				Contractual: 10,
				Ambiguous:   3,
				Incidental:  1,
			}, nil
		},
		docscanStep: func(_ string, _ *adapter.Session, _ io.Writer) (json.RawMessage, error) {
			return json.RawMessage(`{"documents":[],"api_coverage":null}`), nil
		},
	}
}

func TestRunProductionPipeline_AllStepsSucceed(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// All sections should be populated.
	if payload.CRAP == nil {
		t.Error("expected non-nil CRAP")
	}
	if payload.Quality == nil {
		t.Error("expected non-nil Quality")
	}
	if payload.Classify == nil {
		t.Error("expected non-nil Classify")
	}
	if payload.Docscan == nil {
		t.Error("expected non-nil Docscan")
	}

	// No errors should be set.
	if payload.Errors.CRAP != nil {
		t.Errorf("expected nil CRAP error, got: %v", *payload.Errors.CRAP)
	}
	if payload.Errors.Quality != nil {
		t.Errorf("expected nil Quality error, got: %v", *payload.Errors.Quality)
	}
	if payload.Errors.Classify != nil {
		t.Errorf("expected nil Classify error, got: %v", *payload.Errors.Classify)
	}
	if payload.Errors.Docscan != nil {
		t.Errorf("expected nil Docscan error, got: %v", *payload.Errors.Docscan)
	}

	// SSA should not be degraded when all steps succeed.
	if payload.Summary.SSADegraded {
		t.Error("expected SSADegraded=false when all steps succeed")
	}
	if len(payload.Summary.SSADegradedPackages) != 0 {
		t.Errorf("expected empty SSADegradedPackages, got %v", payload.Summary.SSADegradedPackages)
	}

	// Classification counts should be propagated from classify step.
	if payload.Summary.Contractual != 10 {
		t.Errorf("expected Contractual=10, got %d", payload.Summary.Contractual)
	}
	if payload.Summary.Ambiguous != 3 {
		t.Errorf("expected Ambiguous=3, got %d", payload.Summary.Ambiguous)
	}
	if payload.Summary.Incidental != 1 {
		t.Errorf("expected Incidental=1, got %d", payload.Summary.Incidental)
	}
}

func TestRunProductionPipeline_CRAPStepSSADegradation(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ crap.ContractCoverageProvider, _ bool) (*crapStepResult, error) {
		return &crapStepResult{
			JSON:                json.RawMessage(`{"crap":"ok"}`),
			CRAPload:            intPtr(5),
			GazeCRAPload:        intPtr(3),
			TotalFunctions:      20,
			SSADegradedPackages: []string{"pkg/degraded-from-crap"},
		}, nil
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	// SSA degradation from CRAP step should propagate to summary.
	if !payload.Summary.SSADegraded {
		t.Error("expected SSADegraded=true when CRAP step reports degraded packages")
	}
	found := false
	for _, pkg := range payload.Summary.SSADegradedPackages {
		if pkg == "pkg/degraded-from-crap" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected SSADegradedPackages to contain 'pkg/degraded-from-crap', got %v",
			payload.Summary.SSADegradedPackages)
	}
}

func TestRunProductionPipeline_CRAPStepFails(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ crap.ContractCoverageProvider, _ bool) (*crapStepResult, error) {
		return nil, fmt.Errorf("crap analysis failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	// CRAP error captured.
	if payload.Errors.CRAP == nil {
		t.Fatal("expected non-nil CRAP error")
	}
	if payload.CRAP != nil {
		t.Error("expected nil CRAP payload when step failed")
	}

	// Other sections still populated.
	if payload.Quality == nil {
		t.Error("expected non-nil Quality despite CRAP failure")
	}
	if payload.Classify == nil {
		t.Error("expected non-nil Classify despite CRAP failure")
	}
	if payload.Docscan == nil {
		t.Error("expected non-nil Docscan despite CRAP failure")
	}
}

func TestRunProductionPipeline_QualityStepFails(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.qualityStep = func(_ []string, _ string, _ io.Writer, _ ...qualityPipelineDeps) (*qualityStepResult, error) {
		return nil, fmt.Errorf("quality analysis failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	if payload.Errors.Quality == nil {
		t.Fatal("expected non-nil Quality error")
	}
	if payload.Quality != nil {
		t.Error("expected nil Quality payload when step failed")
	}
	if payload.CRAP == nil {
		t.Error("expected non-nil CRAP despite Quality failure")
	}
}

func TestRunProductionPipeline_ClassifyStepFails(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.classifyStep = func(_ []string, _ string, _ io.Writer, _ ...qualityPipelineDeps) (*classifyStepResult, error) {
		return nil, fmt.Errorf("classify failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	if payload.Errors.Classify == nil {
		t.Fatal("expected non-nil Classify error")
	}
	if payload.Classify != nil {
		t.Error("expected nil Classify payload when step failed")
	}
	if payload.CRAP == nil {
		t.Error("expected non-nil CRAP despite Classify failure")
	}
	if payload.Docscan == nil {
		t.Error("expected non-nil Docscan despite Classify failure")
	}
}

func TestRunProductionPipeline_DocscanStepFails(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.docscanStep = func(_ string, _ *adapter.Session, _ io.Writer) (json.RawMessage, error) {
		return nil, fmt.Errorf("docscan failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	if payload.Errors.Docscan == nil {
		t.Fatal("expected non-nil Docscan error")
	}
	if payload.Docscan != nil {
		t.Error("expected nil Docscan payload when step failed")
	}
	if payload.CRAP == nil {
		t.Error("expected non-nil CRAP despite Docscan failure")
	}
	if payload.Quality == nil {
		t.Error("expected non-nil Quality despite Docscan failure")
	}
}

func TestRunProductionPipeline_MultipleStepsFail(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ crap.ContractCoverageProvider, _ bool) (*crapStepResult, error) {
		return nil, fmt.Errorf("crap failed")
	}
	steps.qualityStep = func(_ []string, _ string, _ io.Writer, _ ...qualityPipelineDeps) (*qualityStepResult, error) {
		return nil, fmt.Errorf("quality failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failures, got: %v", err)
	}

	// Both errors captured.
	if payload.Errors.CRAP == nil {
		t.Error("expected non-nil CRAP error")
	}
	if payload.Errors.Quality == nil {
		t.Error("expected non-nil Quality error")
	}

	// Other sections still populated.
	if payload.Classify == nil {
		t.Error("expected non-nil Classify despite CRAP+Quality failures")
	}
	if payload.Docscan == nil {
		t.Error("expected non-nil Docscan despite CRAP+Quality failures")
	}
}

func TestRunProductionPipeline_EmptyPatterns(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	// Track whether any step was called.
	called := false
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ crap.ContractCoverageProvider, _ bool) (*crapStepResult, error) {
		called = true
		return nil, nil
	}

	_, err := runProductionPipeline([]string{}, "/tmp", "", false, &stderr, steps, nil)
	if err == nil {
		t.Fatal("expected error for empty patterns")
	}
	if called {
		t.Error("step functions should not be called when patterns are empty")
	}
}

// TestRunProductionPipeline_GazeCRAPloadFlowsThroughPipeline verifies that
// when the crapStep returns a non-zero GazeCRAPload, it propagates to
// payload.Summary.GazeCRAPload. Callback wiring (non-nil callback passed
// to crapStep) is verified by the integration test
// TestSC002_GazeCRAPloadMatchBetweenCrapAndReport in cmd/gaze/main_test.go.
func TestRunProductionPipeline_GazeCRAPloadFlowsThroughPipeline(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	// Override crapStep to return a known GazeCRAPload value.
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ crap.ContractCoverageProvider, _ bool) (*crapStepResult, error) {
		return &crapStepResult{
			JSON:           json.RawMessage(`{"crap":"ok"}`),
			CRAPload:       intPtr(2),
			GazeCRAPload:   intPtr(7),
			TotalFunctions: 15,
		}, nil
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if payload.Summary.GazeCRAPload == nil || *payload.Summary.GazeCRAPload != 7 {
		t.Errorf("expected GazeCRAPload 7, got %v", payload.Summary.GazeCRAPload)
	}
}

func TestRunProductionPipeline_SummaryFields(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}

	if payload.Summary.CRAPload == nil || *payload.Summary.CRAPload != 5 {
		t.Errorf("expected CRAPload 5, got %v", payload.Summary.CRAPload)
	}
	if payload.Summary.GazeCRAPload == nil || *payload.Summary.GazeCRAPload != 3 {
		t.Errorf("expected GazeCRAPload 3, got %v", payload.Summary.GazeCRAPload)
	}
	if payload.Summary.AvgContractCoverage == nil || *payload.Summary.AvgContractCoverage != 85 {
		t.Errorf("expected AvgContractCoverage 85, got %v", payload.Summary.AvgContractCoverage)
	}
}

// TestRunProductionPipeline_CRAPStepFails_CRAPloadIsNil verifies that when
// the CRAP step fails, payload.Summary.CRAPload is nil (not zero). This is
// the pipeline-level regression test for #102: the *int type ensures that
// a failed step produces nil (unavailable) rather than the Go int zero value 0,
// which would silently pass threshold checks.
func TestRunProductionPipeline_CRAPStepFails_CRAPloadIsNil(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ crap.ContractCoverageProvider, _ bool) (*crapStepResult, error) {
		return nil, fmt.Errorf("crap analysis failed")
	}

	payload, err := runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("pipeline should not return error on step failure, got: %v", err)
	}

	// CRAPload must be nil (unavailable), not zero.
	if payload.Summary.CRAPload != nil {
		t.Errorf("expected CRAPload=nil when CRAP step failed, got %d", *payload.Summary.CRAPload)
	}
	// GazeCRAPload must also be nil (same step).
	if payload.Summary.GazeCRAPload != nil {
		t.Errorf("expected GazeCRAPload=nil when CRAP step failed, got %d", *payload.Summary.GazeCRAPload)
	}
	// AvgContractCoverage should be populated (quality step succeeded).
	if payload.Summary.AvgContractCoverage == nil {
		t.Error("expected AvgContractCoverage to be populated (quality step succeeded)")
	}
}

// TestRunProductionPipeline_TestShortThreadsToStep verifies that the
// testShort parameter flows through runProductionPipeline to the crapStep
// function. This ensures --test-short on the report command reaches the
// line coverage provider.
func TestRunProductionPipeline_TestShortThreadsToStep(t *testing.T) {
	var stderr bytes.Buffer
	steps := fakeSteps()

	// Capture the short parameter passed to crapStep.
	var capturedShort bool
	steps.crapStep = func(_ []string, _ string, _ string, _ io.Writer, _ crap.ContractCoverageProvider, short bool) (*crapStepResult, error) {
		capturedShort = short
		return &crapStepResult{
			JSON:           json.RawMessage(`{"crap":"ok"}`),
			CRAPload:       intPtr(1),
			GazeCRAPload:   intPtr(0),
			TotalFunctions: 5,
		}, nil
	}

	// Pass testShort=true and verify it reaches the step.
	_, err := runProductionPipeline([]string{"./..."}, "/tmp", "", true, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if !capturedShort {
		t.Error("expected testShort=true to be passed to crapStep, got false")
	}

	// Pass testShort=false and verify it reaches the step.
	capturedShort = true // reset to non-default
	_, err = runProductionPipeline([]string{"./..."}, "/tmp", "", false, &stderr, steps, nil)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if capturedShort {
		t.Error("expected testShort=false to be passed to crapStep, got true")
	}
}

// TestResolveModulePackages_EmptyDir verifies that resolveModulePackages
// with an empty moduleDir uses the current working directory and
// successfully loads module packages (when run from a Go module root).
func TestResolveModulePackages_EmptyDir(t *testing.T) {
	pkgs := resolveModulePackages("")
	// When run from the gaze module root, this should return packages.
	// In CI, the working directory is the repo root.
	if pkgs == nil {
		t.Skip("not running from a Go module root — skipping")
	}
	if len(pkgs) == 0 {
		t.Error("expected non-empty package list from module root")
	}
}

// TestResolveModulePackages_InvalidDir verifies that resolveModulePackages
// returns nil for a directory that is not a Go module.
func TestResolveModulePackages_InvalidDir(t *testing.T) {
	pkgs := resolveModulePackages(t.TempDir())
	if pkgs != nil {
		t.Errorf("expected nil for non-module directory, got %d packages", len(pkgs))
	}
}
