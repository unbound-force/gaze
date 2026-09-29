package goprovider_test

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/provider/goprovider"
)

// Compile-time interface satisfaction checks. These verify that all
// four Go provider adapters implement their respective interfaces.
// Behavioral correctness is verified by the E2E self-check (task 5.2)
// which exercises the full pipeline with Go providers.
var (
	_ crap.ComplexityProvider       = (*goprovider.GoComplexityProvider)(nil)
	_ crap.LineCoverageProvider     = (*goprovider.GoLineCoverageProvider)(nil)
	_ crap.SideEffectAnalyzer       = (*goprovider.GoSideEffectAnalyzer)(nil)
	_ crap.ContractCoverageProvider = (*goprovider.GoContractCoverageProvider)(nil)
)

// TestGoLineCoverageProvider_PreGeneratedProfile verifies that Coverage
// parses a pre-generated cover profile without spawning go test.
// Uses a profile referencing real source files in this module so that
// ParseCoverProfile can map coverage lines to functions.
func TestGoLineCoverageProvider_PreGeneratedProfile(t *testing.T) {
	// Write a profile referencing a real file in this module.
	// crap.go line 245 is Formula (a small, real function).
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "cover.out")
	profileData := "mode: set\n" +
		"github.com/unbound-force/gaze/v2/internal/crap/crap.go:245.55,247.2 1 1\n"
	if err := os.WriteFile(profilePath, []byte(profileData), 0644); err != nil {
		t.Fatalf("writing test profile: %v", err)
	}

	// rootDir must be the module root so ParseCoverProfile can find source.
	moduleDir := moduleRoot(t)
	provider := goprovider.NewLineCoverageProvider(io.Discard)
	results, err := provider.Coverage([]string{"./..."}, moduleDir, profilePath)
	if err != nil {
		t.Fatalf("Coverage returned error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected non-empty coverage results from valid profile")
	}

	// Verify at least one result has expected fields.
	for _, r := range results {
		if r.File == "" {
			t.Error("expected non-empty File in FuncCoverage result")
		}
		if r.Percentage < 0 || r.Percentage > 100 {
			t.Errorf("Coverage percentage %f out of [0, 100] range", r.Percentage)
		}
	}
}

// moduleRoot returns the gaze module root directory.
func moduleRoot(t *testing.T) string {
	t.Helper()
	// Walk up from the test file's directory until we find go.mod.
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root (go.mod)")
		}
		dir = parent
	}
}

// TestGoLineCoverageProvider_InvalidProfilePath verifies that Coverage
// returns an error for a non-existent profile path.
func TestGoLineCoverageProvider_InvalidProfilePath(t *testing.T) {
	provider := goprovider.NewLineCoverageProvider(io.Discard)
	_, err := provider.Coverage([]string{"./..."}, t.TempDir(), "/nonexistent/cover.out")
	if err == nil {
		t.Fatal("expected error for non-existent profile path, got nil")
	}
}

// TestGoLineCoverageProvider_DirectoryAsProfile verifies that Coverage
// returns an error when the profile path is a directory.
func TestGoLineCoverageProvider_DirectoryAsProfile(t *testing.T) {
	provider := goprovider.NewLineCoverageProvider(io.Discard)
	_, err := provider.Coverage([]string{"./..."}, t.TempDir(), t.TempDir())
	if err == nil {
		t.Fatal("expected error for directory as profile path, got nil")
	}
}

// TestCoverage_ShortWithCoverProfile verifies that Coverage with
// Short=true and a valid coverProfile path reads the profile directly
// without spawning a subprocess. The Short field does not affect the
// pre-generated profile path — it only matters when go test is invoked.
func TestCoverage_ShortWithCoverProfile(t *testing.T) {
	// Write a profile referencing a real file in this module.
	tmpDir := t.TempDir()
	profilePath := filepath.Join(tmpDir, "cover.out")
	profileData := "mode: set\n" +
		"github.com/unbound-force/gaze/v2/internal/crap/crap.go:245.55,247.2 1 1\n"
	if err := os.WriteFile(profilePath, []byte(profileData), 0644); err != nil {
		t.Fatalf("writing test profile: %v", err)
	}

	moduleDir := moduleRoot(t)
	provider := goprovider.NewLineCoverageProvider(io.Discard)
	provider.Short = true

	results, err := provider.Coverage([]string{"./..."}, moduleDir, profilePath)
	if err != nil {
		t.Fatalf("Coverage with Short=true returned error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected non-empty coverage results from valid profile with Short=true")
	}

	// Verify results have valid fields — same assertions as the
	// non-Short test to confirm Short doesn't alter profile parsing.
	for _, r := range results {
		if r.File == "" {
			t.Error("expected non-empty File in FuncCoverage result")
		}
		if r.Percentage < 0 || r.Percentage > 100 {
			t.Errorf("Coverage percentage %f out of [0, 100] range", r.Percentage)
		}
	}
}

// TestGoSideEffectAnalyzer_WellTestedFixture verifies that Analyze
// detects and classifies side effects using a real Go package fixture.
// This test loads a real Go package via go/packages — it runs in both
// -short and non-short modes because the welltested fixture is small
// (< 2s to load and analyze).
func TestGoSideEffectAnalyzer_WellTestedFixture(t *testing.T) {
	analyzer := goprovider.NewSideEffectAnalyzer(nil, nil, false)
	results, err := analyzer.Analyze("github.com/unbound-force/gaze/v2/internal/quality/testdata/src/welltested")
	if err != nil {
		t.Fatalf("Analyze returned error: %v", err)
	}

	if len(results) == 0 {
		t.Fatal("expected non-empty results from welltested fixture")
	}

	// Verify every side effect has a classification attached.
	for _, r := range results {
		for _, e := range r.SideEffects {
			if e.Classification == nil {
				t.Errorf("SideEffect %s on %s has nil Classification",
					e.Type, r.Target.Function)
			} else if e.Classification.Label == "" {
				t.Errorf("SideEffect %s on %s has empty Classification.Label",
					e.Type, r.Target.Function)
			}
		}
	}
}

// TestGoSideEffectAnalyzer_InvalidPackage verifies that Analyze returns
// an error for a non-existent package path.
func TestGoSideEffectAnalyzer_InvalidPackage(t *testing.T) {
	analyzer := goprovider.NewSideEffectAnalyzer(nil, nil, false)
	_, err := analyzer.Analyze("github.com/nonexistent/package/does/not/exist")
	if err == nil {
		t.Fatal("expected error for non-existent package, got nil")
	}
}
