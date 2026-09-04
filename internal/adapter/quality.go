package adapter

import (
	"context"
	"sort"

	"github.com/unbound-force/gaze/internal/protocol"
	"github.com/unbound-force/gaze/internal/quality"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// BuildQualityFromMappings converts external analyzer test_mapping
// data and side effect analysis results into quality reports suitable
// for the quality command output. This bypasses the Go-specific
// quality.Assess pipeline entirely — the external analyzer provides
// pre-computed assertion mappings via the test_mapping protocol
// method.
//
// Design decision D1: Build quality reports from test_mapping data
// directly, reusing quality.ComputeContractCoverage for the metric
// computation.
//
// AssertionDetectionConfidence is set to 0 for all reports because
// external analyzers provide per-mapping confidence but no aggregate
// detection confidence metric.
func BuildQualityFromMappings(
	mappings []protocol.AssertionMappingData,
	results []taxonomy.AnalysisResult,
) ([]taxonomy.QualityReport, *taxonomy.PackageSummary) {
	type funcKey struct {
		pkg      string
		function string
	}

	// Group side effects by target function.
	effectsByFunc := make(map[funcKey][]taxonomy.SideEffect)
	locationByFunc := make(map[funcKey]string)
	for _, r := range results {
		key := funcKey{pkg: r.Target.Package, function: r.Target.Function}
		effectsByFunc[key] = append(effectsByFunc[key], r.SideEffects...)
		if _, ok := locationByFunc[key]; !ok {
			locationByFunc[key] = r.Target.Location
		}
	}

	// Group mappings by test function.
	type testKey struct {
		testFunction string
		testFile     string
	}
	mappingsByTest := make(map[testKey][]protocol.AssertionMappingData)
	for _, m := range mappings {
		key := testKey{testFunction: m.TestFunction, testFile: m.TestFile}
		mappingsByTest[key] = append(mappingsByTest[key], m)
	}

	// Build one QualityReport per test function.
	var reports []taxonomy.QualityReport

	for tk, testMappings := range mappingsByTest {
		// Collect all distinct target functions referenced by this
		// test's mappings. A single test may assert effects across
		// multiple targets (e.g., integration tests).
		targetSet := make(map[funcKey]struct{})
		for _, m := range testMappings {
			targetSet[funcKey{pkg: m.TargetPackage, function: m.TargetFunction}] = struct{}{}
		}

		// Union side effects across all referenced targets so that
		// contract coverage considers the full effect surface the
		// test exercises, not just the first target.
		var effects []taxonomy.SideEffect
		seen := make(map[string]bool)
		for fk := range targetSet {
			for _, e := range effectsByFunc[fk] {
				if !seen[e.ID] {
					seen[e.ID] = true
					effects = append(effects, e)
				}
			}
		}

		// Use the first target as the primary for the report's
		// TargetFunction metadata (display purposes only).
		primaryTarget := funcKey{
			pkg:      testMappings[0].TargetPackage,
			function: testMappings[0].TargetFunction,
		}

		// Convert protocol mappings to taxonomy mappings for this
		// test function.
		var taxonomyMappings []taxonomy.AssertionMapping
		for _, m := range testMappings {
			fk := funcKey{pkg: m.TargetPackage, function: m.TargetFunction}
			taxonomyMappings = append(taxonomyMappings, taxonomy.AssertionMapping{
				AssertionLocation: m.AssertionLocation,
				AssertionType:     taxonomy.AssertionType(m.AssertionType),
				SideEffectID:      findSideEffectID(effectsByFunc[fk], m.SideEffectType),
				Confidence:        m.Confidence,
			})
		}

		// Compute contract coverage using the shared quality function.
		cc := quality.ComputeContractCoverage(effects, taxonomyMappings)

		// Compute over-specification: assertions on incidental effects.
		overSpec := computeOverSpecification(effects, taxonomyMappings)

		// Collect ambiguous effects.
		var ambiguousEffects []taxonomy.SideEffect
		for _, e := range effects {
			if e.Classification != nil && e.Classification.Label == taxonomy.Ambiguous {
				ambiguousEffects = append(ambiguousEffects, e)
			}
		}

		// Collect unmapped assertions (those with empty SideEffectID).
		var unmapped []taxonomy.AssertionMapping
		for _, m := range taxonomyMappings {
			if m.SideEffectID == "" {
				unmapped = append(unmapped, m)
			}
		}

		report := taxonomy.QualityReport{
			TestFunction: tk.testFunction,
			TestLocation: tk.testFile,
			TargetFunction: taxonomy.FunctionTarget{
				Package:  primaryTarget.pkg,
				Function: primaryTarget.function,
				Location: locationByFunc[primaryTarget],
			},
			ContractCoverage:             cc,
			OverSpecification:            overSpec,
			AmbiguousEffects:             ambiguousEffects,
			UnmappedAssertions:           unmapped,
			AssertionCount:               len(testMappings),
			AssertionDetectionConfidence: 0, // External analyzers don't provide aggregate detection confidence
		}
		reports = append(reports, report)
	}

	// Sort reports by test function name for deterministic output.
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].TestFunction < reports[j].TestFunction
	})

	summary := buildQualitySummary(reports)
	return reports, summary
}

// computeOverSpecification counts assertions that target incidental
// side effects. An assertion is "over-specified" when it verifies a
// side effect classified as incidental — these assertions make tests
// fragile against refactoring.
func computeOverSpecification(
	effects []taxonomy.SideEffect,
	mappings []taxonomy.AssertionMapping,
) taxonomy.OverSpecificationScore {
	// Build a set of side effect IDs classified as incidental.
	incidentalIDs := make(map[string]bool)
	for _, e := range effects {
		if e.Classification != nil && e.Classification.Label == taxonomy.Incidental {
			incidentalIDs[e.ID] = true
		}
	}

	var incidentalAssertions []taxonomy.AssertionMapping
	var suggestions []string

	for _, m := range mappings {
		if m.SideEffectID != "" && incidentalIDs[m.SideEffectID] {
			incidentalAssertions = append(incidentalAssertions, m)
			suggestions = append(suggestions, "consider removing assertion on incidental effect at "+m.AssertionLocation)
		}
	}

	var ratio float64
	if len(mappings) > 0 {
		ratio = float64(len(incidentalAssertions)) / float64(len(mappings))
	}

	return taxonomy.OverSpecificationScore{
		Count:                len(incidentalAssertions),
		Ratio:                ratio,
		IncidentalAssertions: incidentalAssertions,
		Suggestions:          suggestions,
	}
}

// buildQualitySummary aggregates individual quality reports into a
// PackageSummary.
func buildQualitySummary(reports []taxonomy.QualityReport) *taxonomy.PackageSummary {
	if len(reports) == 0 {
		return &taxonomy.PackageSummary{}
	}

	summary := &taxonomy.PackageSummary{
		TotalTests: len(reports),
	}

	var totalCoverage float64
	for _, r := range reports {
		totalCoverage += r.ContractCoverage.Percentage
		summary.TotalOverSpecifications += r.OverSpecification.Count
	}
	summary.AverageContractCoverage = totalCoverage / float64(len(reports))

	// Worst coverage tests: bottom 5 by contract coverage percentage.
	sorted := make([]taxonomy.QualityReport, len(reports))
	copy(sorted, reports)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ContractCoverage.Percentage < sorted[j].ContractCoverage.Percentage
	})
	limit := 5
	if len(sorted) < limit {
		limit = len(sorted)
	}
	summary.WorstCoverageTests = sorted[:limit]

	return summary
}

// FetchTestMappings calls the test_mapping protocol method on the
// given client and returns the assertion mapping data. This is a
// standalone function used by the quality CLI path (which doesn't
// go through ExternalContractCoverageProvider).
//
// Returns the error on failure. The caller is responsible for
// graceful degradation (e.g., producing a zero-coverage report).
func FetchTestMappings(
	client *protocol.Client,
	patterns []string,
	rootDir string,
) ([]protocol.AssertionMappingData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), protocol.AnalysisTimeout)
	defer cancel()

	result, err := callAndUnmarshal[protocol.TestMappingResult](
		ctx, client, protocol.MethodTestMapping,
		protocol.TestMappingParams{
			RootPath: rootDir,
			Patterns: patterns,
		},
	)
	if err != nil {
		return nil, err
	}
	return result.Mappings, nil
}
