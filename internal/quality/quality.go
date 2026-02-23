// Package quality computes test quality metrics by mapping test
// assertions to detected side effects, producing Contract Coverage
// and Over-Specification scores for Go test functions.
package quality

import (
	"fmt"
	"io"
	"runtime"
	"sort"
	"time"

	"golang.org/x/tools/go/packages"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// Options configures quality analysis.
type Options struct {
	// TargetFunc is an optional function name to restrict analysis
	// to tests that exercise this specific function. If empty, all
	// test-target pairs are analyzed via call graph inference.
	TargetFunc string

	// MaxHelperDepth is the maximum call depth for traversing test
	// helper functions when detecting assertions. Default: 3.
	MaxHelperDepth int

	// Verbose enables detailed output including signal breakdowns.
	Verbose bool

	// Version is the Gaze version string for metadata.
	Version string

	// Stderr receives warnings about skipped tests, pairing
	// ambiguity, and other non-fatal issues. If nil, warnings are
	// suppressed.
	Stderr io.Writer
}

// DefaultOptions returns options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		MaxHelperDepth: 3,
	}
}

// Assess computes test quality metrics for the given analysis results.
//
// It takes classified analysis results (from Spec 001 + 002), the
// loaded test package, and options. It returns a QualityReport for
// each test-target pair found, plus a PackageSummary with aggregate
// metrics.
func Assess(
	results []taxonomy.AnalysisResult,
	testPkg *packages.Package,
	opts Options,
) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
	start := time.Now()

	if testPkg == nil {
		return nil, nil, fmt.Errorf("test package is nil")
	}
	if opts.MaxHelperDepth <= 0 {
		opts.MaxHelperDepth = 3
	}

	// Build metadata for reports.
	meta := taxonomy.Metadata{
		GazeVersion: opts.Version,
		GoVersion:   runtime.Version(),
		Timestamp:   start,
	}

	// Build a lookup from qualified function name to analysis result.
	resultMap := make(map[string]*taxonomy.AnalysisResult)
	for i := range results {
		key := results[i].Target.QualifiedName()
		resultMap[key] = &results[i]
	}

	// Step 1: Find test functions in the test package.
	testFuncs := FindTestFunctions(testPkg)
	if len(testFuncs) == 0 {
		if opts.Stderr != nil {
			_, _ = fmt.Fprintln(opts.Stderr, "warning: no test functions found")
		}
		return nil, &taxonomy.PackageSummary{}, nil
	}

	// Step 2: Build SSA for the test package.
	ssaProg, ssaPkg, err := BuildTestSSA(testPkg)
	if err != nil {
		return nil, nil, fmt.Errorf("building test SSA: %w", err)
	}
	_ = ssaProg // reserved for future cross-package analysis

	// Step 3: For each test function, infer the target, detect
	// assertions, map them, and compute metrics.
	var reports []taxonomy.QualityReport

	for _, tf := range testFuncs {
		ssaFunc := ssaPkg.Func(tf.Name)
		if ssaFunc == nil {
			continue
		}

		// Infer the target function.
		targets, warnings := InferTargets(ssaFunc, testPkg, opts)
		for _, w := range warnings {
			if opts.Stderr != nil {
				_, _ = fmt.Fprintf(opts.Stderr, "warning: %s: %s\n", tf.Name, w)
			}
		}

		// If --target flag is set, filter to matching targets.
		if opts.TargetFunc != "" {
			filtered := make([]InferredTarget, 0)
			for _, t := range targets {
				if t.FuncName == opts.TargetFunc {
					filtered = append(filtered, t)
				}
			}
			targets = filtered
		}

		if len(targets) == 0 {
			if opts.Stderr != nil {
				_, _ = fmt.Fprintf(opts.Stderr,
					"warning: %s: no target function identified, skipping\n", tf.Name)
			}
			continue
		}

		// Compute quality report for each target.
		for _, target := range targets {
			pairStart := time.Now()
			result, ok := resultMap[target.FuncName]
			if !ok {
				// Target function was not in the analysis results.
				if opts.Stderr != nil {
					_, _ = fmt.Fprintf(opts.Stderr,
						"warning: %s: target %s not in analysis results, skipping\n",
						tf.Name, target.FuncName)
				}
				continue
			}

			// Detect assertions in the test function.
			sites := DetectAssertions(tf.Decl, testPkg, opts.MaxHelperDepth)

			// Map assertions to side effects via SSA data flow.
			mappings, unmapped, discardedIDs := MapAssertionsToEffects(
				ssaFunc, target.SSAFunc, sites, result.SideEffects,
			)

			// Compute metrics, including discarded return detection.
			coverage := ComputeContractCoverage(result.SideEffects, mappings)
			coverage.DiscardedReturns = collectDiscardedReturns(result.SideEffects, discardedIDs)
			overSpec := ComputeOverSpecification(result.SideEffects, mappings)
			ambiguous := collectAmbiguous(result.SideEffects)
			detectionConf := computeDetectionConfidence(sites)

			report := taxonomy.QualityReport{
				TestFunction:                 tf.Name,
				TestLocation:                 tf.Location,
				TargetFunction:               result.Target,
				ContractCoverage:             coverage,
				OverSpecification:            overSpec,
				AmbiguousEffects:             ambiguous,
				UnmappedAssertions:           unmapped,
				AssertionDetectionConfidence: detectionConf,
				Metadata: taxonomy.Metadata{
					GazeVersion: meta.GazeVersion,
					GoVersion:   meta.GoVersion,
					Timestamp:   meta.Timestamp,
					Duration:    time.Since(pairStart),
				},
			}
			reports = append(reports, report)
		}
	}

	summary := BuildPackageSummary(reports)
	return reports, summary, nil
}

// BuildPackageSummary aggregates QualityReports into a PackageSummary.
func BuildPackageSummary(reports []taxonomy.QualityReport) *taxonomy.PackageSummary {
	if len(reports) == 0 {
		return &taxonomy.PackageSummary{}
	}

	var totalCoverage float64
	totalOverSpec := 0
	totalDetectionConf := 0

	for _, r := range reports {
		totalCoverage += r.ContractCoverage.Percentage
		totalOverSpec += r.OverSpecification.Count
		totalDetectionConf += r.AssertionDetectionConfidence
	}

	n := float64(len(reports))

	// Worst coverage: sort ascending, take bottom 5.
	// Use SliceStable with a secondary key (TestFunction name) to
	// ensure deterministic ordering when coverage percentages are
	// equal (SC-004 determinism requirement).
	sorted := make([]taxonomy.QualityReport, len(reports))
	copy(sorted, reports)
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].ContractCoverage.Percentage != sorted[j].ContractCoverage.Percentage {
			return sorted[i].ContractCoverage.Percentage < sorted[j].ContractCoverage.Percentage
		}
		return sorted[i].TestFunction < sorted[j].TestFunction
	})
	worst := sorted
	if len(worst) > 5 {
		worst = worst[:5]
	}

	return &taxonomy.PackageSummary{
		TotalTests:                   len(reports),
		AverageContractCoverage:      totalCoverage / n,
		TotalOverSpecifications:      totalOverSpec,
		WorstCoverageTests:           worst,
		AssertionDetectionConfidence: int(float64(totalDetectionConf)/n + 0.5),
	}
}

// collectDiscardedReturns filters side effects to those whose IDs
// appear in the discarded set. These are return/error effects whose
// values were explicitly discarded (e.g., _ = target()), making
// them definitively unasserted.
func collectDiscardedReturns(effects []taxonomy.SideEffect, discardedIDs map[string]bool) []taxonomy.SideEffect {
	if len(discardedIDs) == 0 {
		return nil
	}
	var result []taxonomy.SideEffect
	for _, e := range effects {
		if discardedIDs[e.ID] {
			result = append(result, e)
		}
	}
	return result
}

// collectAmbiguous returns side effects with ambiguous classification.
func collectAmbiguous(effects []taxonomy.SideEffect) []taxonomy.SideEffect {
	var ambiguous []taxonomy.SideEffect
	for _, e := range effects {
		if e.Classification != nil && e.Classification.Label == taxonomy.Ambiguous {
			ambiguous = append(ambiguous, e)
		}
	}
	return ambiguous
}

// computeDetectionConfidence computes the assertion detection
// confidence as the ratio of recognized patterns to total sites.
func computeDetectionConfidence(sites []AssertionSite) int {
	if len(sites) == 0 {
		return 0 // no assertions detected — cannot be confident
	}
	recognized := 0
	for _, s := range sites {
		if s.Kind != AssertionKindUnknown {
			recognized++
		}
	}
	return recognized * 100 / len(sites)
}
