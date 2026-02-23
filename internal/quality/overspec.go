package quality

import (
	"fmt"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// ComputeOverSpecification computes the Over-Specification Score:
// how many of the test's assertions target incidental side effects
// (implementation details that should not be asserted on).
//
// A high over-specification score indicates that the test is fragile
// and will break when implementation details change without any
// change to the function's contract.
func ComputeOverSpecification(
	effects []taxonomy.SideEffect,
	mappings []taxonomy.AssertionMapping,
) taxonomy.OverSpecificationScore {
	// Build a map from effect ID to classification.
	effectMap := make(map[string]*taxonomy.SideEffect, len(effects))
	for i := range effects {
		effectMap[effects[i].ID] = &effects[i]
	}

	var incidentalAssertions []taxonomy.AssertionMapping
	totalMappings := 0

	for _, m := range mappings {
		if m.SideEffectID == "" {
			continue
		}
		totalMappings++

		effect, ok := effectMap[m.SideEffectID]
		if !ok {
			continue
		}

		if effect.Classification != nil && effect.Classification.Label == taxonomy.Incidental {
			incidentalAssertions = append(incidentalAssertions, m)
		}
	}

	// Compute ratio.
	var ratio float64
	if totalMappings > 0 {
		ratio = float64(len(incidentalAssertions)) / float64(totalMappings)
	}

	// Generate per-assertion suggestions.
	suggestions := make([]string, 0, len(incidentalAssertions))
	for _, m := range incidentalAssertions {
		effect := effectMap[m.SideEffectID]
		if effect != nil {
			suggestion := generateSuggestion(effect.Type, effect.Description)
			suggestions = append(suggestions, suggestion)
		}
	}

	return taxonomy.OverSpecificationScore{
		Count:                len(incidentalAssertions),
		Ratio:                ratio,
		IncidentalAssertions: incidentalAssertions,
		Suggestions:          suggestions,
	}
}

// generateSuggestion maps side effect types to actionable
// recommendations for removing over-specification.
func generateSuggestion(
	effectType taxonomy.SideEffectType,
	description string,
) string {
	switch effectType {
	case taxonomy.LogWrite:
		return fmt.Sprintf("Consider removing assertion on log output — %s is an implementation detail", description)

	case taxonomy.StdoutWrite:
		return fmt.Sprintf("Consider removing assertion on stdout — %s may change without affecting correctness", description)

	case taxonomy.GoroutineSpawn:
		return fmt.Sprintf("Consider removing assertion on goroutine lifecycle — %s is an internal concurrency detail", description)

	case taxonomy.ContextCancellation:
		return fmt.Sprintf("Consider removing assertion on context usage — %s is an implementation detail", description)

	case taxonomy.CallbackInvocation:
		return fmt.Sprintf("Consider removing assertion on callback invocation — %s may be an implementation detail", description)

	default:
		return fmt.Sprintf("Review whether assertion on %s (%s) is testing contract behavior or implementation details",
			effectType, description)
	}
}
