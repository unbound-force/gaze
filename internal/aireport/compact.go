package aireport

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/docscan"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// compactPayload mirrors ReportPayload but includes the summary with JSON
// tags (not json:"-") and uses compact representations for each section.
type compactPayload struct {
	Metadata   ReportMetadata  `json:"metadata"`
	Summary    compactSummary  `json:"summary"`
	CRAP       json.RawMessage `json:"crap"`
	Quality    json.RawMessage `json:"quality"`
	Classify   json.RawMessage `json:"classify"`
	Docscan    json.RawMessage `json:"docscan"`
	Errors     PayloadErrors   `json:"errors"`
	Assessment *Assessment     `json:"assessment"`
}

// compactSummary mirrors ReportSummary with JSON tags so it appears in
// the compact payload output.
type compactSummary struct {
	CRAPload            *int     `json:"crapload,omitempty"`
	GazeCRAPload        *int     `json:"gaze_crapload,omitempty"`
	AvgContractCoverage *int     `json:"avg_contract_coverage,omitempty"`
	SSADegraded         bool     `json:"ssa_degraded"`
	SSADegradedPackages []string `json:"ssa_degraded_packages"`
	Contractual         int      `json:"contractual"`
	Ambiguous           int      `json:"ambiguous"`
	Incidental          int      `json:"incidental"`
	SkippedTests        int      `json:"skipped_tests"`
}

// compactDocscanEntry retains only the path and priority from a
// docscan.DocumentFile, stripping the Content field which dominates
// payload size.
type compactDocscanEntry struct {
	Path     string           `json:"path"`
	Priority docscan.Priority `json:"priority"`
}

// compactQualityReport mirrors taxonomy.QualityReport but projects
// SideEffect slices to minimal self-contained objects (no classification).
type compactQualityReport struct {
	TestFunction                 string                          `json:"test_function"`
	TestLocation                 string                          `json:"test_location"`
	TargetFunction               taxonomy.FunctionTarget         `json:"target_function"`
	ContractCoverage             compactContractCoverage         `json:"contract_coverage"`
	OverSpecification            taxonomy.OverSpecificationScore `json:"over_specification"`
	AmbiguousEffects             []compactSideEffect             `json:"ambiguous_effects"`
	UnmappedAssertions           []taxonomy.AssertionMapping     `json:"unmapped_assertions"`
	AssertionCount               int                             `json:"assertion_count"`
	AssertionDetectionConfidence int                             `json:"assertion_detection_confidence"`
	Metadata                     taxonomy.Metadata               `json:"metadata"`
}

// compactContractCoverage projects Gaps and DiscardedReturns (full
// SideEffect slices) to minimal self-contained side-effect objects,
// preserving hints and scalar fields.
type compactContractCoverage struct {
	Percentage           float64             `json:"percentage"`
	CoveredCount         int                 `json:"covered_count"`
	TotalContractual     int                 `json:"total_contractual"`
	Gaps                 []compactSideEffect `json:"gaps"`
	GapHints             []string            `json:"gap_hints,omitempty"`
	DiscardedReturns     []compactSideEffect `json:"discarded_returns"`
	DiscardedReturnHints []string            `json:"discarded_return_hints,omitempty"`
}

// compactSideEffect is a minimal projection of taxonomy.SideEffect that
// carries id, type, tier, location, description, and target — enough for
// the reporter to render gaps and ambiguous effects without the Classify
// output — and omits the classification object entirely.
type compactSideEffect struct {
	ID          string                  `json:"id"`
	Type        taxonomy.SideEffectType `json:"type"`
	Tier        taxonomy.Tier           `json:"tier"`
	Location    string                  `json:"location"`
	Description string                  `json:"description"`
	Target      string                  `json:"target"`
}

// compactCRAPReport mirrors crap.Report but omits the full Scores array,
// keeping only the summary (including the bounded worst-offender lists).
type compactCRAPReport struct {
	Summary compactCRAPSummary `json:"summary"`
}

// compactCRAPSummary mirrors crap.Summary, preserving the bounded
// worst-offender lists (WorstCRAP, WorstGazeCRAP, RecommendedActions)
// while the unbounded Scores array is dropped by the parent struct.
type compactCRAPSummary struct {
	TotalFunctions      int                      `json:"total_functions"`
	AvgComplexity       float64                  `json:"avg_complexity"`
	AvgLineCoverage     float64                  `json:"avg_line_coverage"`
	AvgCRAP             float64                  `json:"avg_crap"`
	CRAPload            int                      `json:"crapload"`
	CRAPThreshold       float64                  `json:"crap_threshold"`
	GazeCRAPload        *int                     `json:"gaze_crapload,omitempty"`
	GazeCRAPThreshold   *float64                 `json:"gaze_crap_threshold,omitempty"`
	AvgGazeCRAP         *float64                 `json:"avg_gaze_crap,omitempty"`
	AvgContractCoverage *float64                 `json:"avg_contract_coverage,omitempty"`
	QuadrantCounts      map[crap.Quadrant]int    `json:"quadrant_counts,omitempty"`
	FixStrategyCounts   map[crap.FixStrategy]int `json:"fix_strategy_counts,omitempty"`
	WorstCRAP           []crap.Score             `json:"worst_crap"`
	WorstGazeCRAP       []crap.Score             `json:"worst_gaze_crap,omitempty"`
	RecommendedActions  []crap.RecommendedAction `json:"recommended_actions,omitempty"`
	SSADegradedPackages []string                 `json:"ssa_degraded_packages,omitempty"`
}

// compactQualityOutput mirrors the quality.qualityOutput structure
// but uses compactQualityReport and compactPackageSummary.
type compactQualityOutput struct {
	Reports []compactQualityReport `json:"quality_reports"`
	Summary *compactPackageSummary `json:"quality_summary"`
}

// compactPackageSummary mirrors taxonomy.PackageSummary, preserving the
// bounded WorstCoverageTests list (bottom 5 by coverage).
type compactPackageSummary struct {
	TotalTests                   int                    `json:"total_tests"`
	AverageContractCoverage      float64                `json:"average_contract_coverage"`
	TotalOverSpecifications      int                    `json:"total_over_specifications"`
	AssertionDetectionConfidence int                    `json:"assertion_detection_confidence"`
	SSADegraded                  bool                   `json:"ssa_degraded"`
	SSADegradedPackages          []string               `json:"ssa_degraded_packages,omitempty"`
	SkippedTests                 int                    `json:"skipped_tests"`
	SkippedTestNames             []string               `json:"skipped_test_names,omitempty"`
	WorstCoverageTests           []compactQualityReport `json:"worst_coverage_tests,omitempty"`
}

// qualityReportCap bounds the number of actionable quality reports
// included in the compact payload. Combined with the dropped unbounded
// arrays (crap.scores, classify.results) it keeps the payload within the
// model context window regardless of total codebase size.
const qualityReportCap = 50

// CompactForAI produces a reduced JSON representation of the payload
// for the AI adapter text path. It drops the unbounded full arrays
// (crap scores, classify results), bounds the quality reports to the most
// actionable entries, and preserves the bounded worst-offender lists the
// reporter actually reads. It also strips large docscan content.
//
// The full json.Marshal output is unaffected — this method produces a
// separate compact encoding.
//
// Nil fields (step failures) pass through as JSON null. Empty arrays
// are preserved as [] (not null).
func (p *ReportPayload) CompactForAI() ([]byte, error) {
	cp := compactPayload{
		Metadata:   p.Metadata,
		Assessment: p.Assessment,
		Summary: compactSummary{
			CRAPload:            p.Summary.CRAPload,
			GazeCRAPload:        p.Summary.GazeCRAPload,
			AvgContractCoverage: p.Summary.AvgContractCoverage,
			SSADegraded:         p.Summary.SSADegraded,
			SSADegradedPackages: p.Summary.SSADegradedPackages,
			Contractual:         p.Summary.Contractual,
			Ambiguous:           p.Summary.Ambiguous,
			Incidental:          p.Summary.Incidental,
			SkippedTests:        p.Summary.SkippedTests,
		},
		Errors: p.Errors,
	}

	// CRAP: unmarshal, drop the full Scores array, keep the summary with
	// worst offender lists.
	if p.CRAP != nil {
		compactCRAP, err := compactCRAPField(p.CRAP)
		if err != nil {
			return nil, fmt.Errorf("compacting CRAP: %w", err)
		}
		cp.CRAP = compactCRAP
	}

	// Quality: unmarshal, project gaps/discarded/ambiguous effects to
	// self-contained objects, bound reports to actionable entries, and
	// keep the worst coverage tests.
	if p.Quality != nil {
		compactQuality, err := compactQualityField(p.Quality)
		if err != nil {
			return nil, fmt.Errorf("compacting quality: %w", err)
		}
		cp.Quality = compactQuality
	}

	// Classify: emit counts only (sourced from the top-level summary),
	// dropping the full results array.
	if p.Classify != nil {
		compactClassify, err := compactClassifyField(p.Classify, p.Summary.Contractual, p.Summary.Ambiguous, p.Summary.Incidental)
		if err != nil {
			return nil, fmt.Errorf("compacting classify: %w", err)
		}
		cp.Classify = compactClassify
	}

	// Docscan: unmarshal, strip content.
	if p.Docscan != nil {
		compactDocscan, err := compactDocscanField(p.Docscan)
		if err != nil {
			return nil, fmt.Errorf("compacting docscan: %w", err)
		}
		cp.Docscan = compactDocscan
	}

	return json.Marshal(cp)
}

// compactCRAPField unmarshals a crap.Report, drops the full Scores array,
// and re-marshals the summary (preserving worst offender lists).
func compactCRAPField(raw json.RawMessage) (json.RawMessage, error) {
	var full crap.Report
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("unmarshalling CRAP report: %w", err)
	}

	compact := compactCRAPReport{
		Summary: compactCRAPSummary{
			TotalFunctions:      full.Summary.TotalFunctions,
			AvgComplexity:       full.Summary.AvgComplexity,
			AvgLineCoverage:     full.Summary.AvgLineCoverage,
			AvgCRAP:             full.Summary.AvgCRAP,
			CRAPload:            full.Summary.CRAPload,
			CRAPThreshold:       full.Summary.CRAPThreshold,
			GazeCRAPload:        full.Summary.GazeCRAPload,
			GazeCRAPThreshold:   full.Summary.GazeCRAPThreshold,
			AvgGazeCRAP:         full.Summary.AvgGazeCRAP,
			AvgContractCoverage: full.Summary.AvgContractCoverage,
			QuadrantCounts:      full.Summary.QuadrantCounts,
			FixStrategyCounts:   full.Summary.FixStrategyCounts,
			WorstCRAP:           full.Summary.WorstCRAP,
			WorstGazeCRAP:       full.Summary.WorstGazeCRAP,
			RecommendedActions:  full.Summary.RecommendedActions,
			SSADegradedPackages: full.Summary.SSADegradedPackages,
		},
	}

	return json.Marshal(compact)
}

// qualityOutput mirrors the quality package's top-level JSON structure.
// Defined here to avoid importing the unexported type.
type qualityOutput struct {
	Reports []taxonomy.QualityReport `json:"quality_reports"`
	Summary *taxonomy.PackageSummary `json:"quality_summary"`
}

// compactQualityField unmarshals quality reports, projects gaps,
// discarded returns, and ambiguous effects to self-contained objects,
// bounds reports to actionable entries ordered by ascending contract
// coverage, preserves worst coverage tests, and re-marshals.
func compactQualityField(raw json.RawMessage) (json.RawMessage, error) {
	var full qualityOutput
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("unmarshalling quality report: %w", err)
	}

	compactReports := make([]compactQualityReport, 0, len(full.Reports))
	for _, r := range full.Reports {
		cr := projectQualityReport(r)
		if !isActionable(cr) {
			continue
		}
		compactReports = append(compactReports, cr)
	}

	sort.Slice(compactReports, func(i, j int) bool {
		return compactReports[i].ContractCoverage.Percentage < compactReports[j].ContractCoverage.Percentage
	})

	if len(compactReports) > qualityReportCap {
		compactReports = compactReports[:qualityReportCap]
	}

	var compactSummary *compactPackageSummary
	if full.Summary != nil {
		compactSummary = &compactPackageSummary{
			TotalTests:                   full.Summary.TotalTests,
			AverageContractCoverage:      full.Summary.AverageContractCoverage,
			TotalOverSpecifications:      full.Summary.TotalOverSpecifications,
			AssertionDetectionConfidence: full.Summary.AssertionDetectionConfidence,
			SSADegraded:                  full.Summary.SSADegraded,
			SSADegradedPackages:          full.Summary.SSADegradedPackages,
			SkippedTests:                 full.Summary.SkippedTests,
			SkippedTestNames:             full.Summary.SkippedTestNames,
			WorstCoverageTests:           projectQualityReports(full.Summary.WorstCoverageTests),
		}
	}

	out := compactQualityOutput{
		Reports: compactReports,
		Summary: compactSummary,
	}
	return json.Marshal(out)
}

// projectQualityReport converts a full taxonomy.QualityReport into a
// compact form, projecting side-effect slices to self-contained objects.
func projectQualityReport(r taxonomy.QualityReport) compactQualityReport {
	return compactQualityReport{
		TestFunction:                 r.TestFunction,
		TestLocation:                 r.TestLocation,
		TargetFunction:               r.TargetFunction,
		ContractCoverage:             projectContractCoverage(r.ContractCoverage),
		OverSpecification:            r.OverSpecification,
		AmbiguousEffects:             projectSideEffects(r.AmbiguousEffects),
		UnmappedAssertions:           r.UnmappedAssertions,
		AssertionCount:               r.AssertionCount,
		AssertionDetectionConfidence: r.AssertionDetectionConfidence,
		Metadata:                     r.Metadata,
	}
}

// projectQualityReports converts a slice of full taxonomy.QualityReport
// into compact form. Returns nil for nil input.
func projectQualityReports(reports []taxonomy.QualityReport) []compactQualityReport {
	if reports == nil {
		return nil
	}
	out := make([]compactQualityReport, len(reports))
	for i, r := range reports {
		out[i] = projectQualityReport(r)
	}
	return out
}

// isActionable reports whether a compact quality report carries anything
// the reporter can act on: coverage gaps, discarded returns, ambiguous
// effects, or unmapped assertions.
func isActionable(r compactQualityReport) bool {
	return len(r.ContractCoverage.Gaps) > 0 ||
		len(r.ContractCoverage.DiscardedReturns) > 0 ||
		len(r.AmbiguousEffects) > 0 ||
		len(r.UnmappedAssertions) > 0
}

// projectContractCoverage converts a full ContractCoverage into a
// compact form with self-contained side-effect objects instead of full
// SideEffect objects.
func projectContractCoverage(cc taxonomy.ContractCoverage) compactContractCoverage {
	return compactContractCoverage{
		Percentage:           cc.Percentage,
		CoveredCount:         cc.CoveredCount,
		TotalContractual:     cc.TotalContractual,
		Gaps:                 projectSideEffects(cc.Gaps),
		GapHints:             cc.GapHints,
		DiscardedReturns:     projectSideEffects(cc.DiscardedReturns),
		DiscardedReturnHints: cc.DiscardedReturnHints,
	}
}

// projectSideEffects converts a slice of taxonomy.SideEffect into minimal
// self-contained compactSideEffect objects. Returns an empty non-nil
// slice when effects is empty, and nil when effects is nil, preserving
// the distinction in JSON output.
func projectSideEffects(effects []taxonomy.SideEffect) []compactSideEffect {
	if effects == nil {
		return nil
	}
	out := make([]compactSideEffect, len(effects))
	for i, e := range effects {
		out[i] = compactSideEffect{
			ID:          e.ID,
			Type:        e.Type,
			Tier:        e.Tier,
			Location:    e.Location,
			Description: e.Description,
			Target:      e.Target,
		}
	}
	return out
}

// compactClassifyField emits a counts-only classify object, sourcing the
// contractual/ambiguous/incidental counts from the top-level summary and
// preserving only the version string from the full classify result. The
// full results array is dropped to bound payload size.
func compactClassifyField(raw json.RawMessage, contractual, ambiguous, incidental int) (json.RawMessage, error) {
	var header struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(raw, &header); err != nil {
		return nil, fmt.Errorf("unmarshalling classify result: %w", err)
	}

	compact := struct {
		Version     string `json:"version"`
		Contractual int    `json:"contractual"`
		Ambiguous   int    `json:"ambiguous"`
		Incidental  int    `json:"incidental"`
	}{
		Version:     header.Version,
		Contractual: contractual,
		Ambiguous:   ambiguous,
		Incidental:  incidental,
	}

	return json.Marshal(compact)
}

// compactDocscanField unmarshals the docscan envelope and strips the
// Content field from documents, keeping only Path and Priority. The
// APICoverage field is passed through unchanged.
func compactDocscanField(raw json.RawMessage) (json.RawMessage, error) {
	var envelope struct {
		Documents   []docscan.DocumentFile `json:"documents"`
		APICoverage json.RawMessage        `json:"api_coverage"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("unmarshalling docscan: %w", err)
	}

	compact := make([]compactDocscanEntry, len(envelope.Documents))
	for i, d := range envelope.Documents {
		compact[i] = compactDocscanEntry{
			Path:     d.Path,
			Priority: d.Priority,
		}
	}

	// Build compact envelope preserving the api_coverage field.
	result := struct {
		Documents   []compactDocscanEntry `json:"documents"`
		APICoverage json.RawMessage       `json:"api_coverage,omitempty"`
	}{
		Documents:   compact,
		APICoverage: envelope.APICoverage,
	}

	return json.Marshal(result)
}
