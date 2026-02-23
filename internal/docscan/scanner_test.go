package docscan_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unbound-force/gaze/internal/config"
	"github.com/unbound-force/gaze/internal/docscan"
)

// repoFixture returns the absolute path to the test fixture repo.
func repoFixture(t *testing.T) string {
	t.Helper()
	abs, err := filepath.Abs("testdata/repo")
	if err != nil {
		t.Fatalf("resolving fixture path: %v", err)
	}
	return abs
}

// TestScan_FindsMarkdownFiles verifies that Scan discovers all
// Markdown files in the fixture repo.
func TestScan_FindsMarkdownFiles(t *testing.T) {
	repo := repoFixture(t)
	docs, err := docscan.Scan(repo, docscan.ScanOptions{
		Config: &config.GazeConfig{
			Classification: config.ClassificationConfig{
				DocScan: config.DocScan{
					// No excludes — find everything.
					Exclude: nil,
					Include: nil,
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	// Expect README.md, CHANGELOG.md, CONTRIBUTING.md,
	// docs/architecture.md, vendor/README.md, pkg/mypackage/doc.md
	if len(docs) < 5 {
		t.Errorf("expected >= 5 docs, got %d", len(docs))
		for _, d := range docs {
			t.Logf("  %s (priority %d)", d.Path, d.Priority)
		}
	}
}

// TestScan_DefaultExcludesVendorAndChangelog verifies that the
// default config excludes vendor/ and CHANGELOG.md.
func TestScan_DefaultExcludesVendorAndChangelog(t *testing.T) {
	repo := repoFixture(t)
	docs, err := docscan.Scan(repo, docscan.ScanOptions{
		Config: config.DefaultConfig(),
	})
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	for _, d := range docs {
		if d.Path == "vendor/README.md" {
			t.Errorf("vendor/README.md should be excluded, but was found")
		}
		if d.Path == "CHANGELOG.md" {
			t.Errorf("CHANGELOG.md should be excluded, but was found")
		}
		if d.Path == "CONTRIBUTING.md" {
			t.Errorf("CONTRIBUTING.md should be excluded, but was found")
		}
	}
}

// TestScan_IncludeOverride verifies that setting include patterns
// restricts the scan to matching files only.
func TestScan_IncludeOverride(t *testing.T) {
	repo := repoFixture(t)
	docs, err := docscan.Scan(repo, docscan.ScanOptions{
		Config: &config.GazeConfig{
			Classification: config.ClassificationConfig{
				DocScan: config.DocScan{
					Include: []string{"docs/**"},
				},
			},
		},
	})
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	for _, d := range docs {
		if filepath.Dir(d.Path) != "docs" {
			t.Errorf("include override: unexpected file %q (dir %q)", d.Path, filepath.Dir(d.Path))
		}
	}

	if len(docs) == 0 {
		t.Errorf("expected docs/architecture.md to be found with include override")
	}
}

// TestScan_PriorityOrdering verifies that same-package docs sort
// before module-root docs, which sort before other docs.
func TestScan_PriorityOrdering(t *testing.T) {
	repo := repoFixture(t)
	pkgDir := filepath.Join(repo, "pkg", "mypackage")

	docs, err := docscan.Scan(repo, docscan.ScanOptions{
		Config:     config.DefaultConfig(),
		PackageDir: pkgDir,
	})
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	if len(docs) == 0 {
		t.Fatal("expected documents, got none")
	}

	// First doc should be same-package priority.
	first := docs[0]
	if first.Priority != docscan.PrioritySamePackage {
		t.Errorf("first doc should be PrioritySamePackage (1), got priority %d (path: %s)",
			first.Priority, first.Path)
	}

	// Verify monotonically non-decreasing priority values.
	for i := 1; i < len(docs); i++ {
		if docs[i].Priority < docs[i-1].Priority {
			t.Errorf("priority ordering violated at index %d: %d (%s) > %d (%s)",
				i, docs[i-1].Priority, docs[i-1].Path,
				docs[i].Priority, docs[i].Path)
		}
	}
}

// TestScan_EmptyRepo verifies that scanning an empty directory
// returns an empty slice without error.
func TestScan_EmptyRepo(t *testing.T) {
	tmp := t.TempDir()
	docs, err := docscan.Scan(tmp, docscan.ScanOptions{
		Config: config.DefaultConfig(),
	})
	if err != nil {
		t.Fatalf("Scan() on empty dir error: %v", err)
	}
	if len(docs) != 0 {
		t.Errorf("expected empty docs for empty repo, got %d", len(docs))
	}
}

// TestScan_ContentIsPopulated verifies that Content field is
// non-empty for discovered documents.
func TestScan_ContentIsPopulated(t *testing.T) {
	repo := repoFixture(t)
	docs, err := docscan.Scan(repo, docscan.ScanOptions{
		Config: config.DefaultConfig(),
	})
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}
	for _, d := range docs {
		if d.Content == "" {
			t.Errorf("document %q has empty content", d.Path)
		}
	}
}

// TestFilter_DefaultExcludes verifies that the default exclude list
// blocks vendor, testdata, and common non-useful files.
func TestFilter_DefaultExcludes(t *testing.T) {
	cfg := config.DefaultConfig()
	excluded := []string{
		"vendor/foo.md",
		"vendor/dep/README.md",
		"node_modules/bar.md",
		"testdata/fixtures.md",
		"CHANGELOG.md",
		"CONTRIBUTING.md",
		"CODE_OF_CONDUCT.md",
		"LICENSE.md",
	}
	for _, path := range excluded {
		if docscan.Filter(path, cfg) {
			t.Errorf("Filter(%q) = true, want false (should be excluded)", path)
		}
	}
}

// TestFilter_DefaultIncludes verifies that normal docs pass the
// default filter.
func TestFilter_DefaultIncludes(t *testing.T) {
	cfg := config.DefaultConfig()
	included := []string{
		"README.md",
		"docs/architecture.md",
		"internal/pkg/doc.md",
	}
	for _, path := range included {
		if !docscan.Filter(path, cfg) {
			t.Errorf("Filter(%q) = false, want true (should be included)", path)
		}
	}
}

// TestFilter_IncludeOverride verifies that when include patterns are
// set, only matching files pass.
func TestFilter_IncludeOverride(t *testing.T) {
	cfg := &config.GazeConfig{
		Classification: config.ClassificationConfig{
			DocScan: config.DocScan{
				Include: []string{"docs/**"},
			},
		},
	}

	if docscan.Filter("README.md", cfg) {
		t.Errorf("README.md should be filtered out when include=[docs/**]")
	}
	if !docscan.Filter("docs/architecture.md", cfg) {
		t.Errorf("docs/architecture.md should pass when include=[docs/**]")
	}
}

// TestScan_CustomExcludePatterns verifies that custom exclude
// patterns in the config are applied correctly (T038 — config-driven
// scanning integration test).
func TestScan_CustomExcludePatterns(t *testing.T) {
	repo := repoFixture(t)

	cfg := &config.GazeConfig{
		Classification: config.ClassificationConfig{
			DocScan: config.DocScan{
				Exclude: []string{"docs/**"},
			},
		},
	}

	docs, err := docscan.Scan(repo, docscan.ScanOptions{Config: cfg})
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	for _, d := range docs {
		if d.Path == "docs/architecture.md" {
			t.Errorf("docs/architecture.md should be excluded by custom pattern, but was found")
		}
	}
}

// TestScan_ConfigTimeout verifies that the Timeout field in the
// config is respected in the scan options struct (does not hang).
func TestScan_ConfigTimeout(t *testing.T) {
	repo := repoFixture(t)
	cfg := config.DefaultConfig()
	cfg.Classification.DocScan.Timeout = 5 * 1000000000 // 5s

	docs, err := docscan.Scan(repo, docscan.ScanOptions{Config: cfg})
	if err != nil {
		t.Fatalf("Scan() with timeout config error: %v", err)
	}

	// We don't fail on count here — just verifying no panic/hang.
	if docs == nil {
		t.Error("expected non-nil docs slice")
	}
}

// TestScan_SkipsHiddenDirs verifies that hidden directories like
// .git are not traversed.
func TestScan_SkipsHiddenDirs(t *testing.T) {
	tmp := t.TempDir()
	gitDir := filepath.Join(tmp, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatalf("creating .git dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(gitDir, "COMMIT_EDITMSG.md"), []byte("commit"), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmp, "README.md"), []byte("# Readme"), 0o644); err != nil {
		t.Fatalf("writing file: %v", err)
	}

	docs, err := docscan.Scan(tmp, docscan.ScanOptions{
		Config: config.DefaultConfig(),
	})
	if err != nil {
		t.Fatalf("Scan() error: %v", err)
	}

	for _, d := range docs {
		if d.Path == ".git/COMMIT_EDITMSG.md" {
			t.Errorf(".git directory should be skipped, found: %s", d.Path)
		}
	}
}
