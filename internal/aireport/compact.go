package aireport

import (
	"encoding/json"
	"fmt"

	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/docscan"
	"github.com/unbound-force/gaze/internal/report"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// compactPayload mirrors ReportPayload but includes the summary with JSON
// tags (not json:"-") and uses compact representations for each section.
type compactPayload struct {
	Summary  compactSummary  `json:"summary"`
	CRAP     json.RawMessage `json:"crap"`
	Quality  json.RawMessage `json:"quality"`
	Classify json.RawMessage `json:"classify"`
	Docscan  json.RawMessage `json:"docscan"`
	Errors   PayloadErrors   `json:"errors"`
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

// compactQualityReport mirrors taxonomy.QualityReport but uses
// compactContractCoverage (ID arrays instead of full SideEffect objects)
// and AmbiguousEffectIDs instead of AmbiguousEffects.
type compactQualityReport struct {
	TestFunction                 string                          `json:"test_function"`
	TestLocation                 string                          `json:"test_location"`
	TargetFunction               taxonomy.FunctionTarget         `json:"target_function"`
	ContractCoverage             compactContractCoverage         `json:"contract_coverage"`
	OverSpecification            taxonomy.OverSpecificationScore `json:"over_specification"`
	AmbiguousEffectIDs           []string                        `json:"ambiguous_effect_ids"`
	UnmappedAssertions           []taxonomy.AssertionMapping     `json:"unmapped_assertions"`
	AssertionCount               int                             `json:"assertion_count"`
	AssertionDetectionConfidence int                             `json:"assertion_detection_confidence"`
	Metadata                     taxonomy.Metadata               `json:"metadata"`
}

// compactContractCoverage replaces Gaps and DiscardedReturns (full
// SideEffect slices) with GapIDs and DiscardedReturnIDs (string slices),
// preserving hints and scalar fields.
type compactContractCoverage struct {
	Percentage           float64  `json:"percentage"`
	CoveredCount         int      `json:"covered_count"`
	TotalContractual     int      `json:"total_contractual"`
	GapIDs               []string `json:"gap_ids"`
	GapHints             []string `json:"gap_hints,omitempty"`
	DiscardedReturnIDs   []string `json:"discarded_return_ids"`
	DiscardedReturnHints []string `json:"discarded_return_hints,omitempty"`
}

// compactClassifyResult mirrors report.JSONReport but uses
// compactAnalysisResult with compactClassification on side effects.
type compactClassifyResult struct {
	Version string                  `json:"version"`
	Results []compactAnalysisResult `json:"results"`
}

// compactAnalysisResult mirrors taxonomy.AnalysisResult but uses
// compactSideEffect with stripped classification signals.
type compactAnalysisResult struct {
	Target      taxonomy.FunctionTarget `json:"target"`
	SideEffects []compactSideEffect     `json:"side_effects"`
	Metadata    taxonomy.Metadata       `json:"metadata"`
}

// compactSideEffect mirrors taxonomy.SideEffect but uses
// compactClassification (no Signals).
type compactSideEffect struct {
	ID             string                  `json:"id"`
	Type           taxonomy.SideEffectType `json:"type"`
	Tier           taxonomy.Tier           `json:"tier"`
	Location       string                  `json:"location"`
	Description    string                  `json:"description"`
	Target         string                  `json:"target"`
	Classification *compactClassification  `json:"classification,omitempty"`
}

// compactClassification retains Label, Confidence, and Reasoning
// but omits Signals to reduce payload size.
type compactClassification struct {
	Label      taxonomy.ClassificationLabel `json:"label"`
	Confidence int                          `json:"confidence"`
	Reasoning  string                       `json:"reasoning,omitempty"`
}

// compactCRAPReport mirrors crap.Report but uses compactCRAPSummary
// which omits WorstCrap, WorstGazeCrap, and RecommendedActions.
type compactCRAPReport struct {
	Scores  []crap.Score       `json:"scores"`
	Summary compactCRAPSummary `json:"summary"`
}

// compactCRAPSummary mirrors crap.Summary but omits WorstCRAP,
// WorstGazeCRAP, and RecommendedActions to reduce payload size.
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
	SSADegradedPackages []string                 `json:"ssa_degraded_packages,omitempty"`
}

// compactQualityOutput mirrors the quality.qualityOutput structure
// but uses compactQualityReport and compactPackageSummary.
type compactQualityOutput struct {
	Reports []compactQualityReport `json:"quality_reports"`
	Summary *compactPackageSummary `json:"quality_summary"`
}

// compactPackageSummary mirrors taxonomy.PackageSummary but omits
// WorstCoverageTests to reduce payload size.
type compactPackageSummary struct {
	TotalTests                   int      `json:"total_tests"`
	AverageContractCoverage      float64  `json:"average_contract_coverage"`
	TotalOverSpecifications      int      `json:"total_over_specifications"`
	AssertionDetectionConfidence int      `json:"assertion_detection_confidence"`
	SSADegraded                  bool     `json:"ssa_degraded"`
	SSADegradedPackages          []string `json:"ssa_degraded_packages,omitempty"`
	SkippedTests                 int      `json:"skipped_tests"`
	SkippedTestNames             []string `json:"skipped_test_names,omitempty"`
}

// CompactForAI produces a reduced JSON representation of the payload
// for the AI adapter text path. It strips large fields (docscan content,
// classification signals, worst offender lists) and replaces full
// SideEffect objects with ID strings to fit within model context windows.
//
// The full json.Marshal output is unaffected — this method produces a
// separate compact encoding.
//
// Nil fields (step failures) pass through as JSON null. Empty arrays
// are preserved as [] (not null).
func (p *ReportPayload) CompactForAI() ([]byte, error) {
	cp := compactPayload{
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

	// CRAP: unmarshal, strip worst offender lists, re-marshal.
	if p.CRAP != nil {
		compactCRAP, err := compactCRAPField(p.CRAP)
		if err != nil {
			return nil, fmt.Errorf("compacting CRAP: %w", err)
		}
		cp.CRAP = compactCRAP
	}

	// Quality: unmarshal, replace gaps/discarded returns with IDs,
	// replace ambiguous effects with IDs, strip worst coverage tests.
	if p.Quality != nil {
		compactQuality, err := compactQualityField(p.Quality)
		if err != nil {
			return nil, fmt.Errorf("compacting quality: %w", err)
		}
		cp.Quality = compactQuality
	}

	// Classify: unmarshal, strip signals from classifications.
	if p.Classify != nil {
		compactClassify, err := compactClassifyField(p.Classify)
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

// compactCRAPField unmarshals a crap.Report, strips WorstCRAP,
// WorstGazeCRAP, and RecommendedActions, and re-marshals.
func compactCRAPField(raw json.RawMessage) (json.RawMessage, error) {
	var full crap.Report
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("unmarshalling CRAP report: %w", err)
	}

	compact := compactCRAPReport{
		Scores: full.Scores,
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

// compactQualityField unmarshals quality reports, replaces Gaps and
// DiscardedReturns with ID arrays, replaces AmbiguousEffects with ID
// arrays, strips WorstCoverageTests, and re-marshals.
func compactQualityField(raw json.RawMessage) (json.RawMessage, error) {
	var full qualityOutput
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("unmarshalling quality report: %w", err)
	}

	compactReports := make([]compactQualityReport, len(full.Reports))
	for i, r := range full.Reports {
		compactReports[i] = compactQualityReport{
			TestFunction:                 r.TestFunction,
			TestLocation:                 r.TestLocation,
			TargetFunction:               r.TargetFunction,
			ContractCoverage:             projectContractCoverage(r.ContractCoverage),
			OverSpecification:            r.OverSpecification,
			AmbiguousEffectIDs:           extractEffectIDs(r.AmbiguousEffects),
			UnmappedAssertions:           r.UnmappedAssertions,
			AssertionCount:               r.AssertionCount,
			AssertionDetectionConfidence: r.AssertionDetectionConfidence,
			Metadata:                     r.Metadata,
		}
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
		}
	}

	out := compactQualityOutput{
		Reports: compactReports,
		Summary: compactSummary,
	}
	return json.Marshal(out)
}

// projectContractCoverage converts a full ContractCoverage into a
// compact form with ID arrays instead of full SideEffect objects.
func projectContractCoverage(cc taxonomy.ContractCoverage) compactContractCoverage {
	return compactContractCoverage{
		Percentage:           cc.Percentage,
		CoveredCount:         cc.CoveredCount,
		TotalContractual:     cc.TotalContractual,
		GapIDs:               extractEffectIDs(cc.Gaps),
		GapHints:             cc.GapHints,
		DiscardedReturnIDs:   extractEffectIDs(cc.DiscardedReturns),
		DiscardedReturnHints: cc.DiscardedReturnHints,
	}
}

// extractEffectIDs extracts the ID field from a slice of SideEffect.
// Returns an empty non-nil slice when effects is empty, and nil when
// effects is nil, preserving the distinction in JSON output.
func extractEffectIDs(effects []taxonomy.SideEffect) []string {
	if effects == nil {
		return nil
	}
	ids := make([]string, len(effects))
	for i, e := range effects {
		ids[i] = e.ID
	}
	return ids
}

// compactClassifyField unmarshals a classify result, strips Signals
// from each side effect's classification, and re-marshals.
func compactClassifyField(raw json.RawMessage) (json.RawMessage, error) {
	var full report.JSONReport
	if err := json.Unmarshal(raw, &full); err != nil {
		return nil, fmt.Errorf("unmarshalling classify result: %w", err)
	}

	compact := compactClassifyResult{
		Version: full.Version,
		Results: make([]compactAnalysisResult, len(full.Results)),
	}

	for i, r := range full.Results {
		compactEffects := make([]compactSideEffect, len(r.SideEffects))
		for j, se := range r.SideEffects {
			compactEffects[j] = compactSideEffect{
				ID:          se.ID,
				Type:        se.Type,
				Tier:        se.Tier,
				Location:    se.Location,
				Description: se.Description,
				Target:      se.Target,
			}
			if se.Classification != nil {
				compactEffects[j].Classification = &compactClassification{
					Label:      se.Classification.Label,
					Confidence: se.Classification.Confidence,
					Reasoning:  se.Classification.Reasoning,
				}
			}
		}
		compact.Results[i] = compactAnalysisResult{
			Target:      r.Target,
			SideEffects: compactEffects,
			Metadata:    r.Metadata,
		}
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
