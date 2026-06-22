package loader_test

import (
	"os"
	"path/filepath"
	"strings"
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

func TestLoad_MultiPackagePattern_ReturnsFirst(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test: loads real Go module via go/packages")
	}

	// Load with a wildcard pattern that resolves to multiple packages.
	// Load should return the first package without error.
	root := findModuleRoot(t)
	origDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatalf("Chdir(%q): %v", root, err)
	}
	t.Cleanup(func() { _ = os.Chdir(origDir) })

	result, err := loader.Load("./...")
	if err != nil {
		t.Fatalf("Load(\"./...\") failed: %v", err)
	}
	if result.Pkg == nil {
		t.Fatal("expected non-nil Pkg when multiple packages resolve")
	}
	if result.Fset == nil {
		t.Fatal("expected non-nil Fset when multiple packages resolve")
	}
	if result.Pkg.PkgPath == "" {
		t.Error("expected non-empty PkgPath on first resolved package")
	}
}

func TestResolvePackagePaths_Wildcard(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test: loads real Go module via go/packages")
	}

	root := findModuleRoot(t)
	paths, err := loader.ResolvePackagePaths([]string{"./..."}, root)
	if err != nil {
		t.Fatalf("ResolvePackagePaths failed: %v", err)
	}
	if len(paths) < 2 {
		t.Fatalf("expected multiple packages from ./..., got %d", len(paths))
	}

	// Verify no test-variant packages.
	for _, p := range paths {
		if strings.HasSuffix(p, "_test") {
			t.Errorf("expected no _test packages, got %q", p)
		}
	}

	// Verify deduplication: no path should appear twice.
	seen := make(map[string]bool)
	for _, p := range paths {
		if seen[p] {
			t.Errorf("duplicate package path: %q", p)
		}
		seen[p] = true
	}
}

func TestResolvePackagePaths_SinglePackage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test: loads real Go module via go/packages")
	}

	root := findModuleRoot(t)
	paths, err := loader.ResolvePackagePaths(
		[]string{"github.com/unbound-force/gaze/internal/loader"}, root,
	)
	if err != nil {
		t.Fatalf("ResolvePackagePaths failed: %v", err)
	}
	if len(paths) != 1 {
		t.Fatalf("expected 1 package, got %d: %v", len(paths), paths)
	}
	if paths[0] != "github.com/unbound-force/gaze/internal/loader" {
		t.Errorf("unexpected path: %q", paths[0])
	}
}

func TestResolvePackagePaths_EmptyPatterns(t *testing.T) {
	root := findModuleRoot(t)
	paths, err := loader.ResolvePackagePaths([]string{}, root)
	if err != nil {
		t.Fatalf("ResolvePackagePaths failed: %v", err)
	}
	// When no patterns are given, packages.Load resolves the
	// current directory, which may return the module root
	// package. Either 0 or 1 results is acceptable.
	if len(paths) > 1 {
		t.Errorf("expected 0 or 1 paths for empty patterns, got %d", len(paths))
	}
}

func TestIsMainPkg_Library(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test: loads package via go/packages")
	}
	if loader.IsMainPkg("github.com/unbound-force/gaze/internal/loader") {
		t.Error("expected library package to return false")
	}
}

func TestIsMainPkg_Main(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test: loads package via go/packages")
	}
	if !loader.IsMainPkg("github.com/unbound-force/gaze/cmd/gaze") {
		t.Error("expected main package to return true")
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

func TestFindModuleRoot_AtRoot(t *testing.T) {
	dir := t.TempDir()
	goMod := filepath.Join(dir, "go.mod")
	if err := os.WriteFile(goMod, []byte("module example.com/test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	got, err := loader.FindModuleRoot(dir)
	if err != nil {
		t.Fatalf("FindModuleRoot(%q) returned error: %v", dir, err)
	}
	if got != dir {
		t.Errorf("FindModuleRoot(%q) = %q, want %q", dir, got, dir)
	}
}

func TestFindModuleRoot_FromSubdirectory(t *testing.T) {
	root := t.TempDir()
	goMod := filepath.Join(root, "go.mod")
	if err := os.WriteFile(goMod, []byte("module example.com/test\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	deepDir := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatalf("creating subdirectories: %v", err)
	}

	got, err := loader.FindModuleRoot(deepDir)
	if err != nil {
		t.Fatalf("FindModuleRoot(%q) returned error: %v", deepDir, err)
	}
	if got != root {
		t.Errorf("FindModuleRoot(%q) = %q, want %q", deepDir, got, root)
	}
}

func TestFindModuleRoot_NoGoMod(t *testing.T) {
	dir := t.TempDir()

	_, err := loader.FindModuleRoot(dir)
	if err == nil {
		t.Fatal("FindModuleRoot() expected error for directory without go.mod, got nil")
	}
	if !strings.Contains(err.Error(), "go.mod") {
		t.Errorf("error message should contain 'go.mod', got: %v", err)
	}
}

func TestFindModuleRoot_NestedModules(t *testing.T) {
	root := t.TempDir()

	// Write go.mod at root level.
	rootGoMod := filepath.Join(root, "go.mod")
	if err := os.WriteFile(rootGoMod, []byte("module example.com/root\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("writing root go.mod: %v", err)
	}

	// Write go.mod in sub/ (nested module).
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("creating sub dir: %v", err)
	}
	subGoMod := filepath.Join(subDir, "go.mod")
	if err := os.WriteFile(subGoMod, []byte("module example.com/root/sub\n\ngo 1.21\n"), 0o644); err != nil {
		t.Fatalf("writing sub go.mod: %v", err)
	}

	// Create sub/deep/ directory (no go.mod here).
	deepDir := filepath.Join(subDir, "deep")
	if err := os.MkdirAll(deepDir, 0o755); err != nil {
		t.Fatalf("creating deep dir: %v", err)
	}

	// FindModuleRoot from sub/deep should find sub/ (nearest ancestor), not root.
	got, err := loader.FindModuleRoot(deepDir)
	if err != nil {
		t.Fatalf("FindModuleRoot(%q) returned error: %v", deepDir, err)
	}
	if got != subDir {
		t.Errorf("FindModuleRoot(%q) = %q, want %q (nearest ancestor with go.mod)", deepDir, got, subDir)
	}
}
