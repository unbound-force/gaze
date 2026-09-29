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
	"golang.org/x/tools/go/ssa"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
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

	// BuildSSAFunc overrides the SSA builder used by Assess. When
	// nil, BuildTestSSA is used. This is intended for testing the
	// SSA degradation path — inject a function that returns an
	// error to simulate SSA build failure.
	BuildSSAFunc func(*packages.Package) (*ssa.Program, *ssa.Package, error)

	// AIMapperFunc is an optional callback for AI-assisted assertion
	// mapping. When non-nil, it is called as a final fallback for
	// assertions that all mechanical passes fail to map. The AI
	// evaluates whether the assertion semantically verifies a side
	// effect of the target function (e.g., calling store.Get after
	// store.Set to verify the mutation). Mappings from AI are
	// assigned confidence 50 (lower than all mechanical passes).
	AIMapperFunc AIMapperFunc
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
//
// If SSA construction fails (e.g., due to upstream x/tools bugs
// with certain generic types), Assess degrades gracefully instead
// of returning an error. Degraded reports contain test function
// enumeration and assertion detection confidence (AST-only), but
// contract coverage, over-specification, and assertion mapping are
// zero-valued because target inference requires SSA. The returned
// PackageSummary.SSADegraded is set to true so callers and
// consumers can distinguish partial results from full-fidelity ones.
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
		GazeVersion:     opts.Version,
		Language:        "go",
		LanguageVersion: runtime.Version(),
		Timestamp:       start,
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
	buildSSA := BuildTestSSA
	if opts.BuildSSAFunc != nil {
		buildSSA = opts.BuildSSAFunc
	}
	var ssaDegraded bool
	_, ssaPkg, err := buildSSA(testPkg)
	if err != nil {
		// SSA construction failed — degrade gracefully instead of
		// returning an error. Target inference and assertion mapping
		// require SSA, so degraded reports will have zero-valued
		// coverage and over-specification metrics.
		ssaDegraded = true
		if opts.Stderr != nil {
			_, _ = fmt.Fprintf(opts.Stderr,
				"warning: SSA construction failed for %s, quality results are partial: %v\n",
				testPkg.PkgPath, err)
		}
	}

	// Step 3: Build reports.
	var reports []taxonomy.QualityReport
	var skippedTests int
	var skippedTestNames []string

	if ssaDegraded {
		// Degraded path: SSA unavailable. Produce one report per
		// test function with AST-only data (assertion detection
		// confidence). Target, coverage, and mapping are zero-valued.
		for _, tf := range testFuncs {
			pairStart := time.Now()
			sites := DetectAssertions(tf.Decl, testPkg, opts.MaxHelperDepth)
			detectionConf := computeDetectionConfidence(sites)

			report := taxonomy.QualityReport{
				TestFunction:                 tf.Name,
				TestLocation:                 tf.Location,
				AssertionCount:               len(sites),
				AssertionDetectionConfidence: detectionConf,
				Metadata: taxonomy.Metadata{
					GazeVersion:     meta.GazeVersion,
					Language:        meta.Language,
					LanguageVersion: meta.LanguageVersion,
					Timestamp:       meta.Timestamp,
					Duration:        time.Since(pairStart),
				},
			}
			reports = append(reports, report)
		}
	} else {
		// Normal path: SSA available. Infer targets, map assertions,
		// and compute full-fidelity metrics.
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
				skippedTests++
				skippedTestNames = append(skippedTestNames, tf.Name)
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
				mappings, unmapped, discardedIDs := mapAssertionsToEffectsImpl(
					ssaFunc, target.SSAFunc, sites, result.SideEffects, testPkg,
					opts.Stderr, opts.AIMapperFunc,
				)

				// Compute metrics, including discarded return detection.
				coverage := ComputeContractCoverage(result.SideEffects, mappings)
				coverage.DiscardedReturns = collectDiscardedReturns(result.SideEffects, discardedIDs)
				for _, dr := range coverage.DiscardedReturns {
					coverage.DiscardedReturnHints = append(coverage.DiscardedReturnHints, hintForEffect(dr))
				}
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
					AssertionCount:               len(sites),
					AssertionDetectionConfidence: detectionConf,
					Metadata: taxonomy.Metadata{
						GazeVersion:     meta.GazeVersion,
						Language:        meta.Language,
						LanguageVersion: meta.LanguageVersion,
						Timestamp:       meta.Timestamp,
						Duration:        time.Since(pairStart),
					},
				}
				reports = append(reports, report)
			}
		}
	}

	summary := BuildPackageSummary(reports)
	summary.SSADegraded = ssaDegraded
	if ssaDegraded {
		summary.SSADegradedPackages = []string{testPkg.PkgPath}
	}
	summary.SkippedTests = skippedTests
	summary.SkippedTestNames = skippedTestNames
	return reports, summary, nil
}

// BuildPackageSummary aggregates QualityReports into a PackageSummary.
// It computes report-level statistics only (coverage, over-specification,
// detection confidence, worst tests). Skipped test data (SkippedTests,
// SkippedTestNames) and SSA degradation status must be set by the caller
// post-hoc, since skipped tests don't produce QualityReport entries.
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
