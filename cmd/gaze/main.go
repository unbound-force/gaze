// Package main implements the gaze CLI, a static analysis tool for
// Go that detects observable side effects and computes CRAP scores.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	charmlog "github.com/charmbracelet/log"
	"github.com/spf13/cobra"
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
// is not applied (that requires the /classify-docs command).
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
				"run /classify-docs for document-enhanced results",
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
Use /classify-docs in OpenCode for document-enhanced classification.`,
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
	stdout          io.Writer
	stderr          io.Writer

	// analyzeFunc overrides crap.Analyze for testing.
	// When nil, the production crap.Analyze is called.
	analyzeFunc func([]string, string, crap.Options) (*crap.Report, error)

	// coverageFunc overrides buildContractCoverageFunc for testing.
	// When nil, the production buildContractCoverageFunc is called.
	coverageFunc func([]string, string, io.Writer) func(string, string) (float64, bool)
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
		buildCoverage := p.coverageFunc
		if buildCoverage == nil {
			buildCoverage = buildContractCoverageFunc
		}
		ccFunc := buildCoverage(p.patterns, p.moduleDir, p.stderr)
		if ccFunc != nil {
			p.opts.ContractCoverageFunc = ccFunc
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

	if err := writeCrapReport(p.stdout, p.format, rpt); err != nil {
		return err
	}

	printCISummary(p.stderr, rpt, p.maxCrapload, p.maxGazeCrapload)

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

// resolvePackagePaths resolves package patterns to individual
// package paths, filtering out test-variant packages (those with
// a "_test" suffix). Returns the deduplicated list of package paths
// or an error if pattern resolution fails.
func resolvePackagePaths(patterns []string, moduleDir string) ([]string, error) {
	cfg := &packages.Config{
		Mode: packages.NeedName,
		Dir:  moduleDir,
	}
	pkgs, err := packages.Load(cfg, patterns...)
	if err != nil {
		return nil, fmt.Errorf("resolving package patterns: %w", err)
	}

	pkgPaths := make([]string, 0, len(pkgs))
	seen := make(map[string]bool)
	for _, pkg := range pkgs {
		if pkg.PkgPath == "" || seen[pkg.PkgPath] || strings.HasSuffix(pkg.PkgPath, "_test") {
			continue
		}
		seen[pkg.PkgPath] = true
		pkgPaths = append(pkgPaths, pkg.PkgPath)
	}
	return pkgPaths, nil
}

// analyzePackageCoverage runs the 4-step quality pipeline on a single
// package (analysis -> classify -> test-load -> quality assess) and
// returns the quality reports. Returns nil if any step fails.
func analyzePackageCoverage(
	pkgPath string,
	gazeConfig *config.GazeConfig,
	stderr io.Writer,
) []taxonomy.QualityReport {
	analysisOpts := analysis.Options{
		IncludeUnexported: false,
		Version:           version,
	}

	// Step 1: Analyze (Spec 001).
	results, err := analysis.LoadAndAnalyze(pkgPath, analysisOpts)
	if err != nil {
		logger.Debug("quality pipeline: analysis failed", "pkg", pkgPath, "err", err)
		return nil
	}
	if len(results) == 0 {
		logger.Debug("quality pipeline: no analysis results", "pkg", pkgPath)
		return nil
	}

	// Step 2: Classify (Spec 002).
	classified, err := runClassify(results, pkgPath, gazeConfig, false)
	if err != nil {
		logger.Debug("quality pipeline: classification failed", "pkg", pkgPath, "err", err)
		return nil
	}

	// Step 3: Load test package.
	testPkg, err := loadTestPackage(pkgPath)
	if err != nil {
		logger.Debug("quality pipeline: test package load failed", "pkg", pkgPath, "err", err)
		return nil
	}

	// Step 4: Assess quality (Spec 003).
	qualOpts := quality.Options{
		Version: version,
		Stderr:  stderr,
	}
	reports, _, err := quality.Assess(classified, testPkg, qualOpts)
	if err != nil {
		logger.Debug("quality pipeline: quality assessment failed", "pkg", pkgPath, "err", err)
		return nil
	}
	return reports
}

// buildContractCoverageFunc runs the quality pipeline across the
// given package patterns and returns a ContractCoverageFunc callback
// for GazeCRAP scoring. This is best-effort: if the quality pipeline
// fails for any package (no tests, config errors, etc.), those
// packages are silently skipped. Returns nil if no coverage data
// could be collected.
func buildContractCoverageFunc(
	patterns []string,
	moduleDir string,
	stderr io.Writer,
) func(pkg, function string) (float64, bool) {
	pkgPaths, err := resolvePackagePaths(patterns, moduleDir)
	if err != nil {
		logger.Debug("quality pipeline: failed to resolve packages", "err", err)
		return nil
	}

	if len(pkgPaths) == 0 {
		return nil
	}

	// Load config once for all packages.
	gazeConfig, cfgErr := loadConfig("", -1, -1)
	if cfgErr != nil {
		logger.Debug("quality pipeline: config load failed", "err", cfgErr)
		gazeConfig = config.DefaultConfig()
	}

	// Build coverage map: "shortPkg:qualifiedName" -> percentage.
	coverageMap := make(map[string]float64)

	for _, pkgPath := range pkgPaths {
		reports := analyzePackageCoverage(pkgPath, gazeConfig, stderr)
		for _, report := range reports {
			shortPkg := extractShortPkgName(report.TargetFunction.Package)
			key := shortPkg + ":" + report.TargetFunction.QualifiedName()
			if existing, ok := coverageMap[key]; !ok || report.ContractCoverage.Percentage > existing {
				coverageMap[key] = report.ContractCoverage.Percentage
			}
		}
	}

	if len(coverageMap) == 0 {
		return nil
	}

	logger.Info("quality pipeline complete",
		"functions_with_coverage", len(coverageMap))

	return func(pkg, function string) (float64, bool) {
		key := pkg + ":" + function
		pct, ok := coverageMap[key]
		return pct, ok
	}
}

// extractShortPkgName returns the short package name from a full
// import path. For "github.com/unbound-force/gaze/internal/crap", it
// returns "crap".
func extractShortPkgName(importPath string) string {
	if idx := strings.LastIndex(importPath, "/"); idx >= 0 {
		return importPath[idx+1:]
	}
	return importPath
}

func newCrapCmd() *cobra.Command {
	var (
		format            string
		coverProfile      string
		crapThreshold     float64
		gazeCrapThreshold float64
		maxCrapload       int
		maxGazeCrapload   int
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
to the /classify-docs OpenCode command for document-enhanced
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
	configPath           string
	contractualThresh    int
	incidentalThresh     int
	minContractCoverage  int
	maxOverSpecification int
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
		IncludeUnexported: false,
		Version:           version,
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
		configPath           string
		contractualThresh    int
		incidentalThresh     int
		minContractCoverage  int
		maxOverSpecification int
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
				configPath:           configPath,
				contractualThresh:    contractualThresh,
				incidentalThresh:     incidentalThresh,
				minContractCoverage:  minContractCoverage,
				maxOverSpecification: maxOverSpecification,
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
