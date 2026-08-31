package adapter

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"sync"

	"github.com/unbound-force/gaze/internal/protocol"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// ExternalSideEffectAnalyzer implements crap.SideEffectAnalyzer by
// calling the "analyze" protocol method on an external analyzer.
//
// The analyzer returns per-function results for the whole project.
// This adapter caches the full result set and filters by pkgPath
// when Analyze(pkgPath) is called. Design decision D4: the adapter
// is a composition dependency of ExternalContractCoverageProvider,
// not a standalone adapter passed to crap.Options.
type ExternalSideEffectAnalyzer struct {
	client *protocol.Client
	caps   protocol.Capabilities
	stderr io.Writer

	// rootDir and patterns are set during session initialization
	// and used for the analyze call.
	rootDir  string
	patterns []string

	// mu protects the cached results.
	mu     sync.Mutex
	cached []taxonomy.AnalysisResult
	loaded bool
}

// NewExternalSideEffectAnalyzer creates a side effect analyzer that
// delegates to the given protocol client. The capabilities determine
// whether classify_signals is also called.
func NewExternalSideEffectAnalyzer(
	client *protocol.Client,
	caps protocol.Capabilities,
	rootDir string,
	patterns []string,
	stderr io.Writer,
) *ExternalSideEffectAnalyzer {
	return &ExternalSideEffectAnalyzer{
		client:   client,
		caps:     caps,
		rootDir:  rootDir,
		patterns: patterns,
		stderr:   stderr,
	}
}

// Analyze returns side effect analysis results for the given package
// path. On the first call, it fetches all results from the external
// analyzer and caches them. Subsequent calls filter the cache.
func (a *ExternalSideEffectAnalyzer) Analyze(pkgPath string) ([]taxonomy.AnalysisResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.loaded {
		if err := a.loadAll(); err != nil {
			return nil, err
		}
		a.loaded = true
	}

	// Filter cached results by package path.
	var filtered []taxonomy.AnalysisResult
	for _, r := range a.cached {
		if r.Target.Package == pkgPath {
			filtered = append(filtered, r)
		}
	}
	return filtered, nil
}

// AllResults returns all cached analysis results across all packages.
// Must be called after at least one Analyze call. This is used by
// ExternalContractCoverageProvider to access the full result set.
func (a *ExternalSideEffectAnalyzer) AllResults() ([]taxonomy.AnalysisResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if !a.loaded {
		if err := a.loadAll(); err != nil {
			return nil, err
		}
		a.loaded = true
	}
	return a.cached, nil
}

// loadAll fetches all analysis results from the external analyzer.
// When the analyzer declares streaming capability, it uses the
// analyze/stream method with JSONL output; otherwise it uses the
// batch analyze method. Must be called with a.mu held.
func (a *ExternalSideEffectAnalyzer) loadAll() error {
	if a.caps.Streaming {
		return a.loadStreaming()
	}
	return a.loadBatch()
}

// loadBatch fetches results using the batch analyze method.
func (a *ExternalSideEffectAnalyzer) loadBatch() error {
	ctx, cancel := context.WithTimeout(context.Background(), protocol.AnalysisTimeout)
	defer cancel()

	result, err := callAndUnmarshal[protocol.AnalyzeResult](ctx, a.client, protocol.MethodAnalyze, protocol.AnalyzeParams{
		RootPath: a.rootDir,
		Patterns: a.patterns,
	})
	if err != nil {
		return err
	}

	a.cached = convertAnalysisResults(result.Functions, a.stderr)
	return nil
}

// loadStreaming fetches results using the analyze/stream method.
// The analyzer writes one AnalyzedFunction per line (JSONL).
// Malformed lines cause a fail-fast error with line number context.
func (a *ExternalSideEffectAnalyzer) loadStreaming() error {
	ctx, cancel := context.WithTimeout(context.Background(), protocol.AnalysisTimeout)
	defer cancel()

	scanner, err := a.client.CallStream(ctx, protocol.MethodAnalyzeStream, protocol.AnalyzeParams{
		RootPath: a.rootDir,
		Patterns: a.patterns,
	})
	if err != nil {
		return fmt.Errorf("analyze/stream protocol call: %w", err)
	}

	funcs, err := parseSideEffectStream(scanner)
	if err != nil {
		return err
	}

	a.cached = convertAnalysisResults(funcs, a.stderr)
	return nil
}

// parseSideEffectStream reads JSONL-encoded AnalyzedFunction records
// from scanner. Empty lines are skipped. Malformed lines cause a
// fail-fast error with line number context and a truncated excerpt.
func parseSideEffectStream(scanner *bufio.Scanner) ([]protocol.AnalyzedFunction, error) {
	var funcs []protocol.AnalyzedFunction
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Bytes()
		if len(line) == 0 {
			continue // skip empty lines
		}

		var fn protocol.AnalyzedFunction
		if uerr := json.Unmarshal(line, &fn); uerr != nil {
			return nil, fmt.Errorf("malformed JSONL on line %d: %w (content: %s)",
				lineNum, uerr, truncateBytes(line, 200))
		}
		funcs = append(funcs, fn)
	}
	if scanErr := scanner.Err(); scanErr != nil {
		return nil, fmt.Errorf("reading analyze/stream response: %w", scanErr)
	}
	return funcs, nil
}

// truncateBytes returns the first n bytes as a string, appending
// "..." if truncated.
func truncateBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}

// convertAnalysisResults maps protocol AnalyzedFunction entries to
// taxonomy.AnalysisResult. Unknown SideEffectType values are
// included with a warning logged to stderr (design decision D9).
func convertAnalysisResults(funcs []protocol.AnalyzedFunction, stderr io.Writer) []taxonomy.AnalysisResult {
	results := make([]taxonomy.AnalysisResult, len(funcs))
	for i, f := range funcs {
		effects := make([]taxonomy.SideEffect, len(f.SideEffects))
		for j, se := range f.SideEffects {
			effectType := taxonomy.SideEffectType(se.Type)
			tier := taxonomy.TierOf(effectType)

			// Warn on unknown side effect types (they default to P4).
			if !taxonomy.IsKnownType(effectType) && stderr != nil {
				_, _ = fmt.Fprintf(stderr, "warning: unknown side effect type %q from external analyzer (defaulting to P4)\n", se.Type)
			}

			effect := taxonomy.SideEffect{
				ID:          taxonomy.GenerateID(f.Package, f.Name, se.Type, se.Location),
				Type:        effectType,
				Tier:        tier,
				Location:    se.Location,
				Description: se.Description,
				Target:      se.Target,
				Detail:      se.Detail,
			}

			if se.Classification != nil {
				effect.Classification = &taxonomy.Classification{
					Label:      taxonomy.ClassificationLabel(se.Classification.Label),
					Confidence: se.Classification.Confidence,
				}
			}

			effects[j] = effect
		}

		results[i] = taxonomy.AnalysisResult{
			Target: taxonomy.FunctionTarget{
				Package:  f.Package,
				Function: f.Name,
				Location: fmt.Sprintf("%s:%d", f.File, f.Line),
			},
			SideEffects: effects,
		}
	}
	return results
}
