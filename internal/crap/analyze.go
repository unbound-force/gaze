// Package crap computes CRAP scores for Go functions.
package crap

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fzipp/gocyclo"
)

// Options configures CRAP analysis.
type Options struct {
	// CoverProfile is the path to a coverage profile file.
	// If empty, Gaze will generate one automatically.
	CoverProfile string

	// CRAPThreshold is the threshold for flagging a function as
	// "crappy". Default: 15.
	CRAPThreshold float64

	// GazeCRAPThreshold is the threshold for GazeCRAP. Default: 15.
	// Used only when contract coverage is available.
	GazeCRAPThreshold float64

	// IgnoreGenerated excludes functions in files with
	// "// Code generated" headers. Default: true.
	IgnoreGenerated bool

	// Stderr receives warnings about files that could not be parsed
	// during coverage analysis. If nil, warnings are suppressed.
	Stderr io.Writer

	// ContractCoverageFunc is an optional function that returns the
	// contract coverage percentage (0-100) for a given function.
	// When provided, GazeCRAP scores, contract coverage, and
	// quadrant classifications are computed for each function.
	// If nil, GazeCRAP fields remain unavailable (FR-015).
	ContractCoverageFunc func(pkg, function string) (float64, bool)
}

// DefaultOptions returns options with sensible defaults.
func DefaultOptions() Options {
	return Options{
		CRAPThreshold:     15,
		GazeCRAPThreshold: 15,
		IgnoreGenerated:   true,
	}
}

// Analyze computes CRAP scores for all functions in the given
// package patterns.
func Analyze(patterns []string, moduleDir string, opts Options) (*Report, error) {
	if opts.CRAPThreshold <= 0 {
		opts.CRAPThreshold = 15
	}

	// Step 1: Generate coverage profile if not provided.
	coverProfile := opts.CoverProfile
	if coverProfile == "" {
		var err error
		coverProfile, err = generateCoverProfile(moduleDir, patterns)
		if err != nil {
			return nil, fmt.Errorf("generating coverage: %w", err)
		}
		defer func() { _ = os.Remove(coverProfile) }()
	} else {
		// Validate user-supplied cover profile path.
		coverProfile = filepath.Clean(coverProfile)
		info, err := os.Stat(coverProfile)
		if err != nil {
			return nil, fmt.Errorf("cover profile %q: %w", coverProfile, err)
		}
		if info.IsDir() {
			return nil, fmt.Errorf("cover profile %q is a directory, not a file", coverProfile)
		}
	}

	// Step 2: Compute cyclomatic complexity for all functions.
	absPaths, err := resolvePatterns(patterns, moduleDir)
	if err != nil {
		return nil, fmt.Errorf("resolving patterns: %w", err)
	}

	complexityStats := gocyclo.Analyze(absPaths, testFileRegexp)

	// Step 3: Parse coverage profile for per-function coverage.
	funcCoverages, err := ParseCoverProfile(coverProfile, moduleDir, opts.Stderr)
	if err != nil {
		return nil, fmt.Errorf("parsing coverage profile: %w", err)
	}

	// Step 4: Build coverage lookup map (file:line → coverage).
	coverMap := buildCoverMap(funcCoverages)

	// Step 5: Join complexity with coverage and compute CRAP.
	// Cache generated-file checks to avoid re-reading files.
	generatedCache := make(map[string]bool)

	var scores []Score
	for _, stat := range complexityStats {
		// Skip test files (already excluded by ignore pattern but
		// belt-and-suspenders).
		if strings.HasSuffix(stat.Pos.Filename, "_test.go") {
			continue
		}

		// Skip generated files when configured.
		if opts.IgnoreGenerated {
			gen, ok := generatedCache[stat.Pos.Filename]
			if !ok {
				gen = isGeneratedFile(stat.Pos.Filename)
				generatedCache[stat.Pos.Filename] = gen
			}
			if gen {
				continue
			}
		}

		covPct := lookupCoverage(stat, coverMap)
		crapScore := Formula(stat.Complexity, covPct)

		score := Score{
			Package:      stat.PkgName,
			Function:     stat.FuncName,
			File:         stat.Pos.Filename,
			Line:         stat.Pos.Line,
			Complexity:   stat.Complexity,
			LineCoverage: covPct,
			CRAP:         crapScore,
		}

		// Compute GazeCRAP if contract coverage is available.
		if opts.ContractCoverageFunc != nil {
			ccPct, ok := opts.ContractCoverageFunc(stat.PkgName, stat.FuncName)
			if ok {
				gazeCRAP := Formula(stat.Complexity, ccPct)
				quadrant := ClassifyQuadrant(
					crapScore, gazeCRAP,
					opts.CRAPThreshold, opts.GazeCRAPThreshold,
				)
				score.ContractCoverage = &ccPct
				score.GazeCRAP = &gazeCRAP
				score.Quadrant = &quadrant
			}
		}

		scores = append(scores, score)
	}

	// Step 6: Build summary.
	summary := buildSummary(scores, opts)

	return &Report{
		Scores:  scores,
		Summary: summary,
	}, nil
}

// generateCoverProfile runs go test to produce a coverage profile.
// The profile is written to a temporary file to avoid clobbering
// any existing cover.out in the user's working directory.
func generateCoverProfile(moduleDir string, patterns []string) (string, error) {
	tmpFile, err := os.CreateTemp("", "gaze-cover-*.out")
	if err != nil {
		return "", fmt.Errorf("creating temp cover profile: %w", err)
	}
	profilePath := tmpFile.Name()
	_ = tmpFile.Close()

	// Build args for go test. Patterns come from Cobra positional
	// args (already past flag parsing) and Go package patterns
	// (e.g., "./...") are syntactically distinct from flags.
	// Note: do NOT use "--" separator here — go test doesn't
	// support POSIX-style "--" and would ignore the patterns.
	//
	// The -short flag skips heavyweight tests (e.g., self-check)
	// that would re-invoke go test, causing recursive subprocess
	// chains. Coverage data from unit + integration tests is
	// sufficient for CRAP score computation.
	args := []string{"test", "-short", "-coverprofile=" + profilePath}
	args = append(args, patterns...)

	cmd := exec.Command("go", args...)
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		_ = os.Remove(profilePath)
		return "", fmt.Errorf("go test failed: %s\n%s", err, string(output))
	}

	return profilePath, nil
}

// resolvePatterns converts Go package patterns (./...) to filesystem
// paths that gocyclo can walk.
func resolvePatterns(patterns []string, moduleDir string) ([]string, error) {
	var paths []string
	for _, p := range patterns {
		if p == "./..." {
			paths = append(paths, moduleDir)
			continue
		}
		if strings.HasPrefix(p, "./") {
			abs := filepath.Join(moduleDir, p)
			paths = append(paths, abs)
			continue
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// coverKey creates a lookup key from file path and line number.
type coverKey struct {
	file string
	line int
}

// coverMaps holds both exact-path and basename-based coverage
// lookup maps for O(1) access in both cases.
type coverMaps struct {
	exact    map[coverKey]float64
	basename map[coverKey]float64
}

// buildCoverMap creates lookup maps from (file, startLine) to
// coverage percentage. A secondary basename-keyed index enables
// fast fallback matching when paths differ.
func buildCoverMap(coverages []FuncCoverage) coverMaps {
	exact := make(map[coverKey]float64, len(coverages))
	base := make(map[coverKey]float64, len(coverages))
	for _, fc := range coverages {
		exact[coverKey{file: fc.File, line: fc.StartLine}] = fc.Percentage
		base[coverKey{file: filepath.Base(fc.File), line: fc.StartLine}] = fc.Percentage
	}
	return coverMaps{exact: exact, basename: base}
}

// lookupCoverage finds the coverage for a gocyclo Stat by matching
// on file path and line number.
func lookupCoverage(stat gocyclo.Stat, maps coverMaps) float64 {
	// Try exact match on absolute path + line.
	key := coverKey{file: stat.Pos.Filename, line: stat.Pos.Line}
	if pct, ok := maps.exact[key]; ok {
		return pct
	}

	// Try matching by filename basename + line (handles path differences).
	baseKey := coverKey{file: filepath.Base(stat.Pos.Filename), line: stat.Pos.Line}
	if pct, ok := maps.basename[baseKey]; ok {
		return pct
	}

	// No coverage data — function was never executed.
	return 0
}

// testFileRegexp matches Go test files by suffix.
var testFileRegexp = regexp.MustCompile(`_test\.go$`)

// generatedRegexp matches the Go convention for generated file headers:
// "^// Code generated .* DO NOT EDIT\.$"
var generatedRegexp = regexp.MustCompile(`^// Code generated .* DO NOT EDIT\.$`)

// isGeneratedFile checks whether a Go source file was auto-generated
// by looking for a "// Code generated ... DO NOT EDIT." comment line
// before the package clause, per the Go convention.
func isGeneratedFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		// Stop scanning once we reach the package clause.
		if strings.HasPrefix(trimmed, "package ") {
			return false
		}
		if generatedRegexp.MatchString(trimmed) {
			return true
		}
	}
	return false
}

// buildSummary computes aggregate statistics from the scores.
func buildSummary(scores []Score, opts Options) Summary {
	if len(scores) == 0 {
		return Summary{
			CRAPThreshold: opts.CRAPThreshold,
		}
	}

	var totalComp, totalCov, totalCRAP float64
	var totalGazeCRAP, totalContractCov float64
	crapload := 0
	gazeCRAPload := 0
	gazeCRAPCount := 0
	quadrantCounts := make(map[Quadrant]int)
	hasGazeCRAP := false

	for _, s := range scores {
		totalComp += float64(s.Complexity)
		totalCov += s.LineCoverage
		totalCRAP += s.CRAP
		if s.CRAP >= opts.CRAPThreshold {
			crapload++
		}
		if s.GazeCRAP != nil {
			hasGazeCRAP = true
			gazeCRAPCount++
			totalGazeCRAP += *s.GazeCRAP
			if *s.GazeCRAP >= opts.GazeCRAPThreshold {
				gazeCRAPload++
			}
			// ContractCoverage is always set alongside GazeCRAP
			// (GazeCRAP is computed from ContractCoverage).
			if s.ContractCoverage != nil {
				totalContractCov += *s.ContractCoverage
			}
		}
		if s.Quadrant != nil {
			quadrantCounts[*s.Quadrant]++
		}
	}

	n := float64(len(scores))

	// Worst offenders: sort by CRAP descending, take top 5.
	sorted := make([]Score, len(scores))
	copy(sorted, scores)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CRAP > sorted[j].CRAP
	})
	worst := sorted
	if len(worst) > 5 {
		worst = worst[:5]
	}

	summary := Summary{
		TotalFunctions:  len(scores),
		AvgComplexity:   totalComp / n,
		AvgLineCoverage: totalCov / n,
		AvgCRAP:         totalCRAP / n,
		CRAPload:        crapload,
		CRAPThreshold:   opts.CRAPThreshold,
		WorstCRAP:       worst,
	}

	if hasGazeCRAP {
		summary.GazeCRAPload = &gazeCRAPload
		summary.GazeCRAPThreshold = &opts.GazeCRAPThreshold
		summary.QuadrantCounts = quadrantCounts

		avgGazeCRAP := totalGazeCRAP / float64(gazeCRAPCount)
		summary.AvgGazeCRAP = &avgGazeCRAP

		avgContractCov := totalContractCov / float64(gazeCRAPCount)
		summary.AvgContractCoverage = &avgContractCov

		// Worst offenders by GazeCRAP: filter to non-nil only,
		// sort descending, take top 5.
		var gazeScores []Score
		for _, s := range scores {
			if s.GazeCRAP != nil {
				gazeScores = append(gazeScores, s)
			}
		}
		sort.Slice(gazeScores, func(i, j int) bool {
			return *gazeScores[i].GazeCRAP > *gazeScores[j].GazeCRAP
		})
		if len(gazeScores) > 5 {
			gazeScores = gazeScores[:5]
		}
		summary.WorstGazeCRAP = gazeScores
	}

	return summary
}
