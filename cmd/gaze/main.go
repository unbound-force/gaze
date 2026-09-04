// Package main implements the gaze CLI, a static analysis tool for
// Go that detects observable side effects and computes CRAP scores.
package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	charmlog "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unbound-force/gaze/internal/adapter"
	"github.com/unbound-force/gaze/internal/aireport"
	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/cliutil"
	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/docscan"
	"github.com/unbound-force/gaze/internal/loader"
	"github.com/unbound-force/gaze/internal/provider/goprovider"
	"github.com/unbound-force/gaze/internal/quality"
	"github.com/unbound-force/gaze/internal/report"
	"github.com/unbound-force/gaze/internal/scaffold"
	"github.com/unbound-force/gaze/internal/taxonomy"
	"golang.org/x/tools/go/packages"
)

// logger is the application-wide structured logger (writes to stderr).
var logger = charmlog.NewWithOptions(os.Stderr, charmlog.Options{
	ReportTimestamp: false,
})

// Set by build flags (-ldflags "-X main.version=... -X main.commit=... -X main.date=...").
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	root := &cobra.Command{
		Use:   "gaze",
		Short: "Gaze — test quality analysis via side effect detection",
		Long: `Gaze analyzes Go functions to detect observable side effects
and measures whether unit tests assert on all contractual changes
produced by their test targets.`,
		Version: version,
	}
	// Override the default version template to include commit and build date.
	root.SetVersionTemplate(
		fmt.Sprintf("gaze version %s (commit %s, built %s)\n", version, commit, date),
	)

	root.AddCommand(newAnalyzeCmd())
	root.AddCommand(newCrapCmd())
	root.AddCommand(newInitCmd())
	root.AddCommand(newQualityCmd())
	root.AddCommand(newReportCmd())
	root.AddCommand(newSchemaCmd())
	root.AddCommand(newDocscanCmd())
	root.AddCommand(newSelfCheckCmd())

	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// initParams holds the parsed flags for the init command.
type initParams struct {
	targetDir string
	force     bool
	version   string
	stdout    io.Writer
}

// runInit is the extracted, testable body of the init command.
func runInit(p initParams) error {
	_, err := scaffold.Run(scaffold.Options{
		TargetDir: p.targetDir,
		Force:     p.force,
		Version:   p.version,
		Stdout:    p.stdout,
	})
	return err
}

// newInitCmd creates the "init" subcommand that scaffolds OpenCode
// agent and command files into the current directory.
func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init",
		Short: "Scaffold OpenCode agents and commands for Gaze",
		Long: `Initialize OpenCode integration in the current directory.

Creates .opencode/agents/ and .opencode/commands/ directories with
Gaze's quality reporting agent and commands. After running this,
you can use /gaze in OpenCode to generate quality reports.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			force, _ := cmd.Flags().GetBool("force")
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
			return runInit(initParams{
				targetDir: cwd,
				force:     force,
				version:   version,
				stdout:    cmd.OutOrStdout(),
			})
		},
	}
	cmd.Flags().Bool("force", false, "Overwrite existing files")
	return cmd
}

// analyzeParams holds the parsed flags for the analyze command.
type analyzeParams struct {
	patterns          []string
	format            string
	function          string
	includeUnexported bool
	interactive       bool
	classify          bool
	verbose           bool
	configPath        string
	contractualThresh int
	incidentalThresh  int
	stdout            io.Writer
	stderr            io.Writer
}

// loadConfig loads the GazeConfig from the given path (or searches
// the current directory if path is empty), then applies any CLI
// threshold overrides. A threshold value of -1 means "not set"
// (use config/default). Any other value overrides the loaded config.
//
// Valid threshold values are in [1, 99]. The contractual threshold
// must be strictly greater than the incidental threshold to prevent
// degenerate classifications (e.g., contractual=0 would classify
// every side effect as contractual regardless of signal strength).
func loadConfig(path string, contractualThresh, incidentalThresh int) (*config.GazeConfig, error) {
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return config.DefaultConfig(), nil
		}
		configDir := cwd
		if moduleRoot, findErr := loader.FindModuleRoot(cwd); findErr == nil {
			configDir = moduleRoot
		}
		path = filepath.Join(configDir, ".gaze.yaml")
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, err
	}
	if contractualThresh >= 0 {
		if contractualThresh < 1 || contractualThresh > 99 {
			return nil, fmt.Errorf(
				"--contractual-threshold=%d is invalid: must be in [1, 99]",
				contractualThresh,
			)
		}
		cfg.Classification.Thresholds.Contractual = contractualThresh
	}
	if incidentalThresh >= 0 {
		if incidentalThresh < 1 || incidentalThresh > 99 {
			return nil, fmt.Errorf(
				"--incidental-threshold=%d is invalid: must be in [1, 99]",
				incidentalThresh,
			)
		}
		cfg.Classification.Thresholds.Incidental = incidentalThresh
	}
	// Validate the final thresholds are coherent.
	if cfg.Classification.Thresholds.Contractual <= cfg.Classification.Thresholds.Incidental {
		// Produce an actionable error that tells the user where the bad
		// values came from: CLI flags, the config file, or both.
		source := fmt.Sprintf("config file %s", path)
		if contractualThresh >= 0 || incidentalThresh >= 0 {
			source = "--contractual-threshold / --incidental-threshold flags"
			if contractualThresh >= 0 && incidentalThresh < 0 {
				source = "--contractual-threshold flag"
			} else if incidentalThresh >= 0 && contractualThresh < 0 {
				source = "--incidental-threshold flag"
			}
		}
		return nil, fmt.Errorf(
			"contractual threshold (%d) must be greater than incidental threshold (%d); "+
				"check %s",
			cfg.Classification.Thresholds.Contractual,
			cfg.Classification.Thresholds.Incidental,
			source,
		)
	}
	return cfg, nil
}

// runAnalyze is the extracted, testable body of the analyze command.
func runAnalyze(p analyzeParams) error {
	if err := cliutil.ValidateFormat(p.format); err != nil {
		return err
	}

	// Resolve package patterns to concrete package paths.
	moduleDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	pkgPaths, err := loader.ResolvePackagePaths(p.patterns, moduleDir, p.stderr)
	if err != nil {
		return fmt.Errorf("resolving package patterns: %w", err)
	}
	if len(pkgPaths) == 0 {
		return fmt.Errorf("no packages found for patterns %v", p.patterns)
	}

	// --verbose implies --classify.
	if p.verbose {
		p.classify = true
	}

	// Pre-load config and module packages once (shared across all packages).
	var cfg *config.GazeConfig
	var modPkgs []*packages.Package
	if p.classify {
		contractualThresh := p.contractualThresh
		if contractualThresh == 0 {
			contractualThresh = -1
		}
		incidentalThresh := p.incidentalThresh
		if incidentalThresh == 0 {
			incidentalThresh = -1
		}
		var cfgErr error
		cfg, cfgErr = loadConfig(p.configPath, contractualThresh, incidentalThresh)
		if cfgErr != nil {
			return fmt.Errorf("loading config: %w", cfgErr)
		}

		// Load module once for caller/interface analysis.
		logger.Info("loading module packages for classification")
		modResult, modErr := loader.LoadModule(moduleDir)
		if modErr != nil {
			logger.Warn("module loading failed; caller/interface signals degraded", "err", modErr)
		} else {
			modPkgs = modResult.Packages
		}
	}

	var allResults []taxonomy.AnalysisResult
	for _, pkgPath := range pkgPaths {
		opts := analysis.Options{
			IncludeUnexported: p.includeUnexported,
			FunctionFilter:    p.function,
			Version:           version,
		}
		autoDetectMainPkg(pkgPath, &opts.IncludeUnexported)

		logger.Info("analyzing package", "pkg", pkgPath)
		results, loadErr := analysis.LoadAndAnalyze(pkgPath, opts)
		if loadErr != nil {
			return loadErr
		}

		// Classify per package — each package needs its own target
		// package AST for accurate classification signals.
		if p.classify && len(results) > 0 {
			classified, clErr := runClassify(results, pkgPath, cfg, p.verbose, modPkgs)
			if clErr != nil {
				return fmt.Errorf("classification of %s: %w", pkgPath, clErr)
			}
			results = classified
		}

		allResults = append(allResults, results...)
	}

	if len(allResults) == 0 {
		if p.function != "" {
			return fmt.Errorf("function %q not found in packages %v", p.function, p.patterns)
		}
		logger.Warn("no functions found to analyze")
		return nil
	}

	logger.Info("analysis complete", "functions", len(allResults))

	if p.interactive {
		return runInteractiveAnalyze(allResults)
	}

	switch p.format {
	case "json":
		return report.WriteJSON(p.stdout, allResults, version)
	default:
		textOpts := report.TextOptions{
			Classify: p.classify,
			Verbose:  p.verbose,
		}
		return report.WriteTextOptions(p.stdout, allResults, textOpts)
	}
}

// runClassify runs the mechanical classification pipeline on
// analysis results and returns classified results. It adds a
// metadata warning noting that document-enhanced classification
// is not applied (the gaze-reporter agent handles that in full mode).
//
// When modPkgs is non-nil, it is used directly for caller/interface
// analysis. When nil, the module is loaded from the working directory.
func runClassify(
	results []taxonomy.AnalysisResult,
	pkgPath string,
	cfg *config.GazeConfig,
	verbose bool,
	modPkgs []*packages.Package,
) ([]taxonomy.AnalysisResult, error) {
	// Load the target package for AST access.
	targetResult, err := loader.Load(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("loading target package: %w", err)
	}

	// Load the module for caller/interface analysis if not provided.
	if modPkgs == nil {
		logger.Info("loading module packages for classification")
		cwd, cwdErr := os.Getwd()
		if cwdErr != nil {
			logger.Debug("could not determine working directory for module load", "err", cwdErr)
			cwd = ""
		}
		moduleRoot := cwd
		if cwd != "" {
			if root, findErr := loader.FindModuleRoot(cwd); findErr == nil {
				moduleRoot = root
			} else {
				logger.Warn("could not find module root; classification signals may be degraded", "err", findErr)
			}
		}
		modResult, modErr := loader.LoadModule(moduleRoot)
		if modErr != nil {
			logger.Warn("module loading failed; caller/interface signals degraded", "err", modErr)
		} else {
			modPkgs = modResult.Packages
		}
	}

	clOpts := classify.Options{
		Config:         cfg,
		ModulePackages: modPkgs,
		TargetPkg:      targetResult.Pkg,
		Verbose:        verbose,
	}

	classified := classify.Classify(results, clOpts)

	// Add a warning to each result noting mechanical-only mode.
	for i := range classified {
		classified[i].Metadata.Warnings = append(
			classified[i].Metadata.Warnings,
			"classification: mechanical signals only; "+
				"run /gaze in full mode for document-enhanced results",
		)
	}

	return classified, nil
}

func newAnalyzeCmd() *cobra.Command {
	var (
		function          string
		format            string
		includeUnexported bool
		interactive       bool
		classifyFlag      bool
		verboseFlag       bool
		configPath        string
		contractualThresh int
		incidentalThresh  int
	)

	cmd := &cobra.Command{
		Use:   "analyze [packages...]",
		Short: "Analyze side effects of Go functions",
		Long: `Analyze one or more Go packages and report all observable side
effects each function produces. Accepts multiple package patterns
including ./... wildcards.

Use --classify to attach contractual classification (mechanical signals).
Use /gaze in OpenCode (full mode) for document-enhanced classification.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAnalyze(analyzeParams{
				patterns:          args,
				format:            format,
				function:          function,
				includeUnexported: includeUnexported,
				interactive:       interactive,
				classify:          classifyFlag,
				verbose:           verboseFlag,
				configPath:        configPath,
				contractualThresh: contractualThresh,
				incidentalThresh:  incidentalThresh,
				stdout:            os.Stdout,
				stderr:            os.Stderr,
			})
		},
	}

	cmd.Flags().StringVarP(&function, "function", "f", "",
		"analyze a specific function (default: all exported)")
	cmd.Flags().StringVar(&format, "format", "text",
		"output format: text or json")
	cmd.Flags().BoolVar(&includeUnexported, "include-unexported", false,
		"include unexported functions")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false,
		"launch interactive TUI for browsing results")
	cmd.Flags().BoolVar(&classifyFlag, "classify", false,
		"classify side effects as contractual, incidental, or ambiguous")
	cmd.Flags().BoolVarP(&verboseFlag, "verbose", "v", false,
		"print full signal breakdown (implies --classify)")
	cmd.Flags().StringVar(&configPath, "config", "",
		"path to .gaze.yaml config file (default: search CWD)")
	cmd.Flags().IntVar(&contractualThresh, "contractual-threshold", -1,
		"override contractual confidence threshold (default: from config or 80)")
	cmd.Flags().IntVar(&incidentalThresh, "incidental-threshold", -1,
		"override incidental confidence threshold (default: from config or 50)")

	return cmd
}

// crapParams holds the parsed flags for the crap command.
type crapParams struct {
	patterns        []string
	format          string
	opts            crap.Options
	maxCrapload     int
	maxGazeCrapload int
	moduleDir       string
	aiMapper        string
	aiMapperModel   string
	baselinePath    string
	analyzerFlag    string
	languageFlag    string
	stdout          io.Writer
	stderr          io.Writer

	// thresholdSet is true when any threshold flag was explicitly
	// provided on the command line (via cmd.Flags().Changed). Used
	// by the zero-result gate (#116): when thresholds are set but
	// no functions were analyzed, runCrap returns an error instead
	// of silently passing.
	thresholdSet bool

	// analyzeFunc overrides crap.Analyze for testing.
	// When nil, the production crap.Analyze is called.
	analyzeFunc func([]string, string, crap.Options) (*crap.Report, error)

	// contractProvider overrides the production GoContractCoverageProvider
	// for testing. When non-nil, it is set on opts.ContractCoverageProvider
	// before calling crap.Analyze. When nil and no provider is already set,
	// the production GoContractCoverageProvider is constructed.
	contractProvider crap.ContractCoverageProvider
}

func newSchemaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "schema",
		Short: "Print the JSON Schema for Gaze analysis output",
		Long: `Print the JSON Schema (Draft 2020-12) that documents the
structure of gaze analyze --format=json output. Useful for
validating output or generating client types.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := fmt.Fprintln(cmd.OutOrStdout(), report.Schema)
			return err
		},
	}
}

// runCrap is the extracted, testable body of the crap command.
func runCrap(p crapParams) error {
	if err := cliutil.ValidateFormat(p.format); err != nil {
		return err
	}

	// External analyzer path: when --analyzer is set, use the
	// external protocol adapter instead of Go providers.
	// Design decision D12: deferred for `gaze analyze`.
	if p.analyzerFlag != "" {
		return runCrapWithExternalAnalyzer(p)
	}

	// Wire the quality pipeline to provide contract coverage for
	// GazeCRAP scoring via ContractCoverageProvider. This is
	// best-effort: if quality analysis fails for any package,
	// GazeCRAP falls back to unavailable.
	if p.opts.ContractCoverageProvider == nil {
		if p.contractProvider != nil {
			// Test override — use the injected provider.
			p.opts.ContractCoverageProvider = p.contractProvider
		} else {
			// Production path — construct GoContractCoverageProvider.
			var aiMapperFn quality.AIMapperFunc
			if p.aiMapper != "" {
				var aiErr error
				aiMapperFn, aiErr = buildAIMapperFunc(p.aiMapper, p.aiMapperModel)
				if aiErr != nil {
					return aiErr
				}
			}
			p.opts.ContractCoverageProvider = goprovider.NewContractCoverageProvider(
				p.stderr, aiMapperFn,
			)
		}
	}

	logger.Info("computing CRAP scores", "patterns", p.patterns)

	analyze := p.analyzeFunc
	if analyze == nil {
		analyze = crap.Analyze
	}
	rpt, err := analyze(p.patterns, p.moduleDir, p.opts)
	if err != nil {
		return err
	}

	logger.Info("analysis complete", "functions", len(rpt.Scores))

	// Zero-result gate (#116): when threshold flags are set but no
	// functions were analyzed, return an error. A CI gate that passes
	// when nothing was measured provides false assurance — the user
	// likely misconfigured the package pattern. Without thresholds,
	// warn and continue (exploratory/interactive use).
	if len(rpt.Scores) == 0 {
		if p.thresholdSet {
			return fmt.Errorf("no functions analyzed — cannot evaluate thresholds (check package patterns)")
		}
		_, _ = fmt.Fprintln(p.stderr, "warning: no functions analyzed")
	}

	// FR-015: Warn when GazeCRAP is unavailable. GazeCRAP requires
	// contract coverage data from `gaze quality`. If no
	// ContractCoverageFunc was provided, GazeCRAP fields are nil.
	if rpt.Summary.GazeCRAPload == nil {
		_, _ = fmt.Fprintln(p.stderr,
			"note: GazeCRAP unavailable — run 'gaze quality' to compute contract coverage")
	}

	// Resolve baseline path and run comparison (D4).
	comparisonResult, baselineErr := resolveBaselineAndCompare(
		p.baselinePath, p.moduleDir, p.stderr, rpt)
	if baselineErr != nil {
		return baselineErr
	}

	// Write output and CI summary.
	if err := writeCrapOutputAndSummary(
		p.stdout, p.stderr, p.format, rpt, comparisonResult,
		p.maxCrapload, p.maxGazeCrapload); err != nil {
		return err
	}

	// Evaluate gates: baseline regression then CI thresholds (D7).
	return evaluateCrapGates(rpt, comparisonResult, p.stderr,
		p.maxCrapload, p.maxGazeCrapload)
}

// runCrapWithExternalAnalyzer runs the CRAP pipeline using an
// external analyzer binary via the JSON-RPC protocol. The analyzer
// provides complexity, coverage, and optionally contract coverage
// data instead of the Go-specific providers.
//
// Design decision D5: Three-tier discovery (CLI flag → config → PATH).
// Design decision D12: Only crap/quality/report use this path.
func runCrapWithExternalAnalyzer(p crapParams) error {
	session, providers, err := initExternalSession(
		p.analyzerFlag, p.languageFlag, p.moduleDir, p.patterns, p.stderr)
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	wireExternalProviders(&p.opts, providers)

	analyze := p.analyzeFunc
	if analyze == nil {
		analyze = crap.Analyze
	}
	rpt, err := analyze(p.patterns, p.moduleDir, p.opts)
	if err != nil {
		return err
	}

	return finishExternalCrapReport(p, rpt)
}

// finishExternalCrapReport handles post-analysis output and threshold
// checking for external analyzer CRAP results.
func finishExternalCrapReport(p crapParams, rpt *crap.Report) error {
	if len(rpt.Scores) == 0 && p.thresholdSet {
		return fmt.Errorf("no functions analyzed — cannot evaluate thresholds (check package patterns)")
	}
	emitExternalCrapNotes(p.stderr, rpt)

	if err := writeCrapReport(p.stdout, p.format, rpt); err != nil {
		return err
	}

	printCISummary(p.stderr, rpt, p.maxCrapload, p.maxGazeCrapload)
	return checkCIThresholds(rpt, p.maxCrapload, p.maxGazeCrapload)
}

// initExternalSession discovers, spawns, and initializes an external
// analyzer session. Returns the session (caller must Close) and
// the constructed providers.
func initExternalSession(
	analyzerFlag, languageFlag, moduleDir string,
	patterns []string, stderr io.Writer,
) (*adapter.Session, *adapter.Providers, error) {
	cfg := config.LoadFromDir(moduleDir, stderr)
	binary, args, err := adapter.Discover(analyzerFlag, languageFlag, cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("discovering analyzer: %w", err)
	}
	if binary == "" {
		return nil, nil, fmt.Errorf("analyzer %q not found", analyzerFlag)
	}

	session := adapter.NewSession(binary, args, moduleDir, patterns, stderr, cfg)
	providers, err := session.Initialize()
	if err != nil {
		_ = session.Close()
		return nil, nil, fmt.Errorf("initializing analyzer: %w", err)
	}

	_, _ = fmt.Fprintf(stderr, "Using external analyzer: %s (language: %s)\n",
		providers.AnalyzerName, providers.Language)
	return session, providers, nil
}

// wireExternalProviders sets external provider adapters on crap.Options.
func wireExternalProviders(opts *crap.Options, providers *adapter.Providers) {
	opts.ComplexityProvider = providers.Complexity
	opts.LineCoverageProvider = providers.LineCoverage
	if providers.ContractCoverage != nil {
		opts.ContractCoverageProvider = providers.ContractCoverage
	}
}

// emitExternalCrapNotes writes informational notes about external
// analyzer CRAP results to stderr.
func emitExternalCrapNotes(stderr io.Writer, rpt *crap.Report) {
	if len(rpt.Scores) == 0 {
		_, _ = fmt.Fprintln(stderr, "warning: no functions analyzed")
	}
	if rpt.Summary.GazeCRAPload == nil {
		_, _ = fmt.Fprintln(stderr,
			"note: GazeCRAP unavailable — analyzer does not support test_mapping")
	}
}

// writeCrapReport outputs the CRAP report in the requested format.
func writeCrapReport(w io.Writer, format string, rpt *crap.Report) error {
	switch format {
	case "json":
		return crap.WriteJSON(w, rpt)
	default:
		return crap.WriteText(w, rpt)
	}
}

// writeCrapComparisonReport outputs the comparison report in the
// requested format.
func writeCrapComparisonReport(w io.Writer, format string, result *crap.ComparisonResult) error {
	switch format {
	case "json":
		return crap.WriteComparisonJSON(w, result)
	default:
		return crap.WriteComparisonText(w, result)
	}
}

// resolveBaselinePath determines the baseline file path using the
// D4 detection order: explicit flag → config file → default path.
// Returns the path and whether it was explicitly specified (via
// --baseline flag). Empty path means no baseline available.
func resolveBaselinePath(flagPath, moduleDir string, stderr io.Writer) (string, bool) {
	if flagPath != "" {
		return flagPath, true
	}

	// Config file baseline.file setting (non-default only).
	cfg := config.LoadFromDir(moduleDir, stderr)
	if cfg.Baseline.File != "" && cfg.Baseline.File != ".gaze/baseline.json" {
		return resolveConfigBaselinePath(cfg.Baseline.File, moduleDir), false
	}

	// Default .gaze/baseline.json.
	defaultPath := filepath.Join(moduleDir, ".gaze", "baseline.json")
	if isNonEmptyFile(defaultPath) {
		return defaultPath, false
	}
	return "", false
}

// resolveConfigBaselinePath resolves a non-default baseline path from
// .gaze.yaml. Returns empty string if the file doesn't exist or is empty.
func resolveConfigBaselinePath(cfgFile, moduleDir string) string {
	p := cfgFile
	if !filepath.IsAbs(p) {
		p = filepath.Join(moduleDir, p)
	}
	if isNonEmptyFile(p) {
		return p
	}
	return ""
}

// isNonEmptyFile returns true if the path exists and has size > 0.
// Empty files are skipped to handle the shell redirect race where
// the output file is truncated before gaze writes to it.
func isNonEmptyFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Size() > 0
}

// loadAndCompare loads a baseline file and runs comparison against
// the current report. If baselineExplicit is true (--baseline flag),
// errors are fatal. Otherwise, errors are silently skipped.
func loadAndCompare(
	baselinePath string,
	baselineExplicit bool,
	current *crap.Report,
	moduleDir string,
	stderr io.Writer,
) (*crap.ComparisonResult, error) {
	baseline, err := openAndLoadBaseline(baselinePath)
	if err != nil {
		if baselineExplicit {
			return nil, fmt.Errorf("loading baseline %q: %w", baselinePath, err)
		}
		return nil, nil
	}

	cfg := config.LoadFromDir(moduleDir, stderr)
	opts := crap.CompareOptions{
		Epsilon:                      cfg.Baseline.Epsilon,
		NewFunctionThreshold:         cfg.Baseline.NewFunctionThreshold,
		NewFunctionGazeCRAPThreshold: cfg.Baseline.NewFunctionGazeCRAPThreshold,
	}
	return crap.Compare(baseline, current, opts), nil
}

// openAndLoadBaseline opens a baseline file and deserializes it.
func openAndLoadBaseline(path string) (*crap.Report, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()
	return crap.LoadBaseline(f)
}

// autoDetectMainPkg enables unexported function inclusion when the
// package path identifies a main package. This avoids requiring users
// to pass --include-unexported explicitly for package main targets.
func autoDetectMainPkg(pkgPath string, includeUnexported *bool) {
	if !*includeUnexported && loader.IsMainPkg(pkgPath) {
		*includeUnexported = true
		logger.Info("package main detected, including unexported functions", "pkg", pkgPath)
	}
}

// printCISummary prints a one-line CI summary to stderr when
// threshold flags are set.
func printCISummary(w io.Writer, rpt *crap.Report, maxCrapload, maxGazeCrapload int) {
	if maxCrapload <= 0 && maxGazeCrapload <= 0 {
		return
	}

	var parts []string
	if maxCrapload > 0 {
		status := "PASS"
		if rpt.Summary.CRAPload > maxCrapload {
			status = "FAIL"
		}
		parts = append(parts, fmt.Sprintf("CRAPload: %d/%d (%s)",
			rpt.Summary.CRAPload, maxCrapload, status))
	}
	if maxGazeCrapload > 0 && rpt.Summary.GazeCRAPload != nil {
		status := "PASS"
		if *rpt.Summary.GazeCRAPload > maxGazeCrapload {
			status = "FAIL"
		}
		parts = append(parts, fmt.Sprintf("GazeCRAPload: %d/%d (%s)",
			*rpt.Summary.GazeCRAPload, maxGazeCrapload, status))
	}
	_, _ = fmt.Fprintln(w, strings.Join(parts, " | "))
}

// checkCIThresholds returns an error if any CI thresholds are exceeded.
func checkCIThresholds(rpt *crap.Report, maxCrapload, maxGazeCrapload int) error {
	if maxCrapload > 0 && rpt.Summary.CRAPload > maxCrapload {
		return fmt.Errorf("CRAPload %d exceeds maximum %d",
			rpt.Summary.CRAPload, maxCrapload)
	}
	// When GazeCRAPload is nil, skip the check silently. gaze crap
	// prints a "GazeCRAP unavailable" note separately (line ~502).
	// This differs from gaze report's EvaluateThresholds which fails
	// when the metric is unavailable — see #108.
	if maxGazeCrapload > 0 && rpt.Summary.GazeCRAPload != nil &&
		*rpt.Summary.GazeCRAPload > maxGazeCrapload {
		return fmt.Errorf("GazeCRAPload %d exceeds maximum %d",
			*rpt.Summary.GazeCRAPload, maxGazeCrapload)
	}
	return nil
}

// resolveBaselineAndCompare resolves the baseline file path using
// resolveBaselinePath (D4 detection order: flag → config → default),
// then loads the baseline and compares it against the current report.
// Returns nil, nil if no baseline is configured.
func resolveBaselineAndCompare(
	baselinePath, moduleDir string,
	stderr io.Writer,
	rpt *crap.Report,
) (*crap.ComparisonResult, error) {
	resolved, explicit := resolveBaselinePath(baselinePath, moduleDir, stderr)
	if resolved == "" {
		return nil, nil
	}
	return loadAndCompare(resolved, explicit, rpt, moduleDir, stderr)
}

// writeCrapOutputAndSummary writes the CRAP report (comparison path or
// normal path) to stdout, then prints the CI summary line to stderr.
// The maxCrapload and maxGazeCrapload params are forwarded to
// printCISummary for display.
func writeCrapOutputAndSummary(
	stdout, stderr io.Writer,
	format string,
	rpt *crap.Report,
	cr *crap.ComparisonResult,
	maxCrapload, maxGazeCrapload int,
) error {
	if cr != nil {
		if err := writeCrapComparisonReport(stdout, format, cr); err != nil {
			return err
		}
	} else {
		if err := writeCrapReport(stdout, format, rpt); err != nil {
			return err
		}
	}
	printCISummary(stderr, rpt, maxCrapload, maxGazeCrapload)
	return nil
}

// evaluateCrapGates evaluates CI gates in order: baseline regression
// gate first, then CI threshold gate (D7 ordering). This ensures
// comparison output is always visible before a threshold failure.
// If the baseline comparison exists and failed, the error is returned
// immediately — the threshold gate is not reached.
func evaluateCrapGates(
	rpt *crap.Report,
	cr *crap.ComparisonResult,
	stderr io.Writer,
	maxCrapload, maxGazeCrapload int,
) error {
	// Baseline gate: evaluate first so comparison output is visible (D7).
	if cr != nil && !cr.Summary.Passed {
		_, _ = fmt.Fprintf(stderr, "baseline comparison: FAIL (%d regressions, %d new violations)\n",
			cr.Summary.Regressions,
			cr.Summary.NewViolations)
		return fmt.Errorf("baseline comparison failed: %d regressions, %d new-function violations",
			cr.Summary.Regressions,
			cr.Summary.NewViolations)
	}

	// Threshold gate: evaluate only if baseline gate passed.
	return checkCIThresholds(rpt, maxCrapload, maxGazeCrapload)
}

func newCrapCmd() *cobra.Command {
	var (
		format            string
		coverProfile      string
		crapThreshold     float64
		gazeCrapThreshold float64
		maxCrapload       int
		maxGazeCrapload   int
		aiMapper          string
		aiMapperModel     string
		baselinePath      string
		analyzerFlag      string
		languageFlag      string
		testShort         bool
	)

	cmd := &cobra.Command{
		Use:   "crap [packages...]",
		Short: "Compute CRAP scores for Go functions",
		Long: `Compute CRAP (Change Risk Anti-Patterns) scores by combining
cyclomatic complexity with test coverage. Reports per-function
CRAP scores and the project's CRAPload (count of functions above
the threshold).

If no coverage profile is provided, runs 'go test -coverprofile'
automatically.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
			moduleDir, err := loader.FindModuleRoot(cwd)
			if err != nil {
				return fmt.Errorf("finding module root: %w", err)
			}
			opts := crap.DefaultOptions()
			opts.CoverProfile = coverProfile
			opts.CRAPThreshold = crapThreshold
			opts.GazeCRAPThreshold = gazeCrapThreshold
			opts.Stderr = os.Stderr
			opts.ComplexityProvider = goprovider.NewComplexityProvider()
			lineProv := goprovider.NewLineCoverageProvider(os.Stderr)
			lineProv.Short = testShort
			opts.LineCoverageProvider = lineProv
			return runCrap(crapParams{
				patterns:        args,
				format:          format,
				opts:            opts,
				maxCrapload:     maxCrapload,
				maxGazeCrapload: maxGazeCrapload,
				moduleDir:       moduleDir,
				aiMapper:        aiMapper,
				aiMapperModel:   aiMapperModel,
				baselinePath:    baselinePath,
				analyzerFlag:    analyzerFlag,
				languageFlag:    languageFlag,
				stdout:          os.Stdout,
				stderr:          os.Stderr,
				thresholdSet:    cmd.Flags().Changed("max-crapload") || cmd.Flags().Changed("max-gaze-crapload"),
			})
		},
	}

	cmd.Flags().StringVar(&format, "format", "text",
		"output format: text or json")
	cmd.Flags().StringVar(&coverProfile, "coverprofile", "",
		"path to coverage profile (default: generate via go test)")
	cmd.Flags().Float64Var(&crapThreshold, "crap-threshold", 15,
		"CRAP score threshold for flagging functions")
	cmd.Flags().Float64Var(&gazeCrapThreshold, "gaze-crap-threshold", 15,
		"GazeCRAP score threshold (used when contract coverage available)")
	cmd.Flags().IntVar(&maxCrapload, "max-crapload", 0,
		"fail if CRAPload exceeds this (0 = no limit)")
	cmd.Flags().IntVar(&maxGazeCrapload, "max-gaze-crapload", 0,
		"fail if GazeCRAPload exceeds this (0 = no limit)")
	cmd.Flags().StringVar(&aiMapper, "ai-mapper", "",
		"AI backend for assertion mapping fallback: claude, gemini, ollama, or opencode")
	cmd.Flags().StringVar(&aiMapperModel, "ai-mapper-model", "",
		"model name for AI mapper (required for ollama)")
	cmd.Flags().StringVar(&baselinePath, "baseline", "",
		"path to baseline file for comparison")
	cmd.Flags().StringVar(&analyzerFlag, "analyzer", "",
		"external analyzer binary (e.g., snake-eyes)")
	cmd.Flags().StringVar(&languageFlag, "language", "",
		"target language for analyzer discovery (e.g., python)")
	cmd.Flags().BoolVar(&testShort, "test-short", false,
		"pass -short to internal go test invocation (faster, less accurate coverage)")

	return cmd
}

// docscanParams holds the parsed flags for the docscan command.
type docscanParams struct {
	pkgPath    string
	configPath string
	stdout     io.Writer
	stderr     io.Writer
}

// runDocscan is the extracted, testable body of the docscan command.
func runDocscan(p docscanParams) error {
	cfg, err := loadConfig(p.configPath, -1, -1)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Determine the repo root: walk up from the package directory
	// to find the go.mod file, defaulting to cwd.
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	repoRoot := cwd
	if root, findErr := loader.FindModuleRoot(cwd); findErr == nil {
		repoRoot = root
	}

	// Resolve PackageDir from the import path if it corresponds
	// to a local path pattern, otherwise use the repo root.
	pkgDir := ""
	if strings.HasPrefix(p.pkgPath, "./") || strings.HasPrefix(p.pkgPath, "../") {
		abs, absErr := filepath.Abs(p.pkgPath)
		if absErr == nil {
			pkgDir = abs
		}
	}

	scanOpts := docscan.ScanOptions{
		Config:     cfg,
		PackageDir: pkgDir,
	}

	docs, err := docscan.Scan(repoRoot, scanOpts)
	if err != nil {
		return fmt.Errorf("scanning documents: %w", err)
	}

	enc := json.NewEncoder(p.stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(docs)
}

func newDocscanCmd() *cobra.Command {
	var configPath string

	cmd := &cobra.Command{
		Use:   "docscan [package]",
		Short: "Scan project documentation for classification signals",
		Long: `Scan the repository for Markdown documentation files and
output a prioritized list of documents as JSON. Useful as input
to the gaze-reporter agent's full mode for document-enhanced
classification.

Priority:
  1 = same directory as the target package (highest relevance)
  2 = module root
  3 = other locations`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			pkgPath := "."
			if len(args) > 0 {
				pkgPath = args[0]
			}
			return runDocscan(docscanParams{
				pkgPath:    pkgPath,
				configPath: configPath,
				stdout:     os.Stdout,
				stderr:     os.Stderr,
			})
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "",
		"path to .gaze.yaml config file (default: search CWD)")

	return cmd
}

// qualityParams holds the parsed flags for the quality command.
type qualityParams struct {
	patterns             []string
	format               string
	targetFunc           string
	verbose              bool
	includeUnexported    bool
	configPath           string
	contractualThresh    int
	incidentalThresh     int
	minContractCoverage  int
	maxOverSpecification int
	aiMapper             string
	aiMapperModel        string
	analyzerFlag         string
	languageFlag         string
	stdout               io.Writer
	stderr               io.Writer
}

// runQuality is the extracted, testable body of the quality command.
func runQuality(p qualityParams) error {
	if err := cliutil.ValidateFormat(p.format); err != nil {
		return err
	}

	// External analyzer path: delegate to the external analyzer pipeline
	// which bypasses Go-specific test loading and assertion mapping.
	if p.analyzerFlag != "" {
		return runQualityWithExternalAnalyzer(p)
	}

	// Resolve package patterns to concrete package paths.
	moduleDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	pkgPaths, err := loader.ResolvePackagePaths(p.patterns, moduleDir, p.stderr)
	if err != nil {
		return fmt.Errorf("resolving package patterns: %w", err)
	}
	if len(pkgPaths) == 0 {
		return fmt.Errorf("no packages found for patterns %v", p.patterns)
	}

	cfg, cfgErr := loadQualityConfig(p)
	if cfgErr != nil {
		return cfgErr
	}

	modPkgs, aiMapperFn, setupErr := setupQualityDeps(p, moduleDir)
	if setupErr != nil {
		return setupErr
	}

	var allReports []taxonomy.QualityReport
	var allSummaries []*taxonomy.PackageSummary

	for _, pkgPath := range pkgPaths {
		opts := analysis.Options{
			IncludeUnexported: p.includeUnexported,
			Version:           version,
		}

		autoDetectMainPkg(pkgPath, &opts.IncludeUnexported)

		reports, summary, perPkgErr := runQualityPerPackage(
			pkgPath, p, opts, cfg, modPkgs, aiMapperFn)
		if perPkgErr != nil {
			return perPkgErr
		}
		if reports == nil && summary == nil {
			// Graceful skip: no tests or no analyzable functions.
			continue
		}

		allReports = append(allReports, reports...)
		if summary != nil {
			allSummaries = append(allSummaries, summary)
		}
	}

	// Merge summaries into a single aggregate summary.
	// This must happen before the empty-result check so that
	// skipped test data is available for the summary output.
	merged := mergeSummaries(allSummaries)

	if len(allReports) == 0 {
		return handleQualityEmptyResults(p, merged)
	}

	// Write report and check CI thresholds.
	return writeQualityReport(p, allReports, merged)
}

// runQualityWithExternalAnalyzer runs the quality pipeline using an
// external analyzer binary via the JSON-RPC protocol. The analyzer
// provides side effect analysis and test_mapping data instead of the
// Go-specific quality.Assess pipeline.
//
// Design decisions D6/D7: --target and --ai-mapper are rejected because
// they depend on Go-specific SSA target inference and AST assertion
// detection that external analyzers cannot provide.
func runQualityWithExternalAnalyzer(p qualityParams) error {
	// Validate flag combinations: --target and --ai-mapper are
	// Go-specific features incompatible with external analyzers.
	if p.targetFunc != "" {
		return fmt.Errorf("--target is not supported with --analyzer; " +
			"the external analyzer provides its own test-to-target mapping")
	}
	if p.aiMapper != "" {
		return fmt.Errorf("--ai-mapper is not supported with --analyzer; " +
			"assertion mapping is provided by the external analyzer")
	}

	moduleDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	session, providers, err := initExternalSession(
		p.analyzerFlag, p.languageFlag, moduleDir, p.patterns, p.stderr)
	if err != nil {
		return err
	}
	defer func() { _ = session.Close() }()

	// Graceful degradation: when test_mapping is not supported,
	// produce a zero-coverage report with a warning instead of failing.
	if !providers.Capabilities.TestMapping {
		return handleQualityNoTestMapping(p, providers)
	}

	// Fetch side effect analysis results.
	results, resultsErr := providers.SideEffects.AllResults()
	if resultsErr != nil {
		return fmt.Errorf("fetching side effect analysis: %w", resultsErr)
	}

	// Fetch test mapping data.
	mappings, fetchErr := adapter.FetchTestMappings(
		session.Client(), p.patterns, moduleDir)
	if fetchErr != nil {
		// Graceful degradation on test_mapping method failure:
		// produce a zero-coverage report with reason.
		return handleQualityTestMappingError(p, providers, fetchErr)
	}

	// Build quality reports from external data.
	reports, summary := adapter.BuildQualityFromMappings(mappings, results)

	if len(reports) == 0 {
		return handleQualityEmptyResults(p, summary)
	}

	return writeQualityReport(p, reports, summary)
}

// handleQualityNoTestMapping produces output when the external analyzer
// does not support the test_mapping capability. Prints a warning and
// either exits 0 (no thresholds) or returns an error (thresholds set).
func handleQualityNoTestMapping(p qualityParams, providers *adapter.Providers) error {
	_, _ = fmt.Fprintf(p.stderr,
		"warning: analyzer %q does not support test_mapping — "+
			"contract coverage and over-specification metrics are unavailable\n",
		providers.AnalyzerName)

	summary := &taxonomy.PackageSummary{
		Reason: "test_mapping unavailable",
	}
	if err := writeQualityEmptyOutput(p, summary); err != nil {
		return err
	}

	if p.minContractCoverage > 0 || p.maxOverSpecification > 0 {
		return fmt.Errorf("quality thresholds cannot be evaluated — " +
			"analyzer does not support test_mapping")
	}
	return nil
}

// handleQualityTestMappingError produces output when the test_mapping
// protocol method fails at runtime. The capability was declared but the
// method returned an error.
func handleQualityTestMappingError(p qualityParams, providers *adapter.Providers, fetchErr error) error {
	_, _ = fmt.Fprintf(p.stderr,
		"warning: test_mapping failed for analyzer %q: %v\n",
		providers.AnalyzerName, fetchErr)

	summary := &taxonomy.PackageSummary{
		Reason: fmt.Sprintf("test_mapping error: %v", fetchErr),
	}
	if err := writeQualityEmptyOutput(p, summary); err != nil {
		return err
	}

	if p.minContractCoverage > 0 || p.maxOverSpecification > 0 {
		return fmt.Errorf("quality thresholds cannot be evaluated — "+
			"test_mapping failed: %w", fetchErr)
	}
	return nil
}

// writeQualityEmptyOutput writes an empty quality report in the
// requested format. Used by degraded/error paths.
func writeQualityEmptyOutput(p qualityParams, summary *taxonomy.PackageSummary) error {
	return writeQualityEmptyResults(p.stdout, p.format, summary)
}

// mergeSummaries combines multiple PackageSummary values into one.
// Coverage is averaged, counts are summed.
func mergeSummaries(summaries []*taxonomy.PackageSummary) *taxonomy.PackageSummary {
	if len(summaries) == 0 {
		return &taxonomy.PackageSummary{}
	}
	if len(summaries) == 1 {
		return summaries[0]
	}

	merged := &taxonomy.PackageSummary{}
	var totalCoverage float64
	var totalDetectionConf int
	var allWorst []taxonomy.QualityReport
	for _, s := range summaries {
		merged.TotalTests += s.TotalTests
		merged.TotalOverSpecifications += s.TotalOverSpecifications
		totalCoverage += s.AverageContractCoverage
		totalDetectionConf += s.AssertionDetectionConfidence
		allWorst = append(allWorst, s.WorstCoverageTests...)
		merged.SSADegraded = merged.SSADegraded || s.SSADegraded
		merged.SSADegradedPackages = append(merged.SSADegradedPackages, s.SSADegradedPackages...)
		merged.SkippedTests += s.SkippedTests
		merged.SkippedTestNames = append(merged.SkippedTestNames, s.SkippedTestNames...)
	}
	n := float64(len(summaries))
	merged.AverageContractCoverage = totalCoverage / n
	merged.AssertionDetectionConfidence = int(float64(totalDetectionConf)/n + 0.5)

	// Re-sort combined worst tests by coverage ascending, truncate to 5.
	sort.SliceStable(allWorst, func(i, j int) bool {
		if allWorst[i].ContractCoverage.Percentage != allWorst[j].ContractCoverage.Percentage {
			return allWorst[i].ContractCoverage.Percentage < allWorst[j].ContractCoverage.Percentage
		}
		return allWorst[i].TestFunction < allWorst[j].TestFunction
	})
	if len(allWorst) > 5 {
		allWorst = allWorst[:5]
	}
	merged.WorstCoverageTests = allWorst

	return merged
}

// loadTestPackage loads a Go package with test files included.
func loadTestPackage(pkgPath string) (*packages.Package, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Tests: true,
	}
	pkgs, err := packages.Load(cfg, pkgPath)
	if err != nil {
		return nil, fmt.Errorf("loading test package: %w", err)
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages found for %q", pkgPath)
	}

	// Check for package load errors.
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			msgs := make([]string, len(pkg.Errors))
			for i, e := range pkg.Errors {
				msgs[i] = e.Error()
			}
			return nil, fmt.Errorf("package %s has errors: %s",
				pkg.PkgPath, strings.Join(msgs, "; "))
		}
	}

	// When Tests=true, packages.Load returns multiple packages:
	// the base package, the internal test package (same name, with
	// test files merged), and possibly an external test package
	// (with _test suffix). Prefer the package that contains test
	// function declarations in its syntax.
	for _, pkg := range pkgs {
		if quality.HasTestSyntax(pkg) {
			return pkg, nil
		}
	}

	// No package has test syntax — return an error rather than
	// silently returning a non-test package that would produce
	// empty quality results.
	return nil, fmt.Errorf("no test package found for %q — does the package have *_test.go files?", pkgPath)
}

// loadQualityConfig normalizes threshold values (0 → -1 for "unset")
// and loads the classify config from the config file or defaults.
func loadQualityConfig(p qualityParams) (*config.GazeConfig, error) {
	contractualThresh := p.contractualThresh
	if contractualThresh == 0 {
		contractualThresh = -1
	}
	incidentalThresh := p.incidentalThresh
	if incidentalThresh == 0 {
		incidentalThresh = -1
	}
	cfg, err := loadConfig(p.configPath, contractualThresh, incidentalThresh)
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}
	return cfg, nil
}

// setupQualityDeps loads module packages for classification and wires
// the AI-assisted assertion mapping callback. Module loading failure
// is non-fatal (degraded mode). Returns the module packages (may be
// nil on failure), the AI mapper function (nil when --ai-mapper is
// not set), and an error only when AI mapper setup fails.
func setupQualityDeps(p qualityParams, moduleDir string) ([]*packages.Package, quality.AIMapperFunc, error) {
	logger.Info("loading module packages for classification")
	modResult, modErr := loader.LoadModule(moduleDir)
	var modPkgs []*packages.Package
	if modErr != nil {
		logger.Warn("module loading failed; caller/interface signals degraded", "err", modErr)
	} else {
		modPkgs = modResult.Packages
	}

	var aiMapperFn quality.AIMapperFunc
	if p.aiMapper != "" {
		var aiErr error
		aiMapperFn, aiErr = buildAIMapperFunc(p.aiMapper, p.aiMapperModel)
		if aiErr != nil {
			return nil, nil, aiErr
		}
	}

	return modPkgs, aiMapperFn, nil
}

// handleQualityEmptyResults writes the empty-result output and
// enforces the zero-result threshold gate. Returns an error when
// thresholds are set but no test-target pairs were resolved.
func handleQualityEmptyResults(p qualityParams, merged *taxonomy.PackageSummary) error {
	if err := writeQualityEmptyResults(p.stdout, p.format, merged); err != nil {
		return err
	}
	if p.minContractCoverage > 0 || p.maxOverSpecification > 0 {
		return fmt.Errorf("no test-target pairs resolved — cannot evaluate thresholds "+
			"(--min-contract-coverage=%d, --max-over-specification=%d)",
			p.minContractCoverage, p.maxOverSpecification)
	}
	return nil
}

// writeQualityReport writes the quality report output (JSON or text)
// and then checks CI thresholds. Returns an error if writing fails
// or if a threshold is violated.
func writeQualityReport(p qualityParams, reports []taxonomy.QualityReport, summary *taxonomy.PackageSummary) error {
	switch p.format {
	case "json":
		if err := quality.WriteJSON(p.stdout, reports, summary); err != nil {
			return fmt.Errorf("writing quality JSON: %w", err)
		}
	default:
		if err := quality.WriteText(p.stdout, reports, summary); err != nil {
			return fmt.Errorf("writing quality text report: %w", err)
		}
	}
	return checkQualityThresholds(p, reports, summary)
}

// runQualityPerPackage runs the quality analysis pipeline for a single
// package: analyze side effects, classify, load test package, and
// assess quality. Returns nil, nil, nil when the package has no tests
// or no analyzable functions (graceful skip). Returns a non-nil error
// only for genuine failures (analysis errors, classification errors,
// assessment errors).
func runQualityPerPackage(
	pkgPath string,
	p qualityParams,
	opts analysis.Options,
	cfg *config.GazeConfig,
	modPkgs []*packages.Package,
	aiMapperFn quality.AIMapperFunc,
) ([]taxonomy.QualityReport, *taxonomy.PackageSummary, error) {
	logger.Info("analyzing package", "pkg", pkgPath)
	results, loadErr := analysis.LoadAndAnalyze(pkgPath, opts)
	if loadErr != nil {
		return nil, nil, loadErr
	}
	if len(results) == 0 {
		logger.Warn("no functions found to analyze", "pkg", pkgPath)
		return nil, nil, nil
	}

	// Classify side effects.
	var err error
	results, err = runClassify(results, pkgPath, cfg, p.verbose, modPkgs)
	if err != nil {
		return nil, nil, fmt.Errorf("classification of %s: %w", pkgPath, err)
	}

	// Load the test package with test files.
	testPkg, testErr := loadTestPackage(pkgPath)
	if testErr != nil {
		// Skip packages without tests gracefully.
		logger.Warn("skipping package without tests", "pkg", pkgPath, "err", testErr)
		return nil, nil, nil
	}

	// Assess test quality.
	qualOpts := quality.Options{
		TargetFunc: p.targetFunc,
		Verbose:    p.verbose,
		Version:    version,
		Stderr:     p.stderr,
	}
	if aiMapperFn != nil {
		qualOpts.AIMapperFunc = aiMapperFn
	}

	reports, summary, assessErr := quality.Assess(results, testPkg, qualOpts)
	if assessErr != nil {
		return nil, nil, fmt.Errorf("quality assessment of %s: %w", pkgPath, assessErr)
	}

	return reports, summary, nil
}

// writeQualityEmptyResults writes the empty-result output when no
// test-target pairs were resolved. For JSON format, it writes a valid
// JSON object with an empty quality_reports array. For text format, it
// prints a summary line, skipped test names (truncated at
// MaxSkippedTestDisplay), and a --target hint.
// This function does NOT evaluate thresholds — that is the caller's
// responsibility.
func writeQualityEmptyResults(w io.Writer, format string, merged *taxonomy.PackageSummary) error {
	maxSkippedTestDisplay := quality.MaxSkippedTestDisplay

	switch format {
	case "json":
		// Produce valid JSON even when no reports exist.
		// Use a non-nil empty slice so JSON encodes as [] not null.
		emptyReports := make([]taxonomy.QualityReport, 0)
		if err := quality.WriteJSON(w, emptyReports, merged); err != nil {
			return fmt.Errorf("writing empty quality JSON: %w", err)
		}
	default:
		totalTestFuncs := merged.TotalTests + merged.SkippedTests
		_, _ = fmt.Fprintf(w, "Quality: 0 of %d test functions mapped to a target\n", totalTestFuncs)
		if merged.SkippedTests > 0 {
			_, _ = fmt.Fprintf(w, "\nSkipped test functions (%d):\n", merged.SkippedTests)
			limit := merged.SkippedTests
			if limit > maxSkippedTestDisplay {
				limit = maxSkippedTestDisplay
			}
			if limit > len(merged.SkippedTestNames) {
				limit = len(merged.SkippedTestNames)
			}
			for _, name := range merged.SkippedTestNames[:limit] {
				_, _ = fmt.Fprintf(w, "  - %s\n", name)
			}
			if merged.SkippedTests > maxSkippedTestDisplay {
				_, _ = fmt.Fprintf(w, "  ... and %d more\n", merged.SkippedTests-maxSkippedTestDisplay)
			}
			_, _ = fmt.Fprintf(w, "\nHint: use --target=FuncName to specify the target explicitly\n")
		}
	}
	return nil
}

// checkQualityThresholds enforces CI threshold flags on quality
// metrics. Per the spec (FR-006), thresholds apply to individual
// test-target pairs, not the package average.
func checkQualityThresholds(
	p qualityParams,
	reports []taxonomy.QualityReport,
	summary *taxonomy.PackageSummary,
) error {
	if p.minContractCoverage <= 0 && p.maxOverSpecification <= 0 {
		return nil
	}

	// Skip threshold enforcement on degraded results — SSA failure
	// produces zero-valued coverage and over-specification metrics
	// that would trigger false-positive CI failures.
	if summary != nil && summary.SSADegraded {
		if p.stderr != nil {
			_, _ = fmt.Fprintln(p.stderr,
				"warning: CI thresholds skipped — SSA construction failed, quality metrics are partial")
		}
		return nil
	}

	// Print CI summary to stderr.
	var parts []string
	var failures []string

	// Per-test contract coverage check.
	if p.minContractCoverage > 0 {
		allPass := true
		for _, r := range reports {
			if r.ContractCoverage.Percentage < float64(p.minContractCoverage) {
				allPass = false
				failures = append(failures, fmt.Sprintf(
					"%s: contract coverage %.0f%% is below minimum %d%%",
					r.TestFunction, r.ContractCoverage.Percentage, p.minContractCoverage))
			}
		}
		status := "PASS"
		if !allPass {
			status = "FAIL"
		}
		avg := 0.0
		if summary != nil {
			avg = summary.AverageContractCoverage
		}
		parts = append(parts, fmt.Sprintf("Contract Coverage: %.0f%% avg, min %d%% (%s)",
			avg, p.minContractCoverage, status))
	}

	// Per-test over-specification check (consistent with per-test
	// contract coverage check above; FR-006).
	if p.maxOverSpecification > 0 {
		allPass := true
		for _, r := range reports {
			if r.OverSpecification.Count > p.maxOverSpecification {
				allPass = false
				failures = append(failures, fmt.Sprintf(
					"%s: over-specification count %d exceeds maximum %d",
					r.TestFunction, r.OverSpecification.Count, p.maxOverSpecification))
			}
		}
		status := "PASS"
		if !allPass {
			status = "FAIL"
		}
		total := 0
		if summary != nil {
			total = summary.TotalOverSpecifications
		}
		parts = append(parts, fmt.Sprintf("Over-Specifications: %d total, max %d per test (%s)",
			total, p.maxOverSpecification, status))
	}

	if len(parts) > 0 {
		_, _ = fmt.Fprintln(p.stderr, strings.Join(parts, " | "))
	}

	// Return all failures so users see every violation at once,
	// rather than fixing one at a time (Actionable Output principle).
	if len(failures) > 0 {
		return errors.New(strings.Join(failures, "\n"))
	}

	return nil
}

func newQualityCmd() *cobra.Command {
	var (
		format               string
		targetFunc           string
		verbose              bool
		includeUnexported    bool
		configPath           string
		contractualThresh    int
		incidentalThresh     int
		minContractCoverage  int
		maxOverSpecification int
		aiMapper             string
		aiMapperModel        string
		analyzerFlag         string
		languageFlag         string
	)

	cmd := &cobra.Command{
		Use:   "quality [packages...]",
		Short: "Assess test quality via side effect mapping",
		Long: `Analyze how well one or more packages' tests assert on the
contractual side effects of the functions they test. Reports
Contract Coverage (ratio of contractual effects that are asserted
on) and Over-Specification Score (assertions on incidental
implementation details). Accepts multiple package patterns
including ./... wildcards.

Packages without test files are skipped with a warning.`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runQuality(qualityParams{
				patterns:             args,
				format:               format,
				targetFunc:           targetFunc,
				verbose:              verbose,
				includeUnexported:    includeUnexported,
				configPath:           configPath,
				contractualThresh:    contractualThresh,
				incidentalThresh:     incidentalThresh,
				minContractCoverage:  minContractCoverage,
				maxOverSpecification: maxOverSpecification,
				aiMapper:             aiMapper,
				aiMapperModel:        aiMapperModel,
				analyzerFlag:         analyzerFlag,
				languageFlag:         languageFlag,
				stdout:               os.Stdout,
				stderr:               os.Stderr,
			})
		},
	}

	cmd.Flags().StringVar(&format, "format", "text",
		"output format: text or json")
	cmd.Flags().StringVar(&targetFunc, "target", "",
		"restrict analysis to tests that exercise this function")
	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"show detailed assertion and mapping information")
	cmd.Flags().BoolVar(&includeUnexported, "include-unexported", false,
		"include unexported functions")
	cmd.Flags().StringVar(&configPath, "config", "",
		"path to .gaze.yaml config file (default: search CWD)")
	cmd.Flags().IntVar(&contractualThresh, "contractual-threshold", -1,
		"override contractual confidence threshold (default: from config or 80)")
	cmd.Flags().IntVar(&incidentalThresh, "incidental-threshold", -1,
		"override incidental confidence threshold (default: from config or 50)")
	cmd.Flags().IntVar(&minContractCoverage, "min-contract-coverage", 0,
		"fail if contract coverage is below this percentage (0 = no limit)")
	cmd.Flags().IntVar(&maxOverSpecification, "max-over-specification", 0,
		"fail if over-specification count exceeds this (0 = no limit)")
	cmd.Flags().StringVar(&aiMapper, "ai-mapper", "",
		"AI backend for assertion mapping fallback: claude, gemini, ollama, or opencode")
	cmd.Flags().StringVar(&aiMapperModel, "ai-mapper-model", "",
		"model name for AI mapper (required for ollama)")
	cmd.Flags().StringVar(&analyzerFlag, "analyzer", "",
		"external analyzer binary (e.g., snake-eyes)")
	cmd.Flags().StringVar(&languageFlag, "language", "",
		"target language for analyzer discovery (e.g., python)")
	// Analyzer flags are now visible — D12 deferral lifted.

	return cmd
}

// selfCheckParams holds the parsed flags for the self-check command.
type selfCheckParams struct {
	format          string
	maxCrapload     int
	maxGazeCrapload int
	// testShort passes -short to the internal go test invocation when true.
	testShort bool
	stdout    io.Writer
	stderr    io.Writer

	// thresholdSet is true when any threshold flag was explicitly
	// provided on the command line (via cmd.Flags().Changed). Passed
	// through to crapParams for the zero-result gate (#116).
	thresholdSet bool

	// moduleRootFunc overrides findModuleRoot for testing.
	// When nil, the production findModuleRoot is called.
	moduleRootFunc func() (string, error)

	// runCrapFunc overrides the internal call to runCrap for testing.
	// When nil, runCrap is called directly with the constructed params.
	runCrapFunc func(crapParams) error
}

// runSelfCheck runs the CRAP pipeline on Gaze's own source code.
// It reports CRAPload and worst offenders by CRAP score. GazeCRAP
// is included when contract coverage data is available from the
// quality pipeline. This serves as both a dogfooding exercise and
// a code quality gate.
func runSelfCheck(p selfCheckParams) error {
	if err := cliutil.ValidateFormat(p.format); err != nil {
		return err
	}

	findRoot := p.moduleRootFunc
	if findRoot == nil {
		findRoot = func() (string, error) {
			cwd, err := os.Getwd()
			if err != nil {
				return "", fmt.Errorf("getting working directory: %w", err)
			}
			return loader.FindModuleRoot(cwd)
		}
	}
	moduleDir, err := findRoot()
	if err != nil {
		return fmt.Errorf("finding module root: %w", err)
	}

	selfOpts := crap.DefaultOptions()
	selfOpts.Stderr = p.stderr
	selfOpts.ComplexityProvider = goprovider.NewComplexityProvider()
	lineProv := goprovider.NewLineCoverageProvider(p.stderr)
	lineProv.Short = p.testShort
	selfOpts.LineCoverageProvider = lineProv

	cp := crapParams{
		patterns:        []string{"./..."},
		format:          p.format,
		opts:            selfOpts,
		maxCrapload:     p.maxCrapload,
		maxGazeCrapload: p.maxGazeCrapload,
		moduleDir:       moduleDir,
		stdout:          p.stdout,
		stderr:          p.stderr,
		thresholdSet:    p.thresholdSet,
	}

	doCrap := p.runCrapFunc
	if doCrap == nil {
		doCrap = runCrap
	}
	return doCrap(cp)
}

func newSelfCheckCmd() *cobra.Command {
	var (
		format          string
		maxCrapload     int
		maxGazeCrapload int
		testShort       bool
	)

	cmd := &cobra.Command{
		Use:   "self-check",
		Short: "Run CRAP analysis on Gaze's own source code",
		Long: `Analyze Gaze's own source code for CRAP scores, serving as
both a dogfooding exercise and a code quality gate. Reports
CRAPload and the worst offenders by CRAP score. GazeCRAP
scores are included when contract coverage data is available
(requires integration with the quality pipeline).`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return runSelfCheck(selfCheckParams{
				format:          format,
				maxCrapload:     maxCrapload,
				maxGazeCrapload: maxGazeCrapload,
				testShort:       testShort,
				stdout:          os.Stdout,
				stderr:          os.Stderr,
				thresholdSet:    cmd.Flags().Changed("max-crapload") || cmd.Flags().Changed("max-gaze-crapload"),
			})
		},
	}

	cmd.Flags().StringVar(&format, "format", "text",
		"output format: text or json")
	cmd.Flags().IntVar(&maxCrapload, "max-crapload", 0,
		"fail if CRAPload exceeds this count (0 = no limit)")
	cmd.Flags().IntVar(&maxGazeCrapload, "max-gaze-crapload", 0,
		"fail if GazeCRAPload exceeds this count (0 = no limit)")
	cmd.Flags().BoolVar(&testShort, "test-short", true,
		"pass -short to internal go test invocation (default true for self-check to avoid timeouts)")

	return cmd
}

// reportParams holds the parsed flags for the report command.
// Follows the existing testable CLI pattern (see crapParams, qualityParams).
type reportParams struct {
	patterns    []string
	format      string
	adapterName string
	modelName   string
	aiTimeout   time.Duration
	// Threshold flags use *int: nil = not provided, non-nil (including *0) = active threshold.
	maxCrapload         *int
	maxGazeCrapload     *int
	minContractCoverage *int
	coverProfile        string
	analyzerFlag        string
	languageFlag        string
	// testShort passes -short to internal go test invocations when true.
	testShort bool
	stdout    io.Writer
	stderr    io.Writer

	// runnerFunc overrides aireport.Run for testing. When nil, aireport.Run is called.
	runnerFunc func(aireport.RunnerOptions) error
}

// runReport is the extracted, testable body of the report command.
//
// In text mode it validates the --ai flag, resolves the adapter, loads the
// system prompt, and calls the 4-step analysis pipeline via aireport.Run.
// In json mode it skips AI adapter validation entirely (FR-015).
// Threshold evaluation runs after the pipeline and may set exit code 1.
// validateReportParams checks pre-flight conditions for gaze report:
// adapter requirement in text mode, ollama model requirement, and
// coverprofile path validity.
func validateReportParams(p reportParams) error {
	// In text mode, --ai is required (FR-002).
	if p.format != "json" && p.adapterName == "" {
		return fmt.Errorf(
			"--ai is required in text mode: must be one of \"claude\", \"gemini\", \"ollama\", or \"opencode\"",
		)
	}

	// In text mode, validate ollama requires --model (FR-003).
	if p.format != "json" && p.adapterName == "ollama" && p.modelName == "" {
		return fmt.Errorf("--model is required when using ollama (FR-003)")
	}

	// Pre-flight validation for --coverprofile (FR-004, FR-005): check
	// existence and is-regular-file before the analysis pipeline starts so
	// that an invalid path produces a hard exit, not a silent partial failure.
	if p.coverProfile != "" {
		info, statErr := os.Stat(p.coverProfile)
		if statErr != nil {
			return fmt.Errorf("--coverprofile %q: %w", p.coverProfile, statErr)
		}
		if info.IsDir() {
			return fmt.Errorf("--coverprofile %q is a directory, not a file", p.coverProfile)
		}
	}

	return nil
}

func runReport(p reportParams) error {
	if err := validateReportParams(p); err != nil {
		return err
	}

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	moduleDir, findErr := loader.FindModuleRoot(cwd)
	if findErr != nil {
		return fmt.Errorf("finding module root: %w", findErr)
	}

	timeout := p.aiTimeout
	if timeout <= 0 {
		timeout = 10 * time.Minute
	}

	// adapterCfg is the single source of adapter configuration used for both
	// NewAdapter and RunnerOptions.AdapterCfg.
	adapterCfg := aireport.AdapterConfig{
		Name:    p.adapterName,
		Model:   p.modelName,
		Timeout: timeout,
	}

	// Resolve AI adapter (validates allowlist name). The pre-flight binary
	// check (FR-012) runs inside aireport.Run, before the analysis pipeline,
	// via ValidateAdapterBinary.
	var aiAdapter aireport.AIAdapter
	var systemPrompt string
	if p.format != "json" {
		var adapterErr error
		aiAdapter, adapterErr = aireport.NewAdapter(adapterCfg)
		if adapterErr != nil {
			return fmt.Errorf("invalid --ai value: %w", adapterErr)
		}

		// Load system prompt only in text mode (FR-015): in json mode the
		// prompt file is never needed and a permission error must not block output.
		var promptErr error
		systemPrompt, promptErr = aireport.LoadPrompt(moduleDir)
		if promptErr != nil {
			return fmt.Errorf("loading system prompt: %w", promptErr)
		}
	}

	stepSummaryPath := os.Getenv("GITHUB_STEP_SUMMARY")

	opts := aireport.RunnerOptions{
		Patterns:        p.patterns,
		ModuleDir:       moduleDir,
		Adapter:         aiAdapter,
		AdapterCfg:      adapterCfg,
		SystemPrompt:    systemPrompt,
		Format:          p.format,
		Stdout:          p.stdout,
		Stderr:          p.stderr,
		StepSummaryPath: stepSummaryPath,
		CoverProfile:    p.coverProfile,
		Thresholds: aireport.ThresholdConfig{
			MaxCrapload:         p.maxCrapload,
			MaxGazeCrapload:     p.maxGazeCrapload,
			MinContractCoverage: p.minContractCoverage,
		},
		TestShort: p.testShort,
	}

	// External analyzer path: when --analyzer is set, override the
	// CRAP step's providers with external adapters. The quality,
	// classify, and docscan steps are Go-specific and are skipped
	// when using an external analyzer (their errors are recorded
	// in the payload).
	if p.analyzerFlag != "" {
		analyzeFunc, cleanup, extErr := buildExternalReportAnalyzeFunc(
			p.analyzerFlag, p.languageFlag, moduleDir, p.patterns, p.stderr,
		)
		if extErr != nil {
			return extErr
		}
		defer cleanup()
		opts.AnalyzeFunc = analyzeFunc
	}

	runFn := p.runnerFunc
	if runFn == nil {
		runFn = aireport.Run
	}

	return runFn(opts)
}

// buildExternalReportAnalyzeFunc creates an AnalyzeFunc that uses an
// external analyzer for the CRAP step of the report pipeline. The
// quality, classify, and docscan steps are skipped (they are
// Go-specific). Returns the analyze function, a cleanup function
// (to close the session), and an error.
func buildExternalReportAnalyzeFunc(
	analyzerFlag, languageFlag, moduleDir string,
	patterns []string,
	stderr io.Writer,
) (func([]string, string) (*aireport.ReportPayload, error), func(), error) {
	session, providers, err := initExternalSession(
		analyzerFlag, languageFlag, moduleDir, patterns, stderr)
	if err != nil {
		return nil, nil, err
	}

	analyzeFunc := func(pats []string, modDir string) (*aireport.ReportPayload, error) {
		return runExternalReportCRAP(pats, modDir, providers, stderr)
	}

	cleanup := func() { _ = session.Close() }
	return analyzeFunc, cleanup, nil
}

// runExternalReportCRAP runs the CRAP step using external providers
// and builds a ReportPayload with only the CRAP section populated.
// Quality, classify, and docscan are Go-specific and skipped.
func runExternalReportCRAP(pats []string, modDir string, providers *adapter.Providers, stderr io.Writer) (*aireport.ReportPayload, error) {
	opts := crap.DefaultOptions()
	opts.Stderr = stderr
	wireExternalProviders(&opts, providers)

	rpt, err := crap.Analyze(pats, modDir, opts)
	if err != nil {
		return nil, fmt.Errorf("CRAP analysis with external analyzer: %w", err)
	}

	crapJSON, err := cliutil.CaptureJSON(func(w io.Writer) error {
		return crap.WriteJSON(w, rpt)
	})
	if err != nil {
		return nil, err
	}

	payload := &aireport.ReportPayload{CRAP: crapJSON}
	crapload := rpt.Summary.CRAPload
	payload.Summary.CRAPload = &crapload
	payload.Summary.GazeCRAPload = rpt.Summary.GazeCRAPload
	payload.Summary.TotalFunctions = rpt.Summary.TotalFunctions

	skipped := "skipped: external analyzer mode"
	payload.Errors.Quality = &skipped
	payload.Errors.Classify = &skipped
	payload.Errors.Docscan = &skipped

	return payload, nil
}

// newReportCmd creates the "report" subcommand that orchestrates gaze's four
// analysis operations and formats the result using an external AI CLI.
func newReportCmd() *cobra.Command {
	var (
		format        string
		adapterName   string
		modelName     string
		aiTimeout     time.Duration
		coverProfile  string
		analyzerFlag  string
		languageFlag  string
		testShortFlag bool

		// Threshold raw values and "was set" flags for *int semantics.
		maxCraploadVal     int
		maxGazeCraploadVal int
		minContractCovVal  int
	)

	cmd := &cobra.Command{
		Use:   "report [packages]",
		Short: "Generate an AI-formatted quality report",
		Long: `Orchestrate gaze's four analysis operations (CRAP, quality,
classification, docscan) and pipe the combined JSON payload to an
external AI CLI for formatting into a human-readable report.

The formatted markdown report is written to stdout and optionally
appended to $GITHUB_STEP_SUMMARY for GitHub Actions Step Summary.

Examples:
  gaze report ./... --ai=claude
  gaze report ./... --ai=gemini --model=gemini-2.5-pro
  gaze report ./... --ai=ollama --model=llama3.2
  gaze report ./... --ai=opencode
  gaze report ./... --ai=opencode --model=claude-3-5-sonnet
  gaze report ./... --format=json
  gaze report ./... --ai=claude --coverprofile=coverage.out`,
		Args: cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default package pattern is ./... when none specified (FR-014).
			if len(args) == 0 {
				args = []string{"./..."}
			}

			// Build *int threshold values using cmd.Flags().Changed() to
			// distinguish absent (nil) from explicitly-set zero.
			var maxCrapload, maxGazeCrapload, minContractCoverage *int
			if cmd.Flags().Changed("max-crapload") {
				maxCrapload = &maxCraploadVal
			}
			if cmd.Flags().Changed("max-gaze-crapload") {
				maxGazeCrapload = &maxGazeCraploadVal
			}
			if cmd.Flags().Changed("min-contract-coverage") {
				minContractCoverage = &minContractCovVal
			}

			p := reportParams{
				patterns:            args,
				format:              format,
				adapterName:         adapterName,
				modelName:           modelName,
				aiTimeout:           aiTimeout,
				maxCrapload:         maxCrapload,
				maxGazeCrapload:     maxGazeCrapload,
				minContractCoverage: minContractCoverage,
				coverProfile:        coverProfile,
				analyzerFlag:        analyzerFlag,
				languageFlag:        languageFlag,
				testShort:           testShortFlag,
				stdout:              cmd.OutOrStdout(),
				stderr:              cmd.ErrOrStderr(),
			}
			// Threshold evaluation and exit code are handled inside
			// runReport via aireport.Run; a non-nil error here means
			// a threshold failed or the pipeline errored.
			return runReport(p)
		},
	}

	cmd.Flags().StringVar(&format, "format", "text", "output format: text or json")
	cmd.Flags().StringVar(&adapterName, "ai", "", "AI adapter: claude, gemini, ollama, or opencode")
	cmd.Flags().StringVar(&modelName, "model", "", "model name (required for ollama)")
	cmd.Flags().DurationVar(&aiTimeout, "ai-timeout", 10*time.Minute, "AI adapter timeout")
	cmd.Flags().IntVar(&maxCraploadVal, "max-crapload", 0, "fail if CRAPload exceeds N")
	cmd.Flags().IntVar(&maxGazeCraploadVal, "max-gaze-crapload", 0, "fail if GazeCRAPload exceeds N")
	cmd.Flags().IntVar(&minContractCovVal, "min-contract-coverage", 0, "fail if avg contract coverage is below N%")
	cmd.Flags().StringVar(&coverProfile, "coverprofile", "", "path to a pre-generated coverage profile (skips internal go test run)")
	cmd.Flags().StringVar(&analyzerFlag, "analyzer", "", "external analyzer binary (e.g., snake-eyes)")
	cmd.Flags().StringVar(&languageFlag, "language", "", "target language for analyzer discovery (e.g., python)")
	cmd.Flags().BoolVar(&testShortFlag, "test-short", false, "pass -short to internal go test invocation (faster, less accurate coverage)")

	return cmd
}

// buildAIMapperFunc creates a quality.AIMapperFunc that delegates to
// the specified AI adapter backend. The returned function calls
// BuildAIMapperPrompt to construct the prompt, passes it to the
// adapter's Format method, and parses the response with
// ParseAIMapperResponse.
//
// Valid backend names are "claude", "gemini", "ollama", and "opencode".
// The model parameter is required for ollama and optional for other
// backends. Returns an error if the backend name is not in the
// allowlist or if ollama is specified without a model.
// Binary availability is validated at call time (not at construction
// time), so the returned function may fail when invoked if the
// backend binary is not on PATH.
func buildAIMapperFunc(backend, model string) (quality.AIMapperFunc, error) {
	if backend == "ollama" && model == "" {
		return nil, fmt.Errorf("--ai-mapper=ollama requires --ai-mapper-model to be set")
	}

	cfg := aireport.AdapterConfig{
		Name:    backend,
		Model:   model,
		Timeout: 2 * time.Minute,
	}
	aiAdapter, err := aireport.NewAdapter(cfg)
	if err != nil {
		return nil, fmt.Errorf("invalid --ai-mapper value: %w", err)
	}

	// System prompt provides static instructions; the per-assertion
	// context goes as the payload. This matches the adapter convention
	// where system prompt = agent persona and payload = data.
	const systemPrompt = "You are an assertion-to-side-effect mapper. " +
		"Given a test assertion and a list of side effects, determine " +
		"which side effect (if any) the assertion verifies. " +
		"Respond with ONLY the effect ID, or NONE if no match."

	return func(ctx quality.AIMapperContext) (string, error) {
		prompt := quality.BuildAIMapperPrompt(ctx)

		result, formatErr := aiAdapter.Format(
			context.Background(),
			systemPrompt,
			strings.NewReader(prompt),
		)
		if formatErr != nil {
			return "", fmt.Errorf("AI mapper %s: %w", backend, formatErr)
		}

		// Build valid IDs map from the context's side effects.
		validIDs := make(map[string]bool, len(ctx.SideEffects))
		for _, se := range ctx.SideEffects {
			validIDs[se.ID] = true
		}

		return quality.ParseAIMapperResponse(result, validIDs), nil
	}, nil
}
