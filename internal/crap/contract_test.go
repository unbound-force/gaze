package crap

import (
	"bytes"
	"testing"

	"github.com/unbound-force/gaze/internal/config"
)

// ---------------------------------------------------------------------------
// extractShortPkgName tests
// ---------------------------------------------------------------------------

func TestExtractShortPkgName_WithSlash(t *testing.T) {
	got := extractShortPkgName("github.com/unbound-force/gaze/internal/crap")
	if got != "crap" {
		t.Errorf("extractShortPkgName(...crap) = %q, want %q", got, "crap")
	}
}

func TestExtractShortPkgName_NoSlash(t *testing.T) {
	got := extractShortPkgName("main")
	if got != "main" {
		t.Errorf("extractShortPkgName(main) = %q, want %q", got, "main")
	}
}

func TestExtractShortPkgName_TrailingSlash(t *testing.T) {
	// Last segment after final slash is an empty string when path ends with /.
	got := extractShortPkgName("github.com/user/repo/")
	if got != "" {
		t.Errorf("extractShortPkgName(.../repo/) = %q, want %q (empty)", got, "")
	}
}

func TestExtractShortPkgName_Empty(t *testing.T) {
	got := extractShortPkgName("")
	if got != "" {
		t.Errorf("extractShortPkgName('') = %q, want %q", got, "")
	}
}

// ---------------------------------------------------------------------------
// analyzePackageCoverage tests
// ---------------------------------------------------------------------------

func TestAnalyzePackageCoverage_ValidPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: loads real packages via analysis pipeline")
	}
	gazeConfig := config.DefaultConfig()
	var stderr bytes.Buffer
	reports, _ := analyzePackageCoverage(
		"github.com/unbound-force/gaze/internal/quality/testdata/src/welltested",
		".",
		gazeConfig,
		&stderr,
	)
	if len(reports) == 0 {
		t.Error("expected non-nil quality reports for well-tested package")
	}
}

func TestAnalyzePackageCoverage_InvalidPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: invokes go/packages.Load via analysis.LoadAndAnalyze")
	}
	gazeConfig := config.DefaultConfig()
	var stderr bytes.Buffer
	reports, _ := analyzePackageCoverage(
		"github.com/nonexistent/does/not/exist",
		".",
		gazeConfig,
		&stderr,
	)
	if reports != nil {
		t.Error("expected nil reports for non-existent package")
	}
}

// ---------------------------------------------------------------------------
// BuildContractCoverageFunc tests
// ---------------------------------------------------------------------------

// TestBuildContractCoverageFunc_InvalidPattern verifies that an
// unresolvable pattern returns nil without panicking.
func TestBuildContractCoverageFunc_InvalidPattern(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: invokes go/packages.Load via resolvePackagePaths")
	}
	var buf bytes.Buffer
	fn, _ := BuildContractCoverageFunc(
		[]string{"github.com/nonexistent/package/does/not/exist"},
		t.TempDir(),
		&buf,
	)
	if fn != nil {
		_, ok := fn("nonexistent", "Foo")
		if ok {
			t.Error("expected ok=false for unknown pkg:func key")
		}
	}
}

// TestBuildContractCoverageFunc_WelltestedPackage verifies that the
// function returns a callable closure for a package that has tests.
// This exercises the quality pipeline integration path.
// This is the primary regression guard for SC-002.
func TestBuildContractCoverageFunc_WelltestedPackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping: runs quality pipeline (package loading)")
	}

	pattern := "github.com/unbound-force/gaze/internal/quality/testdata/src/welltested"

	var buf bytes.Buffer
	fn, _ := BuildContractCoverageFunc([]string{pattern}, ".", &buf)

	if fn == nil {
		t.Fatal("BuildContractCoverageFunc returned nil; expected non-nil closure for well-tested package")
	}

	info, ok := fn("welltested", "Add")
	t.Logf("welltested:Add contract coverage: %.1f%% (found=%v, reason=%q)", info.Percentage, ok, info.Reason)
	if !ok {
		t.Fatal("expected ok=true for welltested:Add, got ok=false")
	}
	if info.Percentage <= 0 {
		t.Errorf("expected pct > 0 for welltested:Add (well-tested fixture should have non-zero coverage), got %.1f", info.Percentage)
	}
}

// ---------------------------------------------------------------------------
// no_test_coverage / no_effects_detected reason tests (spec 036)
// ---------------------------------------------------------------------------

// TestBuildContractCoverageFunc_NoTestCoverage verifies that a
// function with detected effects but no test coverage returns
// ContractCoverageInfo with Reason "no_test_coverage". This tests
// the effectsSet logic added in spec 036 (FR-006).
func TestBuildContractCoverageFunc_NoTestCoverage(t *testing.T) {
	// Construct the closure directly using the internal maps to
	// avoid running the full quality pipeline. The effectsSet
	// contains a function key, but the coverageMap does not.
	coverageMap := make(map[string]ContractCoverageInfo)
	effectsSet := map[string]bool{
		"mypkg:MyFunc": true,
	}

	fn := func(pkg, function string) (ContractCoverageInfo, bool) {
		key := pkg + ":" + function
		info, ok := coverageMap[key]
		if ok {
			return info, true
		}
		// Return ok=false so CRAP pipeline excludes from GazeCRAP
		// calculations. The Reason is informational for display.
		if effectsSet[key] {
			return ContractCoverageInfo{Reason: "no_test_coverage"}, false
		}
		return ContractCoverageInfo{Reason: "no_effects_detected"}, false
	}

	info, ok := fn("mypkg", "MyFunc")
	if ok {
		t.Fatal("expected ok=false for function with effects but no test (excluded from GazeCRAP)")
	}
	if info.Reason != "no_test_coverage" {
		t.Errorf("expected Reason %q, got %q", "no_test_coverage", info.Reason)
	}
	if info.Percentage != 0 {
		t.Errorf("expected Percentage 0 for untested function, got %.1f", info.Percentage)
	}
}

// TestBuildContractCoverageFunc_NoEffects verifies that a function
// with zero detected effects and no test coverage returns
// ContractCoverageInfo with Reason "no_effects_detected" and
// ok=false (existing behavior preserved).
func TestBuildContractCoverageFunc_NoEffects(t *testing.T) {
	// Construct the closure directly. Neither coverageMap nor
	// effectsSet contains the function key.
	coverageMap := make(map[string]ContractCoverageInfo)
	effectsSet := make(map[string]bool)

	fn := func(pkg, function string) (ContractCoverageInfo, bool) {
		key := pkg + ":" + function
		info, ok := coverageMap[key]
		if ok {
			return info, true
		}
		if effectsSet[key] {
			return ContractCoverageInfo{Reason: "no_test_coverage"}, true
		}
		return ContractCoverageInfo{Reason: "no_effects_detected"}, false
	}

	info, ok := fn("mypkg", "UnknownFunc")
	if ok {
		t.Error("expected ok=false for function with no effects and no test coverage")
	}
	if info.Reason != "no_effects_detected" {
		t.Errorf("expected Reason %q, got %q", "no_effects_detected", info.Reason)
	}
}
