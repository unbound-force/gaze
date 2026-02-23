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
	"github.com/jflowers/gaze/internal/analysis"
	"github.com/jflowers/gaze/internal/classify"
	"github.com/jflowers/gaze/internal/config"
	"github.com/jflowers/gaze/internal/crap"
	"github.com/jflowers/gaze/internal/docscan"
	"github.com/jflowers/gaze/internal/loader"
	"github.com/jflowers/gaze/internal/quality"
	"github.com/jflowers/gaze/internal/report"
	"github.com/jflowers/gaze/internal/taxonomy"
	"github.com/spf13/cobra"
	"golang.org/x/tools/go/packages"
)

// logger is the application-wide structured logger (writes to stderr).
var logger = charmlog.NewWithOptions(os.Stderr, charmlog.Options{
	ReportTimestamp: false,
})

// Set by build flags.
var version = "dev"

func main() {
	root := &cobra.Command{
		Use:   "gaze",
		Short: "Gaze — test quality analysis via side effect detection",
		Long: `Gaze analyzes Go functions to detect observable side effects
and measures whether unit tests assert on all contractual changes
produced by their test targets.`,
		Version: version,
	}

	root.AddCommand(newAnalyzeCmd())
	root.AddCommand(newCrapCmd())
	root.AddCommand(newQualityCmd())
	root.AddCommand(newSchemaCmd())
	root.AddCommand(newDocscanCmd())

	if err := root.Execute(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
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
	cwd, _ := os.Getwd()
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

	logger.Info("computing CRAP scores", "patterns", p.patterns)
	rpt, err := crap.Analyze(p.patterns, p.moduleDir, p.opts)
	if err != nil {
		return err
	}

	logger.Info("analysis complete", "functions", len(rpt.Scores))

	// FR-015: Warn when GazeCRAP is unavailable (contract coverage
	// requires Spec 003). This helps users understand the omission
	// rather than silently excluding GazeCRAP from output.
	if rpt.Summary.GazeCRAPload == nil {
		_, _ = fmt.Fprintln(p.stderr,
			"note: GazeCRAP unavailable — Spec 004 CRAP integration pending")
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

	// Fallback: return the first package.
	return pkgs[0], nil
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

	// Return first failure.
	if len(failures) > 0 {
		return errors.New(failures[0])
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
