package analysis_test

import (
	"testing"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// TestAnalyzeReturns_Direct_SingleReturn verifies that AnalyzeReturns
// detects 1 ReturnValue effect at Tier P0 for a single-return function.
func TestAnalyzeReturns_Direct_SingleReturn(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "SingleReturn")
	if fd == nil {
		t.Fatal("SingleReturn not found in returns package")
	}

	effects := analysis.AnalyzeReturns(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "SingleReturn")

	if count := countEffects(effects, taxonomy.ReturnValue); count != 1 {
		t.Errorf("SingleReturn: expected 1 ReturnValue, got %d", count)
	}
	for _, e := range effects {
		if e.Type == taxonomy.ReturnValue {
			if e.Tier != taxonomy.TierP0 {
				t.Errorf("ReturnValue tier: got %s, want P0", e.Tier)
			}
			if e.Description == "" {
				t.Error("ReturnValue description must not be empty")
			}
		}
	}
}

// TestAnalyzeReturns_Direct_MultipleReturns verifies that AnalyzeReturns
// detects 2 ReturnValue effects at Tier P0 for a two-return function.
func TestAnalyzeReturns_Direct_MultipleReturns(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "MultipleReturns")
	if fd == nil {
		t.Fatal("MultipleReturns not found in returns package")
	}

	effects := analysis.AnalyzeReturns(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "MultipleReturns")

	if count := countEffects(effects, taxonomy.ReturnValue); count != 2 {
		t.Errorf("MultipleReturns: expected 2 ReturnValue, got %d", count)
	}
	for _, e := range effects {
		if e.Type == taxonomy.ReturnValue {
			if e.Tier != taxonomy.TierP0 {
				t.Errorf("ReturnValue tier: got %s, want P0", e.Tier)
			}
			if e.Description == "" {
				t.Error("ReturnValue description must not be empty")
			}
		}
	}
}

// TestAnalyzeReturns_Direct_ErrorReturn verifies that AnalyzeReturns
// detects 1 ReturnValue and 1 ErrorReturn for a (T, error) function.
func TestAnalyzeReturns_Direct_ErrorReturn(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "ErrorReturn")
	if fd == nil {
		t.Fatal("ErrorReturn not found in returns package")
	}

	effects := analysis.AnalyzeReturns(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "ErrorReturn")

	if count := countEffects(effects, taxonomy.ReturnValue); count != 1 {
		t.Errorf("ErrorReturn: expected 1 ReturnValue (int), got %d", count)
	}
	if count := countEffects(effects, taxonomy.ErrorReturn); count != 1 {
		t.Errorf("ErrorReturn: expected 1 ErrorReturn, got %d", count)
	}
	for _, e := range effects {
		if e.Tier != taxonomy.TierP0 {
			t.Errorf("effect %s tier: got %s, want P0", e.Type, e.Tier)
		}
		if e.Description == "" {
			t.Errorf("effect %s description must not be empty", e.Type)
		}
	}
}

// TestAnalyzeReturns_Direct_NamedReturns verifies that AnalyzeReturns
// detects the correct effects for a function with named return values.
func TestAnalyzeReturns_Direct_NamedReturns(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "NamedReturns")
	if fd == nil {
		t.Fatal("NamedReturns not found in returns package")
	}

	effects := analysis.AnalyzeReturns(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "NamedReturns")

	if count := countEffects(effects, taxonomy.ReturnValue); count != 1 {
		t.Errorf("NamedReturns: expected 1 ReturnValue ([]byte), got %d", count)
	}
	if count := countEffects(effects, taxonomy.ErrorReturn); count != 1 {
		t.Errorf("NamedReturns: expected 1 ErrorReturn, got %d", count)
	}
	for _, e := range effects {
		if e.Type == taxonomy.ReturnValue || e.Type == taxonomy.ErrorReturn {
			if e.Tier != taxonomy.TierP0 {
				t.Errorf("effect %s tier: got %s, want P0", e.Type, e.Tier)
			}
			if e.Description == "" {
				t.Errorf("effect %s description must not be empty", e.Type)
			}
		}
	}
}

// TestAnalyzeReturns_Direct_NamedReturnModifiedInDefer verifies that
// AnalyzeReturns detects DeferredReturnMutation and ErrorReturn for a
// function with a named return modified in a defer statement.
func TestAnalyzeReturns_Direct_NamedReturnModifiedInDefer(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "NamedReturnModifiedInDefer")
	if fd == nil {
		t.Fatal("NamedReturnModifiedInDefer not found in returns package")
	}

	effects := analysis.AnalyzeReturns(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "NamedReturnModifiedInDefer")

	if !hasEffect(effects, taxonomy.DeferredReturnMutation) {
		t.Error("expected DeferredReturnMutation for named return 'err' modified in defer")
	}
	if !hasEffect(effects, taxonomy.ErrorReturn) {
		t.Error("expected ErrorReturn for NamedReturnModifiedInDefer")
	}
	for _, e := range effects {
		if e.Description == "" {
			t.Errorf("effect %s description must not be empty", e.Type)
		}
	}
}

// TestAnalyzeReturns_Direct_PureFunction verifies that AnalyzeReturns
// returns an empty slice for a function with no return values.
func TestAnalyzeReturns_Direct_PureFunction(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "PureFunction")
	if fd == nil {
		t.Fatal("PureFunction not found in returns package")
	}

	effects := analysis.AnalyzeReturns(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "PureFunction")

	if len(effects) != 0 {
		t.Errorf("PureFunction: expected empty slice, got %d effects: %v",
			len(effects), effects)
	}
}
