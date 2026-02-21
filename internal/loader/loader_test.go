package loader_test

import (
	"testing"

	"github.com/jflowers/gaze/internal/loader"
)

func TestLoad_ValidPackage(t *testing.T) {
	// Load the loader package itself (it's a valid Go package).
	result, err := loader.Load("github.com/jflowers/gaze/internal/loader")
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}
	if result.Pkg == nil {
		t.Fatal("expected non-nil Pkg")
	}
	if result.Fset == nil {
		t.Fatal("expected non-nil Fset")
	}
	if result.Pkg.PkgPath != "github.com/jflowers/gaze/internal/loader" {
		t.Errorf("expected pkg path 'github.com/jflowers/gaze/internal/loader', got %q",
			result.Pkg.PkgPath)
	}
}

func TestLoad_InvalidPattern(t *testing.T) {
	_, err := loader.Load("github.com/nonexistent/package/that/does/not/exist")
	if err == nil {
		t.Error("expected error for nonexistent package")
	}
}
