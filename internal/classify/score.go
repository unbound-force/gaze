// Package classify implements the contractual classification engine.
package classify

import (
	"fmt"

	"github.com/unbound-force/gaze/v2/internal/config"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// baseConfidence is the neutral starting point for confidence
// scoring. Signals adjust from this base: positive pushes toward
// contractual, negative pushes toward incidental.
const baseConfidence = 50

// maxContradictionPenalty is the maximum penalty for contradicting
// signals (FR-007).
const maxContradictionPenalty = 20

// tierBoost returns the confidence boost for a side effect based on
// its priority tier. P0 effects (ReturnValue, ErrorReturn, etc.) are
// definitionally contractual — a function's direct observable outputs
// — and receive +25. P1 effects get a moderate +10. P2-P4 effects
// stay at the base (no boost) since their contractual nature
// genuinely depends on context.
func tierBoost(effectType taxonomy.SideEffectType) int {
	switch taxonomy.TierOf(effectType) {
	case taxonomy.TierP0:
		return 25
	case taxonomy.TierP1:
		return 10
	default:
		return 0
	}
}

// accumulateSignals sums signal weights starting from
// baseConfidence + tierBoost(effectType), skipping zero-weight/
// empty-source signals. The effective starting score is 75 for P0,
// 60 for P1, and 50 for P2-P4. Returns the accumulated score and
// whether positive and negative signals were both present.
func accumulateSignals(effectType taxonomy.SideEffectType, signals []taxonomy.Signal) (score int, hasPositive, hasNegative bool) {
	score = baseConfidence + tierBoost(effectType)
	for _, s := range signals {
		if s.Weight == 0 && s.Source == "" {
			continue
		}
		score += s.Weight
		if s.Weight > 0 {
			hasPositive = true
		}
		if s.Weight < 0 {
			hasNegative = true
		}
	}
	return score, hasPositive, hasNegative
}

// classifyLabel determines the classification label and reasoning
// string based on the score and configured thresholds.
func classifyLabel(score, contractualThreshold, incidentalThreshold int) (taxonomy.ClassificationLabel, string) {
	switch {
	case score >= contractualThreshold:
		return taxonomy.Contractual, fmt.Sprintf(
			"confidence %d >= %d (contractual threshold)",
			score, contractualThreshold,
		)
	case score < incidentalThreshold:
		return taxonomy.Incidental, fmt.Sprintf(
			"confidence %d < %d (incidental threshold)",
			score, incidentalThreshold,
		)
	default:
		return taxonomy.Ambiguous, fmt.Sprintf(
			"confidence %d in ambiguous range [%d, %d)",
			score, incidentalThreshold, contractualThreshold,
		)
	}
}

// ComputeScore computes the confidence score from a set of signals,
// applies a tier-based boost, contradiction detection and penalty,
// clamps to 0-100, and returns a Classification based on the
// configured thresholds. The effectType determines the tier boost:
// P0 effects start at 75, P1 at 60, P2-P4 at 50.
func ComputeScore(effectType taxonomy.SideEffectType, signals []taxonomy.Signal, cfg *config.GazeConfig) taxonomy.Classification {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	score, hasPositive, hasNegative := accumulateSignals(effectType, signals)

	// Apply contradiction penalty if both positive and negative
	// signals exist.
	contradictionApplied := false
	if hasPositive && hasNegative {
		score -= maxContradictionPenalty
		contradictionApplied = true
		signals = append(signals, taxonomy.Signal{
			Source:    "contradiction",
			Weight:    -maxContradictionPenalty,
			Reasoning: "contradicting signals detected — positive and negative evidence both present",
		})
	}

	// Clamp to 0-100.
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	label, reasoning := classifyLabel(
		score,
		cfg.Classification.Thresholds.Contractual,
		cfg.Classification.Thresholds.Incidental,
	)

	if contradictionApplied {
		reasoning += "; contradiction penalty applied"
	}

	// Filter out empty signals from the result.
	filtered := make([]taxonomy.Signal, 0, len(signals))
	for _, s := range signals {
		if s.Source != "" {
			filtered = append(filtered, s)
		}
	}

	return taxonomy.Classification{
		Label:      label,
		Confidence: score,
		Signals:    filtered,
		Reasoning:  reasoning,
	}
}
