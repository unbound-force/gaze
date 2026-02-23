package quality_test

import (
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/jflowers/gaze/internal/analysis"
	"github.com/jflowers/gaze/internal/quality"
)

// BenchmarkAssess_SinglePair benchmarks quality assessment for
// a single test-target pair using the welltested fixture.
func BenchmarkAssess_SinglePair(b *testing.B) {
	pkg := loadBenchPkg(b, "welltested")
	nonTestPkg, err := loadNonTestPackage("welltested")
	if err != nil {
		b.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "bench"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		b.Fatalf("analysis failed: %v", err)
	}

	qualOpts := quality.Options{TargetFunc: "Add"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := quality.Assess(results, pkg, qualOpts)
		if err != nil {
			b.Fatalf("Assess failed: %v", err)
		}
	}
}

// BenchmarkAssess_Package benchmarks quality assessment for an
// entire package using the welltested fixture.
func BenchmarkAssess_Package(b *testing.B) {
	pkg := loadBenchPkg(b, "welltested")
	nonTestPkg, err := loadNonTestPackage("welltested")
	if err != nil {
		b.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "bench"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		b.Fatalf("analysis failed: %v", err)
	}

	qualOpts := quality.DefaultOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := quality.Assess(results, pkg, qualOpts)
		if err != nil {
			b.Fatalf("Assess failed: %v", err)
		}
	}
}

// BenchmarkDetectAssertions benchmarks assertion detection for
// a single test function.
func BenchmarkDetectAssertions(b *testing.B) {
	pkg := loadBenchPkg(b, "welltested")
	testFuncs := quality.FindTestFunctions(pkg)
	if len(testFuncs) == 0 {
		b.Fatal("no test functions found")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tf := range testFuncs {
			quality.DetectAssertions(tf.Decl, pkg, 3)
		}
	}
}

// loadBenchPkg loads a test package for benchmarks, using the
// shared test cache for efficiency.
func loadBenchPkg(b *testing.B, name string) *packages.Package {
	b.Helper()
	pkg, err := cachedTestPackage(name)
	if err != nil {
		b.Fatalf("failed to load test package %q: %v", name, err)
	}
	return pkg
}
