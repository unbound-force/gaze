package adapter

import (
	"testing"

	"github.com/unbound-force/gaze/internal/protocol"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// ---------------------------------------------------------------------------
// BuildQualityFromMappings tests (table-driven)
// ---------------------------------------------------------------------------

func TestBuildQualityFromMappings(t *testing.T) {
	// Helper to create classified side effects.
	contractual := func(id, typ string) taxonomy.SideEffect {
		return taxonomy.SideEffect{
			ID:   id,
			Type: taxonomy.SideEffectType(typ),
			Classification: &taxonomy.Classification{
				Label:      taxonomy.Contractual,
				Confidence: 90,
			},
		}
	}
	incidental := func(id, typ string) taxonomy.SideEffect {
		return taxonomy.SideEffect{
			ID:   id,
			Type: taxonomy.SideEffectType(typ),
			Classification: &taxonomy.Classification{
				Label:      taxonomy.Incidental,
				Confidence: 90,
			},
		}
	}
	ambiguous := func(id, typ string) taxonomy.SideEffect {
		return taxonomy.SideEffect{
			ID:   id,
			Type: taxonomy.SideEffectType(typ),
			Classification: &taxonomy.Classification{
				Label:      taxonomy.Ambiguous,
				Confidence: 55,
			},
		}
	}
	unclassified := func(id, typ string) taxonomy.SideEffect {
		return taxonomy.SideEffect{
			ID:   id,
			Type: taxonomy.SideEffectType(typ),
		}
	}

	t.Run("full classification with contract coverage and over-specification", func(t *testing.T) {
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{
					Package:  "math_utils",
					Function: "multiply",
					Location: "math_utils.py:10",
				},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "ReturnValue"),
					contractual("se-002", "ErrorReturn"),
					incidental("se-003", "LogOutput"),
				},
			},
		}
		mappings := []protocol.AssertionMappingData{
			{
				TestFunction:      "test_multiply",
				TestFile:          "tests/test_math.py",
				AssertionLocation: "tests/test_math.py:15",
				AssertionType:     "equality",
				TargetFunction:    "multiply",
				TargetPackage:     "math_utils",
				SideEffectType:    "ReturnValue",
				Confidence:        80,
			},
			{
				TestFunction:      "test_multiply",
				TestFile:          "tests/test_math.py",
				AssertionLocation: "tests/test_math.py:16",
				AssertionType:     "equality",
				TargetFunction:    "multiply",
				TargetPackage:     "math_utils",
				SideEffectType:    "LogOutput",
				Confidence:        70,
			},
		}

		reports, summary := BuildQualityFromMappings(mappings, results)

		if len(reports) != 1 {
			t.Fatalf("got %d reports, want 1", len(reports))
		}
		r := reports[0]
		if r.TestFunction != "test_multiply" {
			t.Errorf("TestFunction = %q, want %q", r.TestFunction, "test_multiply")
		}
		// Contract coverage: 1 of 2 contractual effects covered (ReturnValue asserted, ErrorReturn not)
		if r.ContractCoverage.Percentage != 50 {
			t.Errorf("ContractCoverage.Percentage = %v, want 50", r.ContractCoverage.Percentage)
		}
		if r.ContractCoverage.CoveredCount != 1 {
			t.Errorf("CoveredCount = %d, want 1", r.ContractCoverage.CoveredCount)
		}
		if r.ContractCoverage.TotalContractual != 2 {
			t.Errorf("TotalContractual = %d, want 2", r.ContractCoverage.TotalContractual)
		}
		// Over-specification: 1 assertion on incidental effect (LogOutput)
		if r.OverSpecification.Count != 1 {
			t.Errorf("OverSpecification.Count = %d, want 1", r.OverSpecification.Count)
		}
		if r.AssertionCount != 2 {
			t.Errorf("AssertionCount = %d, want 2", r.AssertionCount)
		}
		if r.AssertionDetectionConfidence != 0 {
			t.Errorf("AssertionDetectionConfidence = %d, want 0", r.AssertionDetectionConfidence)
		}

		// Summary
		if summary.TotalTests != 1 {
			t.Errorf("summary.TotalTests = %d, want 1", summary.TotalTests)
		}
		if summary.AverageContractCoverage != 50 {
			t.Errorf("summary.AverageContractCoverage = %v, want 50", summary.AverageContractCoverage)
		}
		if summary.TotalOverSpecifications != 1 {
			t.Errorf("summary.TotalOverSpecifications = %d, want 1", summary.TotalOverSpecifications)
		}
	})

	t.Run("no mappings produces empty report", func(t *testing.T) {
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{Package: "pkg", Function: "fn"},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "ReturnValue"),
				},
			},
		}

		reports, summary := BuildQualityFromMappings(nil, results)

		if len(reports) != 0 {
			t.Fatalf("got %d reports, want 0", len(reports))
		}
		if summary.TotalTests != 0 {
			t.Errorf("summary.TotalTests = %d, want 0", summary.TotalTests)
		}
	})

	t.Run("empty mappings slice produces empty report", func(t *testing.T) {
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{Package: "pkg", Function: "fn"},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "ReturnValue"),
				},
			},
		}

		reports, summary := BuildQualityFromMappings([]protocol.AssertionMappingData{}, results)

		if len(reports) != 0 {
			t.Fatalf("got %d reports, want 0", len(reports))
		}
		if summary.TotalTests != 0 {
			t.Errorf("summary.TotalTests = %d, want 0", summary.TotalTests)
		}
	})

	t.Run("unclassified effects treated as contractual", func(t *testing.T) {
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{Package: "pkg", Function: "fn"},
				SideEffects: []taxonomy.SideEffect{
					unclassified("se-001", "ReturnValue"),
					unclassified("se-002", "ErrorReturn"),
				},
			},
		}
		mappings := []protocol.AssertionMappingData{
			{
				TestFunction:   "test_fn",
				TestFile:       "test.py",
				TargetFunction: "fn",
				TargetPackage:  "pkg",
				SideEffectType: "ReturnValue",
				Confidence:     80,
			},
		}

		reports, _ := BuildQualityFromMappings(mappings, results)

		if len(reports) != 1 {
			t.Fatalf("got %d reports, want 1", len(reports))
		}
		// 1 of 2 unclassified (= contractual) effects covered
		if reports[0].ContractCoverage.Percentage != 50 {
			t.Errorf("ContractCoverage.Percentage = %v, want 50", reports[0].ContractCoverage.Percentage)
		}
		if reports[0].ContractCoverage.TotalContractual != 2 {
			t.Errorf("TotalContractual = %d, want 2", reports[0].ContractCoverage.TotalContractual)
		}
	})

	t.Run("multiple test functions produce separate reports", func(t *testing.T) {
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{Package: "pkg", Function: "fn"},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "ReturnValue"),
				},
			},
		}
		mappings := []protocol.AssertionMappingData{
			{
				TestFunction:   "test_fn_basic",
				TestFile:       "test.py",
				TargetFunction: "fn",
				TargetPackage:  "pkg",
				SideEffectType: "ReturnValue",
				Confidence:     90,
			},
			{
				TestFunction:   "test_fn_edge",
				TestFile:       "test.py",
				TargetFunction: "fn",
				TargetPackage:  "pkg",
				SideEffectType: "ReturnValue",
				Confidence:     85,
			},
		}

		reports, summary := BuildQualityFromMappings(mappings, results)

		if len(reports) != 2 {
			t.Fatalf("got %d reports, want 2", len(reports))
		}
		// Reports sorted by test function name
		if reports[0].TestFunction != "test_fn_basic" {
			t.Errorf("reports[0].TestFunction = %q, want %q", reports[0].TestFunction, "test_fn_basic")
		}
		if reports[1].TestFunction != "test_fn_edge" {
			t.Errorf("reports[1].TestFunction = %q, want %q", reports[1].TestFunction, "test_fn_edge")
		}
		if summary.TotalTests != 2 {
			t.Errorf("summary.TotalTests = %d, want 2", summary.TotalTests)
		}
		// Both tests cover the same function's effect → 100% each
		if summary.AverageContractCoverage != 100 {
			t.Errorf("summary.AverageContractCoverage = %v, want 100", summary.AverageContractCoverage)
		}
	})

	t.Run("ambiguous effects excluded from contract coverage", func(t *testing.T) {
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{Package: "pkg", Function: "fn"},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "ReturnValue"),
					ambiguous("se-002", "MapMutation"),
				},
			},
		}
		mappings := []protocol.AssertionMappingData{
			{
				TestFunction:   "test_fn",
				TestFile:       "test.py",
				TargetFunction: "fn",
				TargetPackage:  "pkg",
				SideEffectType: "ReturnValue",
				Confidence:     80,
			},
		}

		reports, _ := BuildQualityFromMappings(mappings, results)

		if len(reports) != 1 {
			t.Fatalf("got %d reports, want 1", len(reports))
		}
		// Ambiguous effect excluded → 1 contractual, 1 covered = 100%
		if reports[0].ContractCoverage.Percentage != 100 {
			t.Errorf("ContractCoverage.Percentage = %v, want 100", reports[0].ContractCoverage.Percentage)
		}
		if len(reports[0].AmbiguousEffects) != 1 {
			t.Errorf("AmbiguousEffects = %d, want 1", len(reports[0].AmbiguousEffects))
		}
	})

	t.Run("worst coverage tests in summary", func(t *testing.T) {
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{Package: "pkg", Function: "fn"},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "ReturnValue"),
					contractual("se-002", "ErrorReturn"),
				},
			},
		}
		// Two tests: one covers both effects (100%), one covers none (0%)
		mappings := []protocol.AssertionMappingData{
			{
				TestFunction:   "test_fn_full",
				TestFile:       "test.py",
				TargetFunction: "fn",
				TargetPackage:  "pkg",
				SideEffectType: "ReturnValue",
				Confidence:     90,
			},
			{
				TestFunction:   "test_fn_full",
				TestFile:       "test.py",
				TargetFunction: "fn",
				TargetPackage:  "pkg",
				SideEffectType: "ErrorReturn",
				Confidence:     90,
			},
			{
				TestFunction:      "test_fn_partial",
				TestFile:          "test.py",
				AssertionLocation: "test.py:30",
				AssertionType:     "equality",
				TargetFunction:    "fn",
				TargetPackage:     "pkg",
				SideEffectType:    "UnknownType", // won't match any effect
				Confidence:        50,
			},
		}

		_, summary := BuildQualityFromMappings(mappings, results)

		if len(summary.WorstCoverageTests) != 2 {
			t.Fatalf("WorstCoverageTests = %d, want 2", len(summary.WorstCoverageTests))
		}
		// Worst first
		if summary.WorstCoverageTests[0].ContractCoverage.Percentage != 0 {
			t.Errorf("worst[0].Percentage = %v, want 0", summary.WorstCoverageTests[0].ContractCoverage.Percentage)
		}
		if summary.WorstCoverageTests[1].ContractCoverage.Percentage != 100 {
			t.Errorf("worst[1].Percentage = %v, want 100", summary.WorstCoverageTests[1].ContractCoverage.Percentage)
		}
	})

	t.Run("worst coverage truncated to 5 with 6+ test functions", func(t *testing.T) {
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{Package: "pkg", Function: "fn"},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "ReturnValue"),
					contractual("se-002", "ErrorReturn"),
					contractual("se-003", "PointerArgMutation"),
				},
			},
		}
		// 6 test functions with varying coverage:
		// test_a covers 0/3, test_b covers 1/3, test_c covers 2/3,
		// test_d covers 3/3, test_e covers 0/3, test_f covers 1/3
		types := []string{"ReturnValue", "ErrorReturn", "PointerArgMutation"}
		testCoverages := []struct {
			name   string
			covers []string // which effect types this test covers
		}{
			{"test_a", nil},
			{"test_b", types[:1]},
			{"test_c", types[:2]},
			{"test_d", types[:3]},
			{"test_e", nil},
			{"test_f", types[:1]},
		}
		var mappings []protocol.AssertionMappingData
		for _, tc := range testCoverages {
			if len(tc.covers) == 0 {
				// Need at least one mapping to create a report for this test.
				mappings = append(mappings, protocol.AssertionMappingData{
					TestFunction:   tc.name,
					TestFile:       "test.py",
					TargetFunction: "fn",
					TargetPackage:  "pkg",
					SideEffectType: "UnknownType", // won't match
					Confidence:     50,
				})
			}
			for _, typ := range tc.covers {
				mappings = append(mappings, protocol.AssertionMappingData{
					TestFunction:   tc.name,
					TestFile:       "test.py",
					TargetFunction: "fn",
					TargetPackage:  "pkg",
					SideEffectType: typ,
					Confidence:     80,
				})
			}
		}

		_, summary := BuildQualityFromMappings(mappings, results)

		if len(summary.WorstCoverageTests) != 5 {
			t.Fatalf("WorstCoverageTests = %d, want 5 (truncated from 6)",
				len(summary.WorstCoverageTests))
		}
		// Worst coverage first (ascending order by percentage)
		if summary.WorstCoverageTests[0].ContractCoverage.Percentage != 0 {
			t.Errorf("worst[0].Percentage = %v, want 0",
				summary.WorstCoverageTests[0].ContractCoverage.Percentage)
		}
		// Best coverage last (100% should not appear — it's the 6th)
		last := summary.WorstCoverageTests[4]
		if last.ContractCoverage.Percentage >= 100 {
			t.Errorf("worst[4].Percentage = %v, should be < 100 (100%% test should be truncated)",
				last.ContractCoverage.Percentage)
		}
		if summary.TotalTests != 6 {
			t.Errorf("TotalTests = %d, want 6", summary.TotalTests)
		}
	})

	t.Run("nil results with mappings still produces reports", func(t *testing.T) {
		mappings := []protocol.AssertionMappingData{
			{
				TestFunction:   "test_fn",
				TestFile:       "test.py",
				TargetFunction: "fn",
				TargetPackage:  "pkg",
				SideEffectType: "ReturnValue",
				Confidence:     80,
			},
		}

		reports, summary := BuildQualityFromMappings(mappings, nil)

		if len(reports) != 1 {
			t.Fatalf("got %d reports, want 1", len(reports))
		}
		// No effects → zero coverage
		if reports[0].ContractCoverage.Percentage != 0 {
			t.Errorf("ContractCoverage.Percentage = %v, want 0", reports[0].ContractCoverage.Percentage)
		}
		if summary.TotalTests != 1 {
			t.Errorf("summary.TotalTests = %d, want 1", summary.TotalTests)
		}
	})

	t.Run("multi-target deduplicates shared effects across targets", func(t *testing.T) {
		// Both targets share an ErrorReturn effect with the same ID.
		// The union must count it only once.
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{
					Package:  "db",
					Function: "Save",
					Location: "db.go:10",
				},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "DatabaseWrite"),
					contractual("se-shared", "ErrorReturn"),
				},
			},
			{
				Target: taxonomy.FunctionTarget{
					Package:  "cache",
					Function: "Invalidate",
					Location: "cache.go:20",
				},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-shared", "ErrorReturn"), // same ID as db.Save
					contractual("se-003", "MapMutation"),
				},
			},
		}

		mappings := []protocol.AssertionMappingData{
			{
				TestFunction:   "test_save_and_invalidate",
				TestFile:       "integration_test.py",
				TargetFunction: "Save",
				TargetPackage:  "db",
				SideEffectType: "DatabaseWrite",
				Confidence:     85,
			},
			{
				TestFunction:   "test_save_and_invalidate",
				TestFile:       "integration_test.py",
				TargetFunction: "Save",
				TargetPackage:  "db",
				SideEffectType: "ErrorReturn",
				Confidence:     80,
			},
			{
				TestFunction:   "test_save_and_invalidate",
				TestFile:       "integration_test.py",
				TargetFunction: "Invalidate",
				TargetPackage:  "cache",
				SideEffectType: "MapMutation",
				Confidence:     75,
			},
		}

		reports, summary := BuildQualityFromMappings(mappings, results)

		if len(reports) != 1 {
			t.Fatalf("got %d reports, want 1", len(reports))
		}
		r := reports[0]
		// Shared ErrorReturn (se-shared) must be counted only once in the
		// union. Total distinct effects: DatabaseWrite + ErrorReturn +
		// MapMutation = 3 (not 4).
		if r.ContractCoverage.TotalContractual != 3 {
			t.Errorf("TotalContractual = %d, want 3 (shared effect deduplicated)",
				r.ContractCoverage.TotalContractual)
		}
		if r.ContractCoverage.CoveredCount != 3 {
			t.Errorf("CoveredCount = %d, want 3", r.ContractCoverage.CoveredCount)
		}
		if r.ContractCoverage.Percentage != 100 {
			t.Errorf("ContractCoverage.Percentage = %v, want 100", r.ContractCoverage.Percentage)
		}
		if summary.TotalTests != 1 {
			t.Errorf("summary.TotalTests = %d, want 1", summary.TotalTests)
		}
	})

	t.Run("multi-target test unions effects across targets", func(t *testing.T) {
		// Two distinct target functions with different side effects.
		results := []taxonomy.AnalysisResult{
			{
				Target: taxonomy.FunctionTarget{
					Package:  "db",
					Function: "Save",
					Location: "db.go:10",
				},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-001", "DatabaseWrite"),
					contractual("se-002", "ErrorReturn"),
				},
			},
			{
				Target: taxonomy.FunctionTarget{
					Package:  "cache",
					Function: "Invalidate",
					Location: "cache.go:20",
				},
				SideEffects: []taxonomy.SideEffect{
					contractual("se-003", "MapMutation"),
				},
			},
		}

		// One integration test exercises both targets.
		mappings := []protocol.AssertionMappingData{
			{
				TestFunction:   "test_save_and_invalidate",
				TestFile:       "integration_test.py",
				TargetFunction: "Save",
				TargetPackage:  "db",
				SideEffectType: "DatabaseWrite",
				Confidence:     85,
			},
			{
				TestFunction:   "test_save_and_invalidate",
				TestFile:       "integration_test.py",
				TargetFunction: "Save",
				TargetPackage:  "db",
				SideEffectType: "ErrorReturn",
				Confidence:     80,
			},
			{
				TestFunction:   "test_save_and_invalidate",
				TestFile:       "integration_test.py",
				TargetFunction: "Invalidate",
				TargetPackage:  "cache",
				SideEffectType: "MapMutation",
				Confidence:     75,
			},
		}

		reports, summary := BuildQualityFromMappings(mappings, results)

		if len(reports) != 1 {
			t.Fatalf("got %d reports, want 1", len(reports))
		}
		r := reports[0]
		if r.TestFunction != "test_save_and_invalidate" {
			t.Errorf("TestFunction = %q, want %q", r.TestFunction, "test_save_and_invalidate")
		}
		// All 3 contractual effects from both targets should be in the
		// union: DatabaseWrite, ErrorReturn, MapMutation.
		// All 3 are asserted → 100% contract coverage.
		if r.ContractCoverage.TotalContractual != 3 {
			t.Errorf("TotalContractual = %d, want 3 (union of both targets' effects)",
				r.ContractCoverage.TotalContractual)
		}
		if r.ContractCoverage.CoveredCount != 3 {
			t.Errorf("CoveredCount = %d, want 3", r.ContractCoverage.CoveredCount)
		}
		if r.ContractCoverage.Percentage != 100 {
			t.Errorf("ContractCoverage.Percentage = %v, want 100", r.ContractCoverage.Percentage)
		}
		if r.AssertionCount != 3 {
			t.Errorf("AssertionCount = %d, want 3", r.AssertionCount)
		}
		if summary.TotalTests != 1 {
			t.Errorf("summary.TotalTests = %d, want 1", summary.TotalTests)
		}
		if summary.AverageContractCoverage != 100 {
			t.Errorf("summary.AverageContractCoverage = %v, want 100", summary.AverageContractCoverage)
		}
	})
}

// ---------------------------------------------------------------------------
// computeOverSpecification tests (table-driven)
// ---------------------------------------------------------------------------

func TestComputeOverSpecification(t *testing.T) {
	t.Run("no incidental effects", func(t *testing.T) {
		effects := []taxonomy.SideEffect{
			{
				ID:   "se-001",
				Type: "ReturnValue",
				Classification: &taxonomy.Classification{
					Label:      taxonomy.Contractual,
					Confidence: 90,
				},
			},
		}
		mappings := []taxonomy.AssertionMapping{
			{SideEffectID: "se-001", Confidence: 80},
		}

		os := computeOverSpecification(effects, mappings)

		if os.Count != 0 {
			t.Errorf("Count = %d, want 0", os.Count)
		}
		if os.Ratio != 0 {
			t.Errorf("Ratio = %v, want 0", os.Ratio)
		}
	})

	t.Run("all incidental assertions", func(t *testing.T) {
		effects := []taxonomy.SideEffect{
			{
				ID:   "se-001",
				Type: "LogOutput",
				Classification: &taxonomy.Classification{
					Label:      taxonomy.Incidental,
					Confidence: 90,
				},
			},
		}
		mappings := []taxonomy.AssertionMapping{
			{SideEffectID: "se-001", Confidence: 80},
		}

		os := computeOverSpecification(effects, mappings)

		if os.Count != 1 {
			t.Errorf("Count = %d, want 1", os.Count)
		}
		if os.Ratio != 1.0 {
			t.Errorf("Ratio = %v, want 1.0", os.Ratio)
		}
		if len(os.Suggestions) != 1 {
			t.Errorf("Suggestions = %d, want 1", len(os.Suggestions))
		}
	})

	t.Run("empty mappings", func(t *testing.T) {
		os := computeOverSpecification(nil, nil)

		if os.Count != 0 {
			t.Errorf("Count = %d, want 0", os.Count)
		}
		if os.Ratio != 0 {
			t.Errorf("Ratio = %v, want 0", os.Ratio)
		}
	})

	t.Run("unmapped assertions not counted as over-specification", func(t *testing.T) {
		effects := []taxonomy.SideEffect{
			{
				ID:   "se-001",
				Type: "LogOutput",
				Classification: &taxonomy.Classification{
					Label:      taxonomy.Incidental,
					Confidence: 90,
				},
			},
		}
		// Mapping with empty SideEffectID (unmapped — findSideEffectID
		// returned "" because the type didn't match any effect).
		mappings := []taxonomy.AssertionMapping{
			{SideEffectID: "", Confidence: 80},
		}

		os := computeOverSpecification(effects, mappings)

		// Empty SideEffectID should be skipped — not counted as
		// over-specification even though an incidental effect exists.
		if os.Count != 0 {
			t.Errorf("Count = %d, want 0 (empty SideEffectID should be skipped)", os.Count)
		}
	})
}
