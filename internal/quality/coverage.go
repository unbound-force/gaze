package quality

import (
	"github.com/jflowers/gaze/internal/taxonomy"
)

// ComputeContractCoverage computes the Contract Coverage metric:
// the ratio of contractual side effects that are asserted on by
// at least one assertion mapping.
//
// Ambiguous side effects are excluded from both numerator and
// denominator. Effects with no classification are treated as
// contractual (conservative assumption).
func ComputeContractCoverage(
	effects []taxonomy.SideEffect,
	mappings []taxonomy.AssertionMapping,
) taxonomy.ContractCoverage {
	// Build a set of asserted side effect IDs.
	assertedIDs := make(map[string]bool, len(mappings))
	for _, m := range mappings {
		if m.SideEffectID != "" {
			assertedIDs[m.SideEffectID] = true
		}
	}

	// Count contractual effects and coverage.
	var totalContractual int
	var coveredCount int
	var gaps []taxonomy.SideEffect

	for _, e := range effects {
		// Skip ambiguous effects — they are excluded from the metric.
		if e.Classification != nil && e.Classification.Label == taxonomy.Ambiguous {
			continue
		}

		// Skip incidental effects — they are not part of the contract.
		if e.Classification != nil && e.Classification.Label == taxonomy.Incidental {
			continue
		}

		// This effect is contractual (either explicitly labeled or
		// unclassified, which we treat conservatively as contractual).
		totalContractual++

		if assertedIDs[e.ID] {
			coveredCount++
		} else {
			gaps = append(gaps, e)
		}
	}

	// Compute percentage.
	var percentage float64
	if totalContractual > 0 {
		percentage = float64(coveredCount) * 100.0 / float64(totalContractual)
	}

	return taxonomy.ContractCoverage{
		Percentage:       percentage,
		CoveredCount:     coveredCount,
		TotalContractual: totalContractual,
		Gaps:             gaps,
	}
}
