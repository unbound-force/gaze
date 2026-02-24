package classify_test

import (
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// BenchmarkClassify_ContractsPackage benchmarks mechanical
// classification on the contracts fixture package (SC-004).
func BenchmarkClassify_ContractsPackage(b *testing.B) {
	allPkgs := loadTestPackagesB(b)
	contractsPkg := findPackage(allPkgs, "contracts")
	if contractsPkg == nil {
		b.Fatal("contracts package not found")
	}

	opts := analysis.Options{IncludeUnexported: false}
	results, err := analysis.Analyze(contractsPkg, opts)
	if err != nil {
		b.Fatalf("analysis failed: %v", err)
	}

	classifyOpts := classify.Options{
		Config:         config.DefaultConfig(),
		ModulePackages: allPkgs,
		TargetPkg:      contractsPkg,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Make a copy so classification doesn't accumulate.
		resultsCopy := make([]taxonomy.AnalysisResult, len(results))
		copy(resultsCopy, results)
		classify.Classify(resultsCopy, classifyOpts)
	}
}

// loadTestPackagesB is the benchmark variant of loadTestPackages.
func loadTestPackagesB(b *testing.B) []*packages.Package {
	b.Helper()
	dir := testdataDir()

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Dir:   dir,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		b.Fatalf("loading test packages: %v", err)
	}

	var valid []*packages.Package
	for _, pkg := range pkgs {
		if len(pkg.Errors) == 0 {
			valid = append(valid, pkg)
		}
	}

	return valid
}
