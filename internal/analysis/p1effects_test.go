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
	// http.ResponseWriter embeds io.Writer, but the more specific
	// HTTPResponseWrite classification must take precedence — no
	// duplicate WriterOutput should be emitted (issue #131).
	if hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("http.ResponseWriter.Write must not produce WriterOutput (HTTPResponseWrite takes precedence)")
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
// detects exactly one MapMutation with the correct target for a function
// that assigns to a map index parameter.
func TestAnalyzeP1Effects_Direct_MapMutation(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "WriteToMap")
	if fd == nil {
		t.Fatal("WriteToMap not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteToMap")

	if count := countEffects(effects, taxonomy.MapMutation); count != 1 {
		t.Fatalf("expected exactly 1 MapMutation, got %d", count)
	}
	e := effectWithTarget(effects, taxonomy.MapMutation, "m")
	if e == nil {
		t.Fatal("expected MapMutation with target \"m\"")
	}
	if e.Tier != taxonomy.TierP1 {
		t.Errorf("MapMutation tier: got %s, want P1", e.Tier)
	}
	if e.Description == "" {
		t.Error("MapMutation description must not be empty")
	}
}

// TestAnalyzeP1Effects_Direct_SliceMutation verifies that AnalyzeP1Effects
// detects exactly one SliceMutation with the correct target for a function
// that assigns to a slice index parameter.
func TestAnalyzeP1Effects_Direct_SliceMutation(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "WriteToSlice")
	if fd == nil {
		t.Fatal("WriteToSlice not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteToSlice")

	if count := countEffects(effects, taxonomy.SliceMutation); count != 1 {
		t.Fatalf("expected exactly 1 SliceMutation, got %d", count)
	}
	e := effectWithTarget(effects, taxonomy.SliceMutation, "s")
	if e == nil {
		t.Fatal("expected SliceMutation with target \"s\"")
	}
	if e.Tier != taxonomy.TierP1 {
		t.Errorf("SliceMutation tier: got %s, want P1", e.Tier)
	}
	if e.Description == "" {
		t.Error("SliceMutation description must not be empty")
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

// --- Local Variable False Positive Tests ---

// TestAnalyzeP1Effects_Direct_LocalMapWrite verifies that a function
// creating a local map, writing to it, and returning it produces zero
// P1 effects. Also verifies the existing WriteToMap (parameter) still
// produces MapMutation as a regression check.
func TestAnalyzeP1Effects_Direct_LocalMapWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")

	// Negative: local map should produce zero P1 effects.
	fd := analysis.FindFuncDecl(pkg, "LocalMapWrite")
	if fd == nil {
		t.Fatal("LocalMapWrite not found in p1effects package")
	}
	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "LocalMapWrite")
	if len(effects) != 0 {
		t.Errorf("LocalMapWrite: expected 0 P1 effects, got %d: %v", len(effects), effects)
	}

	// Positive regression: parameter map still produces MapMutation.
	fd = analysis.FindFuncDecl(pkg, "WriteToMap")
	if fd == nil {
		t.Fatal("WriteToMap not found in p1effects package")
	}
	effects = analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteToMap")
	if !hasEffect(effects, taxonomy.MapMutation) {
		t.Error("WriteToMap: expected MapMutation effect (regression)")
	}
}

// TestAnalyzeP1Effects_Direct_LocalSliceWrite verifies that a function
// creating a local slice, writing to it, and returning it produces zero
// P1 effects. Also verifies the existing WriteToSlice (parameter) still
// produces SliceMutation as a regression check.
func TestAnalyzeP1Effects_Direct_LocalSliceWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")

	// Negative: local slice should produce zero P1 effects.
	fd := analysis.FindFuncDecl(pkg, "LocalSliceWrite")
	if fd == nil {
		t.Fatal("LocalSliceWrite not found in p1effects package")
	}
	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "LocalSliceWrite")
	if len(effects) != 0 {
		t.Errorf("LocalSliceWrite: expected 0 P1 effects, got %d: %v", len(effects), effects)
	}

	// Positive regression: parameter slice still produces SliceMutation.
	fd = analysis.FindFuncDecl(pkg, "WriteToSlice")
	if fd == nil {
		t.Fatal("WriteToSlice not found in p1effects package")
	}
	effects = analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteToSlice")
	if !hasEffect(effects, taxonomy.SliceMutation) {
		t.Error("WriteToSlice: expected SliceMutation effect (regression)")
	}
}

// TestAnalyzeP1Effects_Direct_LocalChannelSend verifies that a function
// creating a local buffered channel, sending on it, and receiving from
// it produces zero P1 effects. Also verifies the existing SendOnChannel
// (parameter) still produces ChannelSend as a regression check.
func TestAnalyzeP1Effects_Direct_LocalChannelSend(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")

	// Negative: local channel should produce zero P1 effects.
	fd := analysis.FindFuncDecl(pkg, "LocalChannelSend")
	if fd == nil {
		t.Fatal("LocalChannelSend not found in p1effects package")
	}
	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "LocalChannelSend")
	if len(effects) != 0 {
		t.Errorf("LocalChannelSend: expected 0 P1 effects, got %d: %v", len(effects), effects)
	}

	// Positive regression: parameter channel still produces ChannelSend.
	fd = analysis.FindFuncDecl(pkg, "SendOnChannel")
	if fd == nil {
		t.Fatal("SendOnChannel not found in p1effects package")
	}
	effects = analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "SendOnChannel")
	if !hasEffect(effects, taxonomy.ChannelSend) {
		t.Error("SendOnChannel: expected ChannelSend effect (regression)")
	}
}

// TestAnalyzeP1Effects_Direct_LocalChannelClose verifies that a function
// creating a local channel and closing it produces zero P1 effects. Also
// verifies the existing CloseChannel (parameter) still produces
// ChannelClose as a regression check.
func TestAnalyzeP1Effects_Direct_LocalChannelClose(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")

	// Negative: local channel close should produce zero P1 effects.
	fd := analysis.FindFuncDecl(pkg, "LocalChannelClose")
	if fd == nil {
		t.Fatal("LocalChannelClose not found in p1effects package")
	}
	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "LocalChannelClose")
	if len(effects) != 0 {
		t.Errorf("LocalChannelClose: expected 0 P1 effects, got %d: %v", len(effects), effects)
	}

	// Positive regression: parameter channel still produces ChannelClose.
	fd = analysis.FindFuncDecl(pkg, "CloseChannel")
	if fd == nil {
		t.Fatal("CloseChannel not found in p1effects package")
	}
	effects = analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "CloseChannel")
	if !hasEffect(effects, taxonomy.ChannelClose) {
		t.Error("CloseChannel: expected ChannelClose effect (regression)")
	}
}

// TestAnalyzeP1Effects_Direct_NamedReturnMapWrite verifies that a
// function with a named return of map type produces MapMutation when
// writing to it. Named returns are part of the function signature and
// are externally observable.
func TestAnalyzeP1Effects_Direct_NamedReturnMapWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "NamedReturnMapWrite")
	if fd == nil {
		t.Fatal("NamedReturnMapWrite not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "NamedReturnMapWrite")

	if len(effects) != 1 {
		t.Fatalf("NamedReturnMapWrite: expected 1 effect, got %d: %v", len(effects), effects)
	}
	if !hasEffect(effects, taxonomy.MapMutation) {
		t.Error("NamedReturnMapWrite: expected MapMutation effect for named return")
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

// TestAnalyzeP1Effects_Direct_NonWriterWrite verifies that a type with
// a Write method that does NOT match the io.Writer signature does NOT
// produce WriterOutput. Notifier.Write(string) is not io.Writer.Write([]byte)(int, error).
// Regression test for issue #109.
func TestAnalyzeP1Effects_Direct_NonWriterWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "SendNote")
	if fd == nil {
		t.Fatal("SendNote not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "SendNote")

	if hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("SendNote must not produce WriterOutput — Notifier.Write(string) is not io.Writer")
	}
}

// TestAnalyzeP1Effects_Direct_CustomResponseWriter verifies that a
// custom struct implementing http.ResponseWriter produces
// HTTPResponseWrite, not WriterOutput. Tests that isHTTPResponseWriter
// correctly identifies custom implementations, not just the named type.
// Regression test for issue #132.
func TestAnalyzeP1Effects_Direct_CustomResponseWriter(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "HandleCustomRW")
	if fd == nil {
		t.Fatal("HandleCustomRW not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "HandleCustomRW")

	if !hasEffect(effects, taxonomy.HTTPResponseWrite) {
		t.Error("expected HTTPResponseWrite effect for HandleCustomRW")
	}
	if hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("HandleCustomRW must not produce WriterOutput — HTTPResponseWrite takes precedence")
	}
}

// TestAnalyzeP1Effects_Direct_WriteToStructMap verifies that a method
// writing to a map field on its receiver produces MapMutation. The
// receiver is externally observable, and unwrapToIdent resolves the
// SelectorExpr (c.M) to the base identifier (c).
func TestAnalyzeP1Effects_Direct_WriteToStructMap(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindMethodDecl(pkg, "*Container", "WriteToStructMap")
	if fd == nil {
		t.Fatal("WriteToStructMap not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "WriteToStructMap")

	if len(effects) != 1 {
		t.Fatalf("WriteToStructMap: expected 1 effect, got %d: %v", len(effects), effects)
	}
	if !hasEffect(effects, taxonomy.MapMutation) {
		t.Error("WriteToStructMap: expected MapMutation effect for receiver field map write")
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

// TestAnalyzeP1Effects_PartialResponseWriter verifies that a type with
// Write + Header but NOT WriteHeader produces WriterOutput (it satisfies
// io.Writer) but does NOT produce HTTPResponseWrite.
func TestAnalyzeP1Effects_PartialResponseWriter(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "HandlePartialRW")
	if fd == nil {
		t.Fatal("HandlePartialRW not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "HandlePartialRW")

	if hasEffect(effects, taxonomy.HTTPResponseWrite) {
		t.Error("PartialResponseWriter (missing WriteHeader) must not produce HTTPResponseWrite")
	}
	if !hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("PartialResponseWriter has valid Write([]byte)(int,error) — expected WriterOutput")
	}
}

// TestAnalyzeP1Effects_WrongWriteReturn verifies that a type whose Write
// method returns only int (not (int, error)) is not classified as either
// io.Writer or http.ResponseWriter.
func TestAnalyzeP1Effects_WrongWriteReturn(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "HandleWrongWriteReturn")
	if fd == nil {
		t.Fatal("HandleWrongWriteReturn not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "HandleWrongWriteReturn")

	if len(effects) != 0 {
		t.Errorf("expected no effects for WrongWriteReturn, got %d: %v", len(effects), effects)
	}
	if hasEffect(effects, taxonomy.HTTPResponseWrite) {
		t.Error("WrongWriteReturn must not produce HTTPResponseWrite — Write signature mismatch")
	}
	if hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("WrongWriteReturn must not produce WriterOutput — Write([]byte) int is not io.Writer")
	}
}

// TestAnalyzeP1Effects_WrongParamType verifies that a type whose Write
// method accepts string (not []byte) is not classified as io.Writer or
// http.ResponseWriter.
func TestAnalyzeP1Effects_WrongParamType(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "HandleWrongParamType")
	if fd == nil {
		t.Fatal("HandleWrongParamType not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "HandleWrongParamType")

	if len(effects) != 0 {
		t.Errorf("expected no effects for WrongParamType, got %d: %v", len(effects), effects)
	}
	if hasEffect(effects, taxonomy.HTTPResponseWrite) {
		t.Error("WrongParamType must not produce HTTPResponseWrite — Write(string) is not Write([]byte)")
	}
	if hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("WrongParamType must not produce WriterOutput — Write(string) is not io.Writer")
	}
}

// TestAnalyzeP1Effects_WrongIntReturn64 verifies that a type whose Write
// method returns (int64, error) instead of (int, error) is not classified
// as io.Writer or http.ResponseWriter.
func TestAnalyzeP1Effects_WrongIntReturn64(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "HandleWrongIntReturn64")
	if fd == nil {
		t.Fatal("HandleWrongIntReturn64 not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "HandleWrongIntReturn64")

	if len(effects) != 0 {
		t.Errorf("expected no effects for WrongIntReturn64, got %d: %v", len(effects), effects)
	}
	if hasEffect(effects, taxonomy.HTTPResponseWrite) {
		t.Error("WrongIntReturn64 must not produce HTTPResponseWrite — Write returns int64, not int")
	}
	if hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("WrongIntReturn64 must not produce WriterOutput — Write([]byte)(int64,error) is not io.Writer")
	}
}

// TestAnalyzeP1Effects_ExtraParamWrite verifies that a type whose Write
// method takes an extra parameter is not classified as io.Writer or
// http.ResponseWriter.
func TestAnalyzeP1Effects_ExtraParamWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "HandleExtraParamWrite")
	if fd == nil {
		t.Fatal("HandleExtraParamWrite not found in p1effects package")
	}

	effects := analysis.AnalyzeP1Effects(pkg.Fset, pkg.TypesInfo, fd, pkg.PkgPath, "HandleExtraParamWrite")

	if len(effects) != 0 {
		t.Errorf("expected no effects for ExtraParamWrite, got %d: %v", len(effects), effects)
	}
	if hasEffect(effects, taxonomy.HTTPResponseWrite) {
		t.Error("ExtraParamWrite must not produce HTTPResponseWrite — Write has extra param")
	}
	if hasEffect(effects, taxonomy.WriterOutput) {
		t.Error("ExtraParamWrite must not produce WriterOutput — Write([]byte, int) is not io.Writer")
	}
}
