package quality_test

import (
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/quality"
	"github.com/unbound-force/gaze/internal/taxonomy"
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

// BenchmarkMapAssertions benchmarks MapAssertionsToEffects using the
// indirectmatch fixture, which exercises the two-pass matching
// strategy (resolveExprRoot, selector/index/builtin unwinding,
// and helper return tracing). This is the SC-005 benchmark.
func BenchmarkMapAssertions(b *testing.B) {
	pkg := loadBenchPkg(b, "indirectmatch")
	nonTestPkg, err := loadNonTestPackage("indirectmatch")
	if err != nil {
		b.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "bench"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		b.Fatalf("analysis failed: %v", err)
	}

	// Build result map for target lookup (keyed by qualified name).
	resultMap := make(map[string]*taxonomy.AnalysisResult)
	for i := range results {
		resultMap[results[i].Target.QualifiedName()] = &results[i]
	}

	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		b.Fatalf("BuildTestSSA failed: %v", err)
	}

	testFuncs := quality.FindTestFunctions(pkg)
	qualOpts := quality.DefaultOptions()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, tf := range testFuncs {
			ssaFunc := ssaPkg.Func(tf.Name)
			if ssaFunc == nil {
				continue
			}
			targets, _ := quality.InferTargets(ssaFunc, pkg, qualOpts)
			for _, target := range targets {
				result, ok := resultMap[target.FuncName]
				if !ok {
					continue
				}
				sites := quality.DetectAssertions(tf.Decl, pkg, 3)
				quality.MapAssertionsToEffects(
					ssaFunc, target.SSAFunc, sites, result.SideEffects, pkg,
				)
			}
		}
	}
}

// loadBenchPkg loads a test package for benchmarks, using the
// shared test cache for efficiency.
func loadBenchPkg(b *testing.B, name string) *packages.Package {
	b.Helper()
	if testFixtureCache == nil {
		b.Fatal("testFixtureCache is nil — TestMain was not called")
	}
	pkg, err := testFixtureCache.get(name)
	if err != nil {
		b.Fatalf("failed to load test package %q: %v", name, err)
	}
	return pkg
}
