package analysis_test

import (
	"go/ast"
	"go/token"
	"testing"

	"github.com/unbound-force/gaze/internal/analysis"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// TestAnalyzeP1Effects_Direct_GlobalMutation verifies that AnalyzeP1Effects
// detects GlobalMutation for a function that assigns to a package-level variable.
func TestAnalyzeP1Effects_Direct_GlobalMutation(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "MutateGlobal")
	if fd == nil {
		t.Fatal("MutateGlobal not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "MutateGlobal")

	if !hasEffect(effects, taxonomy.GlobalMutation) {
		t.Error("expected GlobalMutation effect for MutateGlobal")
	}
	for _, e := range effects {
		if e.Type == taxonomy.GlobalMutation {
			if e.Tier != taxonomy.TierP1 {
				t.Errorf("GlobalMutation tier: got %s, want P1", e.Tier)
			}
			if e.Description == "" {
				t.Error("GlobalMutation description must not be empty")
			}
		}
	}
}

// TestAnalyzeP1Effects_Direct_ChannelSend verifies that AnalyzeP1Effects
// detects ChannelSend for a function that sends on a channel.
func TestAnalyzeP1Effects_Direct_ChannelSend(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "SendOnChannel")
	if fd == nil {
		t.Fatal("SendOnChannel not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "SendOnChannel")

	if !hasEffect(effects, taxonomy.ChannelSend) {
		t.Error("expected ChannelSend effect for SendOnChannel")
	}
	for _, e := range effects {
		if e.Type == taxonomy.ChannelSend {
			if e.Tier != taxonomy.TierP1 {
				t.Errorf("ChannelSend tier: got %s, want P1", e.Tier)
			}
			if e.Description == "" {
				t.Error("ChannelSend description must not be empty")
			}
		}
	}
}

// TestAnalyzeP1Effects_Direct_ChannelClose verifies that AnalyzeP1Effects
// detects ChannelClose for a function that closes a channel.
func TestAnalyzeP1Effects_Direct_ChannelClose(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "CloseChannel")
	if fd == nil {
		t.Fatal("CloseChannel not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "CloseChannel")

	if !hasEffect(effects, taxonomy.ChannelClose) {
		t.Error("expected ChannelClose effect for CloseChannel")
	}
	for _, e := range effects {
		if e.Type == taxonomy.ChannelClose {
			if e.Tier != taxonomy.TierP1 {
				t.Errorf("ChannelClose tier: got %s, want P1", e.Tier)
			}
			if e.Description == "" {
				t.Error("ChannelClose description must not be empty")
			}
		}
	}
}

// TestAnalyzeP1Effects_Direct_WriterOutput verifies that AnalyzeP1Effects
// detects WriterOutput for a function that calls Write on an io.Writer.
func TestAnalyzeP1Effects_Direct_WriterOutput(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "WriteToWriter")
	if fd == nil {
		t.Fatal("WriteToWriter not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteToWriter")

	if !hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("expected WriterOutput effect for WriteToWriter")
	}
	for _, e := range effects {
		if e.Type == taxonomy.WriterOutput {
			if e.Tier != taxonomy.TierP1 {
				t.Errorf("WriterOutput tier: got %s, want P1", e.Tier)
			}
			if e.Description == "" {
				t.Error("WriterOutput description must not be empty")
			}
		}
	}
}

// TestAnalyzeP1Effects_Direct_HTTPResponseWrite verifies that AnalyzeP1Effects
// detects HTTPResponseWrite for a function that writes to an http.ResponseWriter.
func TestAnalyzeP1Effects_Direct_HTTPResponseWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "HandleHTTP")
	if fd == nil {
		t.Fatal("HandleHTTP not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "HandleHTTP")

	if !hasEffect(effects, taxonomy.HTTPResponseWrite) {
		t.Error("expected HTTPResponseWrite effect for HandleHTTP")
	}
	for _, e := range effects {
		if e.Type == taxonomy.HTTPResponseWrite {
			if e.Tier != taxonomy.TierP1 {
				t.Errorf("HTTPResponseWrite tier: got %s, want P1", e.Tier)
			}
			if e.Description == "" {
				t.Error("HTTPResponseWrite description must not be empty")
			}
		}
	}
}

// TestAnalyzeP1Effects_Direct_MapMutation verifies that AnalyzeP1Effects
// detects MapMutation for a function that assigns to a map index.
func TestAnalyzeP1Effects_Direct_MapMutation(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "WriteToMap")
	if fd == nil {
		t.Fatal("WriteToMap not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteToMap")

	if !hasEffect(effects, taxonomy.MapMutation) {
		t.Error("expected MapMutation effect for WriteToMap")
	}
	for _, e := range effects {
		if e.Type == taxonomy.MapMutation {
			if e.Tier != taxonomy.TierP1 {
				t.Errorf("MapMutation tier: got %s, want P1", e.Tier)
			}
			if e.Description == "" {
				t.Error("MapMutation description must not be empty")
			}
		}
	}
}

// TestAnalyzeP1Effects_Direct_SliceMutation verifies that AnalyzeP1Effects
// detects SliceMutation for a function that assigns to a slice index.
func TestAnalyzeP1Effects_Direct_SliceMutation(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "WriteToSlice")
	if fd == nil {
		t.Fatal("WriteToSlice not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteToSlice")

	if !hasEffect(effects, taxonomy.SliceMutation) {
		t.Error("expected SliceMutation effect for WriteToSlice")
	}
	for _, e := range effects {
		if e.Type == taxonomy.SliceMutation {
			if e.Tier != taxonomy.TierP1 {
				t.Errorf("SliceMutation tier: got %s, want P1", e.Tier)
			}
			if e.Description == "" {
				t.Error("SliceMutation description must not be empty")
			}
		}
	}
}

// TestAnalyzeP1Effects_Direct_PureFunction verifies that AnalyzeP1Effects
// returns an empty slice for a function with no P1 side effects.
func TestAnalyzeP1Effects_Direct_PureFunction(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "PureP1")
	if fd == nil {
		t.Fatal("PureP1 not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "PureP1")

	if len(effects) != 0 {
		t.Errorf("PureP1: expected 0 effects, got %d: %v", len(effects), effects)
	}
}

// TestAnalyzeP1Effects_Direct_NilBody verifies that AnalyzeP1Effects
// handles a FuncDecl with nil Body gracefully (returns empty slice, no panic).
func TestAnalyzeP1Effects_Direct_NilBody(t *testing.T) {
	fd := &ast.FuncDecl{
		Name: ast.NewIdent("NilBodyFunc"),
		Type: &ast.FuncType{},
		Body: nil,
	}

	effects := analysis.AnalyzeP1Effects(token.NewFileSet(), nil, fd, "test/pkg", "NilBodyFunc")

	if len(effects) != 0 {
		t.Errorf("nil body: expected empty slice, got %d effects", len(effects))
	}
}
