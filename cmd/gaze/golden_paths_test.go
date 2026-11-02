package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// aireportGoldenDir is the shared golden directory owned by
// internal/aireport (its goldenDir, "testdata/golden"), resolved
// relative to this package. CLI-level golden tests read the SAME
// files the renderer tests read, so both levels enforce one contract
// (spec 040 SC-003). cmd/gaze cannot import internal/aireport's
// _test.go helpers, so the path is duplicated here and guarded by
// TestAireportGoldenDir_MatchesAireport.
var aireportGoldenDir = filepath.Join("..", "..", "internal", "aireport", "testdata", "golden")

// aireportGoldenDecl is the exact goldenDir declaration in
// internal/aireport/golden_helpers_test.go that aireportGoldenDir
// mirrors.
const aireportGoldenDecl = `const goldenDir = "testdata/golden"`

// aireportGoldenPath returns the path of a shared golden file (for
// example "report-full.md" or "payload-full.json").
func aireportGoldenPath(name string) string {
	return filepath.Join(aireportGoldenDir, name)
}

// TestAireportGoldenDir_MatchesAireport guards against drift between
// the duplicated golden path and internal/aireport's goldenDir.
func TestAireportGoldenDir_MatchesAireport(t *testing.T) {
	aireportDir := filepath.Join("..", "..", "internal", "aireport")
	src, err := os.ReadFile(filepath.Join(aireportDir, "golden_helpers_test.go"))
	if err != nil {
		t.Fatalf("reading aireport golden helpers: %v", err)
	}
	if !strings.Contains(string(src), aireportGoldenDecl) {
		t.Fatalf("internal/aireport/golden_helpers_test.go no longer declares %s; "+
			"update aireportGoldenDir", aireportGoldenDecl)
	}
	want := filepath.Join(aireportDir, "testdata", "golden", "report-full.md")
	if got := aireportGoldenPath("report-full.md"); got != want {
		t.Errorf("aireportGoldenPath() = %q, want %q", got, want)
	}
	info, err := os.Stat(filepath.Dir(aireportGoldenDir))
	if err != nil || !info.IsDir() {
		t.Errorf("aireport testdata dir %s missing: %v", filepath.Dir(aireportGoldenDir), err)
	}
}
