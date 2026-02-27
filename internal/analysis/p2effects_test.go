package analysis_test

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// TestAnalyzeP2Effects_Direct_GoroutineSpawn verifies that AnalyzeP2Effects
// detects GoroutineSpawn for a function containing a go statement.
func TestAnalyzeP2Effects_Direct_GoroutineSpawn(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "SpawnGoroutine")
	if fd == nil {
		t.Fatal("SpawnGoroutine not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "SpawnGoroutine")

	if !hasEffect(effects, taxonomy.GoroutineSpawn) {
		t.Error("expected GoroutineSpawn effect for SpawnGoroutine")
	}
	for _, e := range effects {
		if e.Type == taxonomy.GoroutineSpawn {
			if e.Tier != taxonomy.TierP2 {
				t.Errorf("GoroutineSpawn tier: got %s, want P2", e.Tier)
			}
			if e.Description == "" {
				t.Error("GoroutineSpawn description must not be empty")
			}
		}
	}
}

// TestAnalyzeP2Effects_Direct_Panic verifies that AnalyzeP2Effects
// detects Panic for a function that calls the builtin panic().
func TestAnalyzeP2Effects_Direct_Panic(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "PanicWithString")
	if fd == nil {
		t.Fatal("PanicWithString not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "PanicWithString")

	if !hasEffect(effects, taxonomy.Panic) {
		t.Error("expected Panic effect for PanicWithString")
	}
	for _, e := range effects {
		if e.Type == taxonomy.Panic {
			if e.Tier != taxonomy.TierP2 {
				t.Errorf("Panic tier: got %s, want P2", e.Tier)
			}
			if e.Description == "" {
				t.Error("Panic description must not be empty")
			}
		}
	}
}

// TestAnalyzeP2Effects_Direct_FileSystemWrite verifies that AnalyzeP2Effects
// detects FileSystemWrite for a function that calls os.WriteFile.
func TestAnalyzeP2Effects_Direct_FileSystemWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "WriteFileOS")
	if fd == nil {
		t.Fatal("WriteFileOS not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteFileOS")

	if !hasEffect(effects, taxonomy.FileSystemWrite) {
		t.Error("expected FileSystemWrite effect for WriteFileOS")
	}
	for _, e := range effects {
		if e.Type == taxonomy.FileSystemWrite {
			if e.Tier != taxonomy.TierP2 {
				t.Errorf("FileSystemWrite tier: got %s, want P2", e.Tier)
			}
			if e.Description == "" {
				t.Error("FileSystemWrite description must not be empty")
			}
		}
	}
}

// TestAnalyzeP2Effects_Direct_FileSystemDelete verifies that AnalyzeP2Effects
// detects FileSystemDelete for a function that calls os.Remove.
func TestAnalyzeP2Effects_Direct_FileSystemDelete(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "RemoveFile")
	if fd == nil {
		t.Fatal("RemoveFile not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "RemoveFile")

	if !hasEffect(effects, taxonomy.FileSystemDelete) {
		t.Error("expected FileSystemDelete effect for RemoveFile")
	}
	for _, e := range effects {
		if e.Type == taxonomy.FileSystemDelete {
			if e.Tier != taxonomy.TierP2 {
				t.Errorf("FileSystemDelete tier: got %s, want P2", e.Tier)
			}
			if e.Description == "" {
				t.Error("FileSystemDelete description must not be empty")
			}
		}
	}
}

// TestAnalyzeP2Effects_Direct_LogWrite verifies that AnalyzeP2Effects
// detects LogWrite for a function that calls log.Println.
func TestAnalyzeP2Effects_Direct_LogWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "LogPrint")
	if fd == nil {
		t.Fatal("LogPrint not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "LogPrint")

	if !hasEffect(effects, taxonomy.LogWrite) {
		t.Error("expected LogWrite effect for LogPrint")
	}
	for _, e := range effects {
		if e.Type == taxonomy.LogWrite {
			if e.Tier != taxonomy.TierP2 {
				t.Errorf("LogWrite tier: got %s, want P2", e.Tier)
			}
			if e.Description == "" {
				t.Error("LogWrite description must not be empty")
			}
		}
	}
}

// TestAnalyzeP2Effects_Direct_ContextCancellation verifies that AnalyzeP2Effects
// detects ContextCancellation for a function that calls context.WithCancel.
func TestAnalyzeP2Effects_Direct_ContextCancellation(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "CancelContext")
	if fd == nil {
		t.Fatal("CancelContext not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "CancelContext")

	if !hasEffect(effects, taxonomy.ContextCancellation) {
		t.Error("expected ContextCancellation effect for CancelContext")
	}
	for _, e := range effects {
		if e.Type == taxonomy.ContextCancellation {
			if e.Tier != taxonomy.TierP2 {
				t.Errorf("ContextCancellation tier: got %s, want P2", e.Tier)
			}
			if e.Description == "" {
				t.Error("ContextCancellation description must not be empty")
			}
		}
	}
}

// TestAnalyzeP2Effects_Direct_CallbackInvocation verifies that AnalyzeP2Effects
// detects CallbackInvocation for a function that calls a function-typed parameter.
func TestAnalyzeP2Effects_Direct_CallbackInvocation(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "InvokeCallback")
	if fd == nil {
		t.Fatal("InvokeCallback not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "InvokeCallback")

	if !hasEffect(effects, taxonomy.CallbackInvocation) {
		t.Error("expected CallbackInvocation effect for InvokeCallback")
	}
	for _, e := range effects {
		if e.Type == taxonomy.CallbackInvocation {
			if e.Tier != taxonomy.TierP2 {
				t.Errorf("CallbackInvocation tier: got %s, want P2", e.Tier)
			}
			if e.Description == "" {
				t.Error("CallbackInvocation description must not be empty")
			}
		}
	}
}

// TestAnalyzeP2Effects_Direct_DatabaseWrite verifies that AnalyzeP2Effects
// detects DatabaseWrite for a function that calls db.Exec.
func TestAnalyzeP2Effects_Direct_DatabaseWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "DBExec")
	if fd == nil {
		t.Fatal("DBExec not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "DBExec")

	if !hasEffect(effects, taxonomy.DatabaseWrite) {
		t.Error("expected DatabaseWrite effect for DBExec")
	}
	for _, e := range effects {
		if e.Type == taxonomy.DatabaseWrite {
			if e.Tier != taxonomy.TierP2 {
				t.Errorf("DatabaseWrite tier: got %s, want P2", e.Tier)
			}
			if e.Description == "" {
				t.Error("DatabaseWrite description must not be empty")
			}
		}
	}
}

// TestAnalyzeP2Effects_Direct_PureFunction verifies that AnalyzeP2Effects
// returns an empty slice for a function with no P2 side effects.
func TestAnalyzeP2Effects_Direct_PureFunction(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "PureP2")
	if fd == nil {
		t.Fatal("PureP2 not found in p2effects package")
	}

	effects := analysis.AnalyzeP2Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "PureP2")

	if len(effects) != 0 {
		t.Errorf("PureP2: expected 0 effects, got %d: %v", len(effects), effects)
	}
}

// TestAnalyzeP2Effects_Direct_NilBody verifies that AnalyzeP2Effects
// handles a FuncDecl with nil Body gracefully (returns empty slice, no panic).
func TestAnalyzeP2Effects_Direct_NilBody(t *testing.T) {
	fd := &ast.FuncDecl{
		Name: ast.NewIdent("NilBodyFunc"),
		Type: &ast.FuncType{},
		Body: nil,
	}

	effects := analysis.AnalyzeP2Effects(token.NewFileSet(), nil, fd, "test/pkg", "NilBodyFunc")

	if len(effects) != 0 {
		t.Errorf("nil body: expected empty slice, got %d effects", len(effects))
	}
}
