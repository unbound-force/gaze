// Package docscan discovers and prioritizes documentation files.
package docscan

import (
	"path/filepath"
	"strings"

	"github.com/unbound-force/gaze/internal/config"
)

// Filter returns true if the given relative path should be included
// in the document scan, based on the exclude/include patterns in cfg.
//
// Logic:
//  1. If include patterns are set, the file must match at least one
//     include pattern to be processed (overrides default full-repo scan).
//  2. If the file matches any exclude pattern, it is excluded.
//  3. Otherwise, the file is included.
func Filter(rel string, cfg *config.GazeConfig) bool {
	if cfg == nil {
		cfg = config.DefaultConfig()
	}

	docScan := cfg.Classification.DocScan

	// Normalize separators to forward slash for matching consistency.
	rel = filepath.ToSlash(rel)

	// Include override: if include patterns are set, file must match
	// at least one.
	if len(docScan.Include) > 0 {
		matched := false
		for _, pattern := range docScan.Include {
			if matchGlob(pattern, rel) {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Exclude: if the file matches any exclude pattern, skip it.
	for _, pattern := range docScan.Exclude {
		if matchGlob(pattern, rel) {
			return false
		}
	}

	return true
}

// matchGlob matches a path against a glob pattern. It supports
// both simple glob syntax (filepath.Match) and double-star
// prefix patterns like "vendor/**" and "testdata/**".
func matchGlob(pattern, rel string) bool {
	// Handle double-star suffix: "vendor/**" matches any file under
	// the "vendor/" directory.
	if strings.HasSuffix(pattern, "/**") {
		prefix := strings.TrimSuffix(pattern, "/**")
		if rel == prefix || strings.HasPrefix(rel, prefix+"/") {
			return true
		}
		return false
	}

	// Handle simple patterns like "CHANGELOG.md" or "*.md".
	matched, err := filepath.Match(pattern, rel)
	if err != nil {
		return false
	}
	if matched {
		return true
	}

	// Also try matching just the base name for patterns without
	// path separators (e.g., "LICENSE" matches any "LICENSE" file).
	if !strings.Contains(pattern, "/") {
		base := filepath.Base(rel)
		matched, err = filepath.Match(pattern, base)
		if err != nil {
			return false
		}
		return matched
	}

	return false
}
