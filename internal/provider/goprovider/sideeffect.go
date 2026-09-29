package goprovider

import (
	"fmt"

	"golang.org/x/tools/go/packages"

	"github.com/unbound-force/gaze/v2/internal/analysis"
	"github.com/unbound-force/gaze/v2/internal/classify"
	"github.com/unbound-force/gaze/v2/internal/config"
	"github.com/unbound-force/gaze/v2/internal/crap"
	"github.com/unbound-force/gaze/v2/internal/loader"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// GoSideEffectAnalyzer implements crap.SideEffectAnalyzer by wrapping
// analysis.LoadAndAnalyze and classify.Classify. It handles both
// side effect detection and classification internally — the returned
// AnalysisResult entries have Classification already attached to each
// SideEffect.
//
// Design decision D5: SideEffectAnalyzer is NOT a field on
// crap.Options. It is consumed only by ContractCoverageProvider
// implementations as a composition dependency.
type GoSideEffectAnalyzer struct {
	// Config is the Gaze configuration for classification
	// thresholds. If nil, defaults are used.
	Config *config.GazeConfig

	// ModulePackages is the list of all packages in the module,
	// used for interface satisfaction and caller analysis during
	// classification.
	ModulePackages []*packages.Package

	// TargetPkg is the loaded target package for AST access
	// during classification. When nil, the analyzer loads the
	// target package from the pkgPath argument.
	TargetPkg *packages.Package

	// Verbose controls whether classification signal detail
	// fields (SourceFile, Excerpt, Reasoning) are populated.
	Verbose bool
}

// Ensure GoSideEffectAnalyzer satisfies crap.SideEffectAnalyzer.
var _ crap.SideEffectAnalyzer = (*GoSideEffectAnalyzer)(nil)

// NewSideEffectAnalyzer creates a GoSideEffectAnalyzer with the
// given configuration dependencies. The config parameter may be nil
// (defaults will be used). The modPkgs parameter provides module
// packages for classification signals (may be nil for degraded
// classification).
func NewSideEffectAnalyzer(
	cfg *config.GazeConfig,
	modPkgs []*packages.Package,
	verbose bool,
) *GoSideEffectAnalyzer {
	return &GoSideEffectAnalyzer{
		Config:         cfg,
		ModulePackages: modPkgs,
		Verbose:        verbose,
	}
}

// Analyze detects and classifies side effects for all functions in
// the given package path. Returns classified results where every
// SideEffect has a non-zero Classification with a valid Label
// (contractual, ambiguous, or incidental).
//
// The method calls analysis.LoadAndAnalyze for side effect detection,
// then classify.Classify for classification. Go's 5-signal
// classification model (interface satisfaction, visibility, callers,
// naming, godoc) stays internal to this adapter.
func (a *GoSideEffectAnalyzer) Analyze(pkgPath string) ([]taxonomy.AnalysisResult, error) {
	analysisOpts := analysis.Options{
		IncludeUnexported: loader.IsMainPkg(pkgPath),
	}

	results, err := analysis.LoadAndAnalyze(pkgPath, analysisOpts)
	if err != nil {
		return nil, fmt.Errorf("side effect analysis for %s: %w", pkgPath, err)
	}
	if len(results) == 0 {
		return results, nil
	}

	// Load the target package for AST access if not pre-set.
	targetPkg := a.TargetPkg
	if targetPkg == nil {
		targetResult, loadErr := loader.Load(pkgPath)
		if loadErr != nil {
			return nil, fmt.Errorf("loading target package for classification: %w", loadErr)
		}
		targetPkg = targetResult.Pkg
	}

	clOpts := classify.Options{
		Config:         a.Config,
		ModulePackages: a.ModulePackages,
		TargetPkg:      targetPkg,
		Verbose:        a.Verbose,
	}

	classified := classify.Classify(results, clOpts)
	return classified, nil
}
