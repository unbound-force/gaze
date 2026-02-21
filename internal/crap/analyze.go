package crap

import (
	"bufio"
	"fmt"
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

	// MaxCRAPload causes a non-zero exit if CRAPload exceeds this.
	// Zero means no limit (report-only).
	MaxCRAPload int

	// MaxGazeCRAPload causes a non-zero exit if GazeCRAPload exceeds
	// this. Zero means no limit.
	MaxGazeCRAPload int

	// IgnoreGenerated excludes functions in files with
	// "// Code generated" headers. Default: true.
	IgnoreGenerated bool
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
		defer os.Remove(coverProfile)
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

	ignorePattern := regexp.MustCompile(`_test\.go$`)
	complexityStats := gocyclo.Analyze(absPaths, ignorePattern)

	// Step 3: Parse coverage profile for per-function coverage.
	funcCoverages, err := ParseCoverProfile(coverProfile, moduleDir)
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

		// GazeCRAP, ContractCoverage, and Quadrant remain nil
		// until contract coverage is available (Specs 002-003).
		// Per FR-015: report as unavailable and exclude from
		// GazeCRAPload counts.

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
	tmpFile.Close()

	// Build args for go test. Patterns come from Cobra positional
	// args (already past flag parsing) and Go package patterns
	// (e.g., "./...") are syntactically distinct from flags.
	// Note: do NOT use "--" separator here — go test doesn't
	// support POSIX-style "--" and would ignore the patterns.
	args := []string{"test", "-coverprofile=" + profilePath}
	args = append(args, patterns...)

	cmd := exec.Command("go", args...)
	cmd.Dir = moduleDir
	output, err := cmd.CombinedOutput()
	if err != nil {
		os.Remove(profilePath)
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
	defer f.Close()

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
	crapload := 0
	gazeCRAPload := 0
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
			if *s.GazeCRAP >= opts.GazeCRAPThreshold {
				gazeCRAPload++
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
	}

	return summary
}
