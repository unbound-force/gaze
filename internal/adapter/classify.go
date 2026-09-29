package adapter

import (
	"context"
	"fmt"
	"io"

	"github.com/unbound-force/gaze/v2/internal/classify"
	"github.com/unbound-force/gaze/v2/internal/config"
	"github.com/unbound-force/gaze/v2/internal/protocol"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// fetchClassifySignals calls the classify_signals protocol method on
// the external analyzer and returns the raw signal data. On failure
// (transport error, protocol error, or unmarshal error), it warns to
// stderr and returns nil — callers do not need to distinguish between
// "no signals available" and "classify_signals failed."
//
// This is a standalone function (not a method) for testability — it
// can be tested with a mock client without constructing a full
// ExternalSideEffectAnalyzer.
func fetchClassifySignals(
	client *protocol.Client,
	rootDir string,
	patterns []string,
	stderr io.Writer,
) []protocol.ClassifySignalData {
	ctx, cancel := context.WithTimeout(context.Background(), protocol.AnalysisTimeout)
	defer cancel()

	result, err := callAndUnmarshal[protocol.ClassifySignalsResult](
		ctx, client, protocol.MethodClassifySignals, protocol.ClassifySignalsParams{
			RootPath: rootDir,
			Patterns: patterns,
		},
	)
	if err != nil {
		if stderr != nil {
			_, _ = fmt.Fprintf(stderr, "warning: classify_signals failed: %v\n", err)
		}
		return nil
	}

	return result.Signals
}

// mergeClassifications groups signals by (Package, Function,
// SideEffectType) and attaches computed classifications to cached
// effects. Effects that already have a non-nil Classification (inline
// from the analyze response) are preserved — the analyzer's explicit
// classification takes precedence (design D6).
//
// When a function has multiple effects of the same SideEffectType,
// the computed classification is applied to all matching effects that
// have nil Classification. Signals for (Package, Function,
// SideEffectType) tuples with no corresponding cached effect are
// silently discarded.
//
// When cfg is nil, config.DefaultConfig() is used.
func mergeClassifications(
	cached []taxonomy.AnalysisResult,
	signals []protocol.ClassifySignalData,
	cfg *config.GazeConfig,
) {
	if len(signals) == 0 {
		return
	}

	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	// Group signals by (Package, Function, SideEffectType).
	type signalKey struct {
		pkg      string
		function string
		seType   string
	}
	grouped := make(map[signalKey][]taxonomy.Signal)
	for _, s := range signals {
		key := signalKey{
			pkg:      s.Package,
			function: s.Function,
			seType:   s.SideEffectType,
		}
		grouped[key] = append(grouped[key], taxonomy.Signal{
			Source:    s.Source,
			Weight:    s.Weight,
			Reasoning: s.Reasoning,
		})
	}

	// Match grouped signals to cached effects and compute classifications.
	for i := range cached {
		r := &cached[i]
		for j := range r.SideEffects {
			e := &r.SideEffects[j]

			// Preserve pre-classified effects (design D6).
			if e.Classification != nil {
				continue
			}

			key := signalKey{
				pkg:      r.Target.Package,
				function: r.Target.Function,
				seType:   string(e.Type),
			}
			sigs, ok := grouped[key]
			if !ok {
				continue
			}

			c := classify.ComputeScore(e.Type, sigs, cfg)
			e.Classification = &c
		}
	}
}
