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
	"strings"
	"time"

	charmlog "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
	"github.com/unbound-force/gaze/internal/aireport"
	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/classify"
	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/crap"
	"github.com/unbound-force/gaze/internal/docscan"
	"github.com/unbound-force/gaze/internal/loader"
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

Creates .opencode/agents/ and .opencode/command/ directories with
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
	pkgPath           string
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
		path = filepath.Join(cwd, ".gaze.yaml")
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
	if p.format != "text" && p.format != "json" {
		return fmt.Errorf("invalid format %q: must be 'text' or 'json'", p.format)
	}

	opts := analysis.Options{
		IncludeUnexported: p.includeUnexported,
		FunctionFilter:    p.function,
		Version:           version,
	}

	logger.Info("analyzing package", "pkg", p.pkgPath)
	results, err := analysis.LoadAndAnalyze(p.pkgPath, opts)
	if err != nil {
		return err
	}

	if len(results) == 0 {
		if p.function != "" {
			return fmt.Errorf("function %q not found in package %q", p.function, p.pkgPath)
		}
		logger.Warn("no functions found to analyze")
		return nil
	}

	logger.Info("analysis complete", "functions", len(results))

	// --verbose implies --classify.
	if p.verbose {
		p.classify = true
	}

	// Run mechanical classification if requested.
	if p.classify {
		// Normalize zero to -1 (not set). The flag default is -1 but
		// struct literals in tests may leave these fields at their Go
		// zero value (0). Both mean "use config/default".
		contractualThresh := p.contractualThresh
		if contractualThresh == 0 {
			contractualThresh = -1
		}
		incidentalThresh := p.incidentalThresh
		if incidentalThresh == 0 {
			incidentalThresh = -1
		}
		cfg, cfgErr := loadConfig(p.configPath, contractualThresh, incidentalThresh)
		if cfgErr != nil {
			return fmt.Errorf("loading config: %w", cfgErr)
		}
		results, err = runClassify(results, p.pkgPath, cfg, p.verbose)
		if err != nil {
			return fmt.Errorf("classification: %w", err)
		}
	}

	if p.interactive {
		return runInteractiveAnalyze(results)
	}

	switch p.format {
	case "json":
		return report.WriteJSON(p.stdout, results, version)
	default:
		textOpts := report.TextOptions{
			Classify: p.classify,
			Verbose:  p.verbose,
		}
		return report.WriteTextOptions(p.stdout, results, textOpts)
	}
}

// runClassify runs the mechanical classification pipeline on
// analysis results and returns classified results. It adds a
// metadata warning noting that document-enhanced classification
// is not applied (the gaze-reporter agent handles that in full mode).
func runClassify(
	results []taxonomy.AnalysisResult,
	pkgPath string,
	cfg *config.GazeConfig,
	verbose bool,
) ([]taxonomy.AnalysisResult, error) {
	// Load the target package for AST access.
	targetResult, err := loader.Load(pkgPath)
	if err != nil {
		return nil, fmt.Errorf("loading target package: %w", err)
	}

	// Load the module for caller/interface analysis. Use the
	// directory containing the target package if possible.
	logger.Info("loading module packages for classification")
	cwd, err := os.Getwd()
	if err != nil {
		logger.Debug("could not determine working directory for module load", "err", err)
		cwd = ""
	}
	modResult, modErr := loader.LoadModule(cwd)
	var modPkgs []*packages.Package
	if modErr != nil {
		// Non-fatal: module loading failure means caller analysis
		// and interface signals will be degraded but not broken.
		logger.Warn("module loading failed; caller/interface signals degraded", "err", modErr)
	} else {
		modPkgs = modResult.Packages
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
		Use:   "analyze [package]",
		Short: "Analyze side effects of Go functions",
		Long: `Analyze a Go package (or specific function) and report all
observable side effects each function produces.

Use --classify to attach contractual classification (mechanical signals).
Use /gaze in OpenCode (full mode) for document-enhanced classification.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runAnalyze(analyzeParams{
				pkgPath:           args[0],
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
	stdout          io.Writer
	stderr          io.Writer

	// analyzeFunc overrides crap.Analyze for testing.
	// When nil, the production crap.Analyze is called.
	analyzeFunc func([]string, string, crap.Options) (*crap.Report, error)

	// coverageFunc overrides crap.BuildContractCoverageFunc for testing.
	// When nil, the production crap.BuildContractCoverageFunc is called.
	coverageFunc func([]string, string, io.Writer) (func(string, string) (crap.ContractCoverageInfo, bool), []string)
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
	if p.format != "text" && p.format != "json" {
		return fmt.Errorf("invalid format %q: must be 'text' or 'json'", p.format)
	}

	// Wire the quality pipeline to provide contract coverage for
	// GazeCRAP scoring. This is best-effort: if quality analysis
	// fails for any package, GazeCRAP falls back to unavailable.
	if p.opts.ContractCoverageFunc == nil {
		var ccFunc func(string, string) (crap.ContractCoverageInfo, bool)
		var degradedPkgs []string

		if p.coverageFunc != nil {
			// Test override — use the injected coverage function.
			ccFunc, degradedPkgs = p.coverageFunc(p.patterns, p.moduleDir, p.stderr)
		} else {
			// Production path — build AI mapper if requested.
			var aiMapperFn quality.AIMapperFunc
			if p.aiMapper != "" {
				var aiErr error
				aiMapperFn, aiErr = buildAIMapperFunc(p.aiMapper, p.aiMapperModel)
				if aiErr != nil {
					return aiErr
				}
			}
			ccFunc, degradedPkgs = crap.BuildContractCoverageFunc(
				p.patterns, p.moduleDir, p.stderr, aiMapperFn,
			)
		}

		if ccFunc != nil {
			p.opts.ContractCoverageFunc = ccFunc
		}
		if len(degradedPkgs) > 0 {
			p.opts.SSADegradedPackages = degradedPkgs
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

	// FR-015: Warn when GazeCRAP is unavailable. GazeCRAP requires
	// contract coverage data from `gaze quality`. If no
	// ContractCoverageFunc was provided, GazeCRAP fields are nil.
	if rpt.Summary.GazeCRAPload == nil {
		_, _ = fmt.Fprintln(p.stderr,
			"note: GazeCRAP unavailable — run 'gaze quality' to compute contract coverage")
	}

	// Resolve baseline path for comparison (D4).
	var comparisonResult *crap.ComparisonResult
	baselinePath, baselineExplicit := resolveBaselinePath(p.baselinePath, p.moduleDir)
	if baselinePath != "" {
		cr, baselineErr := loadAndCompare(baselinePath, baselineExplicit, rpt, p.moduleDir)
		if baselineErr != nil {
			return baselineErr
		}
		comparisonResult = cr
	}

	// Write output: comparison path or normal path.
	if comparisonResult != nil {
		if err := writeCrapComparisonReport(p.stdout, p.format, comparisonResult); err != nil {
			return err
		}
	} else {
		if err := writeCrapReport(p.stdout, p.format, rpt); err != nil {
			return err
		}
	}

	printCISummary(p.stderr, rpt, p.maxCrapload, p.maxGazeCrapload)

	// Evaluate baseline comparison gate before threshold gate
	// so comparison output is always visible (D7).
	if comparisonResult != nil && !comparisonResult.Summary.Passed {
		// Print comparison failure to stderr for visibility.
		_, _ = fmt.Fprintf(p.stderr, "baseline comparison: FAIL (%d regressions, %d new violations)\n",
			comparisonResult.Summary.Regressions,
			comparisonResult.Summary.NewViolations)
		return fmt.Errorf("baseline comparison failed: %d regressions, %d new-function violations",
			comparisonResult.Summary.Regressions,
			comparisonResult.Summary.NewViolations)
	}

	return checkCIThresholds(rpt, p.maxCrapload, p.maxGazeCrapload)
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
func resolveBaselinePath(flagPath, moduleDir string) (string, bool) {
	// 1. Explicit --baseline flag.
	if flagPath != "" {
		return flagPath, true
	}

	// 2. Config file baseline.file setting.
	cfg := loadGazeConfigBestEffort(moduleDir)
	if cfg.Baseline.File != "" && cfg.Baseline.File != ".gaze/baseline.json" {
		// Non-default config path — check if it exists and is
		// non-empty, skip silently if not found (D4: config path
		// is not explicit).
		configPath := cfg.Baseline.File
		if !filepath.IsAbs(configPath) {
			configPath = filepath.Join(moduleDir, configPath)
		}
		if info, err := os.Stat(configPath); err == nil && info.Size() > 0 {
			return configPath, false
		}
		return "", false
	}

	// 3. Default .gaze/baseline.json — skip silently if not found
	// or empty (an empty file is not a valid baseline; this
	// handles the shell redirect race where the output file is
	// truncated before gaze writes to it).
	defaultPath := filepath.Join(moduleDir, ".gaze", "baseline.json")
	if info, err := os.Stat(defaultPath); err == nil && info.Size() > 0 {
		return defaultPath, false
	}

	return "", false
}

// loadAndCompare loads a baseline file and runs comparison against
// the current report. If baselineExplicit is true (--baseline flag),
// a missing file is an error. Otherwise, a missing file is silently
// skipped.
func loadAndCompare(
	baselinePath string,
	baselineExplicit bool,
	current *crap.Report,
	moduleDir string,
) (*crap.ComparisonResult, error) {
	f, err := os.Open(baselinePath)
	if err != nil {
		if baselineExplicit {
			return nil, fmt.Errorf("--baseline %q: %w", baselinePath, err)
		}
		// Auto-detected path not found — silently skip.
		return nil, nil
	}
	defer f.Close()

	baseline, err := crap.LoadBaseline(f)
	if err != nil {
		if baselineExplicit {
			return nil, fmt.Errorf("loading baseline %q: %w", baselinePath, err)
		}
		// Auto-detected baseline is malformed or empty — skip
		// silently. The user didn't explicitly request comparison,
		// so a broken auto-detected file shouldn't block analysis.
		return nil, nil
	}

	// Load config for comparison options (epsilon, threshold).
	cfg := loadGazeConfigBestEffort(moduleDir)

	opts := crap.CompareOptions{
		Epsilon:              cfg.Baseline.Epsilon,
		NewFunctionThreshold: cfg.Baseline.NewFunctionThreshold,
	}

	return crap.Compare(baseline, current, opts), nil
}

// loadGazeConfigBestEffort loads the GazeConfig from the given
// module directory, falling back to default config on any error.
func loadGazeConfigBestEffort(moduleDir string) *config.GazeConfig {
	cfgPath := filepath.Join(moduleDir, ".gaze.yaml")
	cfg, err := config.Load(cfgPath)
	if err != nil {
		return config.DefaultConfig()
	}
	return cfg
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
	if maxGazeCrapload > 0 && rpt.Summary.GazeCRAPload != nil &&
		*rpt.Summary.GazeCRAPload > maxGazeCrapload {
		return fmt.Errorf("GazeCRAPload %d exceeds maximum %d",
			*rpt.Summary.GazeCRAPload, maxGazeCrapload)
	}
	return nil
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
		RunE: func(_ *cobra.Command, args []string) error {
			moduleDir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("getting working directory: %w", err)
			}
			opts := crap.DefaultOptions()
			opts.CoverProfile = coverProfile
			opts.CRAPThreshold = crapThreshold
			opts.GazeCRAPThreshold = gazeCrapThreshold
			opts.Stderr = os.Stderr
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
				stdout:          os.Stdout,
				stderr:          os.Stderr,
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
	repoRoot, err := os.Getwd()
	if err != nil {
		repoRoot = "."
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
	pkgPath              string
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
	stdout               io.Writer
	stderr               io.Writer
}

// runQuality is the extracted, testable body of the quality command.
func runQuality(p qualityParams) error {
	if p.format != "text" && p.format != "json" {
		return fmt.Errorf("invalid format %q: must be 'text' or 'json'", p.format)
	}

	// Step 1: Load and analyze the package (Spec 001).
	opts := analysis.Options{
		IncludeUnexported: p.includeUnexported,
		Version:           version,
	}

	// Auto-detect package main: include unexported functions
	// automatically since main packages have no exported API.
	if !opts.IncludeUnexported {
		if isMainPackage(p.pkgPath) {
			opts.IncludeUnexported = true
			logger.Info("package main detected, including unexported functions")
		}
	}

	logger.Info("analyzing package", "pkg", p.pkgPath)
	results, err := analysis.LoadAndAnalyze(p.pkgPath, opts)
	if err != nil {
		return err
	}
	if len(results) == 0 {
		logger.Warn("no functions found to analyze")
		return nil
	}

	// Step 2: Classify side effects (Spec 002).
	contractualThresh := p.contractualThresh
	if contractualThresh == 0 {
		contractualThresh = -1
	}
	incidentalThresh := p.incidentalThresh
	if incidentalThresh == 0 {
		incidentalThresh = -1
	}
	cfg, cfgErr := loadConfig(p.configPath, contractualThresh, incidentalThresh)
	if cfgErr != nil {
		return fmt.Errorf("loading config: %w", cfgErr)
	}
	results, err = runClassify(results, p.pkgPath, cfg, p.verbose)
	if err != nil {
		return fmt.Errorf("classification: %w", err)
	}

	// Step 3: Load the test package with test files.
	testPkg, err := loadTestPackage(p.pkgPath)
	if err != nil {
		return fmt.Errorf("loading test package: %w", err)
	}

	// Step 4: Assess test quality (Spec 003).
	qualOpts := quality.Options{
		TargetFunc: p.targetFunc,
		Verbose:    p.verbose,
		Version:    version,
		Stderr:     p.stderr,
	}

	// Wire AI-assisted assertion mapping when --ai-mapper is set.
	if p.aiMapper != "" {
		aiMapperFn, aiErr := buildAIMapperFunc(p.aiMapper, p.aiMapperModel)
		if aiErr != nil {
			return aiErr
		}
		qualOpts.AIMapperFunc = aiMapperFn
	}

	reports, summary, err := quality.Assess(results, testPkg, qualOpts)
	if err != nil {
		return fmt.Errorf("quality assessment: %w", err)
	}

	// Step 5: Write report.
	switch p.format {
	case "json":
		if err := quality.WriteJSON(p.stdout, reports, summary); err != nil {
			return err
		}
	default:
		if err := quality.WriteText(p.stdout, reports, summary); err != nil {
			return err
		}
	}

	// Step 6: Check CI thresholds.
	return checkQualityThresholds(p, reports, summary)
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

// isMainPackage checks if the given package pattern resolves to a
// package main. Uses a lightweight packages.Load with NeedName mode.
func isMainPackage(pattern string) bool {
	cfg := &packages.Config{Mode: packages.NeedName}
	pkgs, err := packages.Load(cfg, pattern)
	if err != nil || len(pkgs) == 0 {
		return false
	}
	return pkgs[0].Name == "main"
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
	)

	cmd := &cobra.Command{
		Use:   "quality [package]",
		Short: "Assess test quality via side effect mapping",
		Long: `Analyze how well a package's tests assert on the contractual
side effects of the functions they test. Reports Contract Coverage
(ratio of contractual effects that are asserted on) and Over-
Specification Score (assertions on incidental implementation details).

Requires the target package to have existing test files.`,
		Args: cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return runQuality(qualityParams{
				pkgPath:              args[0],
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

	return cmd
}

// findModuleRoot walks up from the current working directory to find
// the nearest directory containing a go.mod file (the module root).
// This ensures self-check always analyzes the full module, even when
// invoked from a subdirectory.
func findModuleRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no go.mod found in %q or any parent directory", dir)
		}
		dir = parent
	}
}

// selfCheckParams holds the parsed flags for the self-check command.
type selfCheckParams struct {
	format          string
	maxCrapload     int
	maxGazeCrapload int
	stdout          io.Writer
	stderr          io.Writer

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
	if p.format != "text" && p.format != "json" {
		return fmt.Errorf("invalid format %q: must be 'text' or 'json'", p.format)
	}

	findRoot := p.moduleRootFunc
	if findRoot == nil {
		findRoot = findModuleRoot
	}
	moduleDir, err := findRoot()
	if err != nil {
		return fmt.Errorf("finding module root: %w", err)
	}

	cp := crapParams{
		patterns:        []string{"./..."},
		format:          p.format,
		opts:            crap.DefaultOptions(),
		maxCrapload:     p.maxCrapload,
		maxGazeCrapload: p.maxGazeCrapload,
		moduleDir:       moduleDir,
		stdout:          p.stdout,
		stderr:          p.stderr,
	}
	cp.opts.Stderr = p.stderr

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
		RunE: func(_ *cobra.Command, _ []string) error {
			return runSelfCheck(selfCheckParams{
				format:          format,
				maxCrapload:     maxCrapload,
				maxGazeCrapload: maxGazeCrapload,
				stdout:          os.Stdout,
				stderr:          os.Stderr,
			})
		},
	}

	cmd.Flags().StringVar(&format, "format", "text",
		"output format: text or json")
	cmd.Flags().IntVar(&maxCrapload, "max-crapload", 0,
		"fail if CRAPload exceeds this count (0 = no limit)")
	cmd.Flags().IntVar(&maxGazeCrapload, "max-gaze-crapload", 0,
		"fail if GazeCRAPload exceeds this count (0 = no limit)")

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
	stdout              io.Writer
	stderr              io.Writer

	// runnerFunc overrides aireport.Run for testing. When nil, aireport.Run is called.
	runnerFunc func(aireport.RunnerOptions) error
}

// runReport is the extracted, testable body of the report command.
//
// In text mode it validates the --ai flag, resolves the adapter, loads the
// system prompt, and calls the 4-step analysis pipeline via aireport.Run.
// In json mode it skips AI adapter validation entirely (FR-015).
// Threshold evaluation runs after the pipeline and may set exit code 1.
func runReport(p reportParams) error {
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

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
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
	var adapter aireport.AIAdapter
	var systemPrompt string
	if p.format != "json" {
		var adapterErr error
		adapter, adapterErr = aireport.NewAdapter(adapterCfg)
		if adapterErr != nil {
			return fmt.Errorf("invalid --ai value: %w", adapterErr)
		}

		// Load system prompt only in text mode (FR-015): in json mode the
		// prompt file is never needed and a permission error must not block output.
		var promptErr error
		systemPrompt, promptErr = aireport.LoadPrompt(cwd)
		if promptErr != nil {
			return fmt.Errorf("loading system prompt: %w", promptErr)
		}
	}

	stepSummaryPath := os.Getenv("GITHUB_STEP_SUMMARY")

	opts := aireport.RunnerOptions{
		Patterns:        p.patterns,
		ModuleDir:       cwd,
		Adapter:         adapter,
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
	}

	runFn := p.runnerFunc
	if runFn == nil {
		runFn = aireport.Run
	}

	return runFn(opts)
}

// newReportCmd creates the "report" subcommand that orchestrates gaze's four
// analysis operations and formats the result using an external AI CLI.
func newReportCmd() *cobra.Command {
	var (
		format       string
		adapterName  string
		modelName    string
		aiTimeout    time.Duration
		coverProfile string

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
	adapter, err := aireport.NewAdapter(cfg)
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

		result, formatErr := adapter.Format(
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
