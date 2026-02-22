// Package docscan discovers and prioritizes documentation files
// within a Go module repository for use in document-enhanced
// classification.
package docscan

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jflowers/gaze/internal/config"
)

// Priority defines the scanning priority of a document file,
// determined by its proximity to the target package.
type Priority int

const (
	// PrioritySamePackage is the highest priority, given to docs
	// in the same directory as the target package.
	PrioritySamePackage Priority = iota + 1

	// PriorityModuleRoot is given to docs in the module root.
	PriorityModuleRoot

	// PriorityOther is the base priority for all other docs.
	PriorityOther
)

// DocumentFile represents a discovered documentation file with
// its content and scan priority.
type DocumentFile struct {
	// Path is the path to the file, relative to the repo root.
	Path string `json:"path"`

	// Content is the full text content of the file.
	Content string `json:"content"`

	// Priority indicates how close the file is to the target
	// package (higher = more relevant).
	Priority Priority `json:"priority"`
}

// ScanOptions configures a Scan invocation.
type ScanOptions struct {
	// Config is the Gaze configuration providing exclude/include
	// patterns. If nil, DefaultConfig() is used.
	Config *config.GazeConfig

	// PackageDir is the directory of the target package, relative
	// to RepoRoot. Used to determine document priority.
	PackageDir string
}

// Scan walks the repository rooted at repoRoot, discovers all
// Markdown (.md) files, applies exclude/include filters from opts,
// and returns a prioritized list of DocumentFile values.
//
// If opts.Config.Classification.DocScan.Timeout is non-zero, the
// walk is bounded by that deadline. A context.DeadlineExceeded error
// is returned when the timeout is hit.
//
// Proximity priority:
//   - PrioritySamePackage: file is in the same directory as PackageDir
//   - PriorityModuleRoot:  file is directly in repoRoot
//   - PriorityOther:       all other matching files
func Scan(repoRoot string, opts ScanOptions) ([]DocumentFile, error) {
	if opts.Config == nil {
		opts.Config = config.DefaultConfig()
	}

	// Derive a context with timeout if configured.
	ctx := context.Background()
	timeout := opts.Config.Classification.DocScan.Timeout
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	var docs []DocumentFile

	err := filepath.WalkDir(repoRoot, func(path string, d fs.DirEntry, walkErr error) error {
		// Check for context cancellation on every entry.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return fmt.Errorf("doc scan timed out after %s: %w", timeout, ctxErr)
		}

		if walkErr != nil {
			return walkErr
		}

		// Compute the path relative to repoRoot for filtering.
		rel, relErr := filepath.Rel(repoRoot, path)
		if relErr != nil {
			return relErr
		}

		// Skip hidden directories (like .git) early to avoid deep walks.
		if d.IsDir() {
			base := d.Name()
			if strings.HasPrefix(base, ".") && base != "." {
				return filepath.SkipDir
			}
			return nil
		}

		// Only process Markdown files.
		if !strings.HasSuffix(strings.ToLower(d.Name()), ".md") {
			return nil
		}

		// Apply exclude/include filtering.
		if !Filter(rel, opts.Config) {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			// Skip unreadable files rather than aborting the scan.
			return nil //nolint:nilerr
		}

		priority := classifyPriority(rel, repoRoot, opts.PackageDir)

		docs = append(docs, DocumentFile{
			Path:     rel,
			Content:  string(content),
			Priority: priority,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Sort: PrioritySamePackage first, then PriorityModuleRoot,
	// then PriorityOther. Within a tier, alphabetical by path.
	sortDocuments(docs)

	return docs, nil
}

// scanTimeout returns the effective scan timeout, accounting for
// zero meaning "no timeout". Exported for testing.
func scanTimeout(cfg *config.GazeConfig) time.Duration {
	if cfg == nil {
		return config.DefaultConfig().Classification.DocScan.Timeout
	}
	return cfg.Classification.DocScan.Timeout
}

// classifyPriority determines the Priority of a document based on
// its relative path from the repo root.
func classifyPriority(rel, repoRoot, packageDir string) Priority {
	dir := filepath.Dir(rel)

	// Normalize: filepath.Dir of a root-level file is ".".
	if packageDir != "" {
		pkgRel, err := filepath.Rel(repoRoot, packageDir)
		if err == nil {
			if dir == pkgRel {
				return PrioritySamePackage
			}
		} else {
			// packageDir may already be relative.
			if dir == packageDir {
				return PrioritySamePackage
			}
		}
	}

	if dir == "." {
		return PriorityModuleRoot
	}

	return PriorityOther
}

// sortDocuments sorts docs in-place: ascending priority value means
// higher importance, so PrioritySamePackage (1) sorts before
// PriorityModuleRoot (2) before PriorityOther (3).
// Within a priority tier, files are sorted alphabetically by path.
func sortDocuments(docs []DocumentFile) {
	n := len(docs)
	// Insertion sort — small slices expected (< 50 files per spec).
	for i := 1; i < n; i++ {
		key := docs[i]
		j := i - 1
		for j >= 0 && (docs[j].Priority > key.Priority ||
			(docs[j].Priority == key.Priority && docs[j].Path > key.Path)) {
			docs[j+1] = docs[j]
			j--
		}
		docs[j+1] = key
	}
}
