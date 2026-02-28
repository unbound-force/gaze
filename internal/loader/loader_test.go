package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/unbound-force/gaze/internal/loader"
)

func TestLoad_ValidPackage(t *testing.T) {
	// Load the loader package itself (it's a valid Go package).
	result, err := loader.Load("github.com/unbound-force/gaze/internal/loader")
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if result.Pkg == nil {
		t.Fatal("expected non-nil Pkg")
	}
	if result.Fset == nil {
		t.Fatal("expected non-nil Fset")
	}
	if result.Pkg.PkgPath != "github.com/unbound-force/gaze/internal/loader" {
		t.Errorf("expected pkg path 'github.com/unbound-force/gaze/internal/loader', got %q",
			result.Pkg.PkgPath)
	}
}

func TestLoad_InvalidPattern(t *testing.T) {
	_, err := loader.Load("github.com/nonexistent/package/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent package")
	}
}

// findModuleRoot walks up from the current directory to find go.mod.
func findModuleRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("could not find module root (go.mod)")
		}
		dir = parent
	}
}

func TestLoadModule_ValidModule(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test: loads real Go module via go/packages")
	}

	root := findModuleRoot(t)
	result, err := loader.LoadModule(root)
	if err != nil {
		t.Fatalf("LoadModule(%q) failed: %v", root, err)
	}
	if len(result.Packages) == 0 {
		t.Fatal("expected at least one package")
	}
	if result.Fset == nil {
		t.Fatal("expected non-nil Fset")
	}

	// Verify at least one package has resolved type information.
	hasTypes := false
	for _, pkg := range result.Packages {
		if pkg.Types != nil {
			hasTypes = true
			break
		}
	}
	if !hasTypes {
		t.Error("expected at least one package with resolved type information")
	}
}

func TestLoadModule_NonExistentDir(t *testing.T) {
	_, err := loader.LoadModule("/nonexistent/path/that/does/not/exist")
	if err == nil {
		t.Error("expected error for non-existent directory")
	}
}

func TestLoadModule_ExcludesBrokenPackages(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test: creates temp module and invokes Go toolchain")
	}

	// Create a temporary directory with a go.mod and two packages:
	// one valid and one broken.
	tmpDir := t.TempDir()

	// Write go.mod.
	goMod := "module example.com/testmod\n\ngo 1.21\n"
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Create valid package.
	validDir := filepath.Join(tmpDir, "valid")
	if err := os.MkdirAll(validDir, 0o755); err != nil {
		t.Fatalf("creating valid dir: %v", err)
	}
	validSrc := "package valid\n\nfunc Hello() string { return \"hello\" }\n"
	if err := os.WriteFile(filepath.Join(validDir, "valid.go"), []byte(validSrc), 0o644); err != nil {
		t.Fatalf("writing valid.go: %v", err)
	}

	// Create broken package (syntax error).
	brokenDir := filepath.Join(tmpDir, "broken")
	if err := os.MkdirAll(brokenDir, 0o755); err != nil {
		t.Fatalf("creating broken dir: %v", err)
	}
	brokenSrc := "package broken\n\nfunc Oops() { this is not valid go }\n"
	if err := os.WriteFile(filepath.Join(brokenDir, "broken.go"), []byte(brokenSrc), 0o644); err != nil {
		t.Fatalf("writing broken.go: %v", err)
	}

	result, err := loader.LoadModule(tmpDir)
	if err != nil {
		t.Fatalf("LoadModule(%q) failed: %v", tmpDir, err)
	}

	// Should have at least the valid package.
	if len(result.Packages) == 0 {
		t.Fatal("expected at least one valid package")
	}

	// Verify the valid package is present.
	foundValid := false
	for _, pkg := range result.Packages {
		if pkg.PkgPath == "example.com/testmod/valid" {
			foundValid = true
		}
		// Verify broken package is excluded.
		if pkg.PkgPath == "example.com/testmod/broken" {
			t.Errorf("broken package should have been excluded, but found %q", pkg.PkgPath)
		}
	}
	if !foundValid {
		t.Error("expected valid package 'example.com/testmod/valid' in result")
	}
}
