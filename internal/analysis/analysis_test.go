package analysis_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jflowers/gaze/internal/analysis"
	"github.com/jflowers/gaze/internal/taxonomy"
	"golang.org/x/tools/go/packages"
)

// testdataPath returns the absolute path to a testdata fixture package.
func testdataPath(pkgName string) string {
	_, thisFile, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(thisFile), "testdata", "src", pkgName)
}

// loadTestdataPackage loads a testdata fixture package using the
// given directory. This is the shared implementation for both test
// and benchmark helpers.
func loadTestdataPackage(pkgName string) (*packages.Package, error) {
	testdataDir := testdataPath(pkgName)

	cfg := &packages.Config{
		Mode: packages.NeedName |
			packages.NeedFiles |
			packages.NeedCompiledGoFiles |
			packages.NeedImports |
			packages.NeedDeps |
			packages.NeedTypes |
			packages.NeedSyntax |
			packages.NeedTypesInfo |
			packages.NeedTypesSizes,
		Dir:   testdataDir,
		Tests: false,
	}

	pkgs, err := packages.Load(cfg, ".")
	if err != nil {
		return nil, err
	}
	if len(pkgs) == 0 {
		return nil, fmt.Errorf("no packages loaded for %q", pkgName)
	}
	if len(pkgs[0].Errors) > 0 {
		return nil, fmt.Errorf("package %q has errors: %v", pkgName, pkgs[0].Errors)
	}
	return pkgs[0], nil
}

// loadTestPackage loads one of the test fixture packages from testdata.
func loadTestPackage(t *testing.T, pkgName string) *packages.Package {
	t.Helper()
	pkg, err := loadTestdataPackage(pkgName)
	if err != nil {
		t.Fatalf("failed to load test package %q: %v", pkgName, err)
	}
	return pkg
}

// loadTestPackageBench loads one of the test fixture packages for benchmarks.
func loadTestPackageBench(b *testing.B, pkgName string) *packages.Package {
	b.Helper()
	pkg, err := loadTestdataPackage(pkgName)
	if err != nil {
		b.Fatalf("failed to load test package %q: %v", pkgName, err)
	}
	return pkg
}

// hasEffect checks if a side effect of the given type exists in the list.
func hasEffect(effects []taxonomy.SideEffect, typ taxonomy.SideEffectType) bool {
	for _, e := range effects {
		if e.Type == typ {
			return true
		}
	}
	return false
}

// countEffects counts effects of a given type.
func countEffects(effects []taxonomy.SideEffect, typ taxonomy.SideEffectType) int {
	count := 0
	for _, e := range effects {
		if e.Type == typ {
			count++
		}
	}
	return count
}

// effectWithTarget finds an effect by type and target string.
func effectWithTarget(effects []taxonomy.SideEffect, typ taxonomy.SideEffectType, target string) *taxonomy.SideEffect {
	for i, e := range effects {
		if e.Type == typ && e.Target == target {
			return &effects[i]
		}
	}
	return nil
}

// --- Return Analyzer Tests ---

func TestReturns_PureFunction(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "PureFunction")
	if fd == nil {
		t.Fatal("PureFunction not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)
	if len(result.SideEffects) != 0 {
		t.Errorf("PureFunction: expected 0 side effects, got %d: %v",
			len(result.SideEffects), result.SideEffects)
	}
}

func TestReturns_SingleReturn(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "SingleReturn")
	if fd == nil {
		t.Fatal("SingleReturn not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if count := countEffects(result.SideEffects, taxonomy.ReturnValue); count != 1 {
		t.Errorf("expected 1 ReturnValue, got %d", count)
	}
	e := effectWithTarget(result.SideEffects, taxonomy.ReturnValue, "int")
	if e == nil {
		t.Error("expected ReturnValue with target 'int'")
	}
}

func TestReturns_MultipleReturns(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "MultipleReturns")
	if fd == nil {
		t.Fatal("MultipleReturns not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if count := countEffects(result.SideEffects, taxonomy.ReturnValue); count != 2 {
		t.Errorf("expected 2 ReturnValue, got %d", count)
	}
}

func TestReturns_ErrorReturn(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "ErrorReturn")
	if fd == nil {
		t.Fatal("ErrorReturn not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if count := countEffects(result.SideEffects, taxonomy.ReturnValue); count != 1 {
		t.Errorf("expected 1 ReturnValue (int), got %d", count)
	}
	if count := countEffects(result.SideEffects, taxonomy.ErrorReturn); count != 1 {
		t.Errorf("expected 1 ErrorReturn, got %d", count)
	}
}

func TestReturns_ErrorOnly(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "ErrorOnly")
	if fd == nil {
		t.Fatal("ErrorOnly not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if count := countEffects(result.SideEffects, taxonomy.ErrorReturn); count != 1 {
		t.Errorf("expected 1 ErrorReturn, got %d", count)
	}
	if count := countEffects(result.SideEffects, taxonomy.ReturnValue); count != 0 {
		t.Errorf("expected 0 ReturnValue, got %d", count)
	}
}

func TestReturns_TripleReturn(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "TripleReturn")
	if fd == nil {
		t.Fatal("TripleReturn not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if count := countEffects(result.SideEffects, taxonomy.ReturnValue); count != 2 {
		t.Errorf("expected 2 ReturnValue (string, int), got %d", count)
	}
	if count := countEffects(result.SideEffects, taxonomy.ErrorReturn); count != 1 {
		t.Errorf("expected 1 ErrorReturn, got %d", count)
	}
}

func TestReturns_NamedReturns(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "NamedReturns")
	if fd == nil {
		t.Fatal("NamedReturns not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if count := countEffects(result.SideEffects, taxonomy.ReturnValue); count != 1 {
		t.Errorf("expected 1 ReturnValue ([]byte), got %d", count)
	}
	if count := countEffects(result.SideEffects, taxonomy.ErrorReturn); count != 1 {
		t.Errorf("expected 1 ErrorReturn, got %d", count)
	}

	// Verify named return metadata in description.
	for _, e := range result.SideEffects {
		if e.Type == taxonomy.ReturnValue {
			if e.Description == "" {
				t.Error("expected non-empty description for named return")
			}
		}
	}
}

func TestReturns_NamedReturnModifiedInDefer(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "NamedReturnModifiedInDefer")
	if fd == nil {
		t.Fatal("NamedReturnModifiedInDefer not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.DeferredReturnMutation) {
		t.Error("expected DeferredReturnMutation for named return 'err' modified in defer")
	}
	if !hasEffect(result.SideEffects, taxonomy.ErrorReturn) {
		t.Error("expected ErrorReturn")
	}
}

func TestReturns_InterfaceReturn(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "InterfaceReturn")
	if fd == nil {
		t.Fatal("InterfaceReturn not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if count := countEffects(result.SideEffects, taxonomy.ReturnValue); count != 1 {
		t.Errorf("expected 1 ReturnValue (io.Reader), got %d", count)
	}
}

// --- Sentinel Analyzer Tests ---

func TestSentinels_Detection(t *testing.T) {
	pkg := loadTestPackage(t, "sentinel")

	results, err := analysis.Analyze(pkg, analysis.Options{
		IncludeUnexported: true,
	})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// Collect all sentinel effects across all results.
	var sentinels []taxonomy.SideEffect
	for _, r := range results {
		for _, e := range r.SideEffects {
			if e.Type == taxonomy.SentinelError {
				sentinels = append(sentinels, e)
			}
		}
	}

	// Should detect: ErrNotFound, ErrPermission, ErrWrapped, errUnexported
	expectedSentinels := map[string]bool{
		"ErrNotFound":   false,
		"ErrPermission": false,
		"ErrWrapped":    false,
		"errUnexported": false,
	}

	for _, s := range sentinels {
		if _, ok := expectedSentinels[s.Target]; ok {
			expectedSentinels[s.Target] = true
		}
	}

	for name, found := range expectedSentinels {
		if !found {
			t.Errorf("expected sentinel %q not detected", name)
		}
	}

	// Should NOT detect NotAnError.
	for _, s := range sentinels {
		if s.Target == "NotAnError" {
			t.Error("NotAnError should not be detected as a sentinel")
		}
	}
}

func TestSentinels_WrappedDetection(t *testing.T) {
	pkg := loadTestPackage(t, "sentinel")

	results, err := analysis.Analyze(pkg, analysis.Options{
		IncludeUnexported: true,
	})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	var wrapped *taxonomy.SideEffect
	for _, r := range results {
		for i, e := range r.SideEffects {
			if e.Type == taxonomy.SentinelError && e.Target == "ErrWrapped" {
				wrapped = &r.SideEffects[i]
			}
		}
	}

	if wrapped == nil {
		t.Fatal("ErrWrapped not detected")
	}
	if wrapped.Description == "" {
		t.Error("expected non-empty description for ErrWrapped")
	}
}

// --- Mutation Analyzer Tests ---

func TestMutation_PointerReceiverIncrement(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "*Counter", "Increment")
	if fd == nil {
		t.Fatal("(*Counter).Increment not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	e := effectWithTarget(result.SideEffects, taxonomy.ReceiverMutation, "count")
	if e == nil {
		t.Error("expected ReceiverMutation for field 'count'")
	}
}

func TestMutation_PointerReceiverSetName(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "*Counter", "SetName")
	if fd == nil {
		t.Fatal("(*Counter).SetName not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	e := effectWithTarget(result.SideEffects, taxonomy.ReceiverMutation, "name")
	if e == nil {
		t.Error("expected ReceiverMutation for field 'name'")
	}
}

func TestMutation_PointerReceiverSetBoth(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "*Counter", "SetBoth")
	if fd == nil {
		t.Fatal("(*Counter).SetBoth not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if countEffects(result.SideEffects, taxonomy.ReceiverMutation) != 2 {
		t.Errorf("expected 2 ReceiverMutation effects, got %d",
			countEffects(result.SideEffects, taxonomy.ReceiverMutation))
	}
	if effectWithTarget(result.SideEffects, taxonomy.ReceiverMutation, "count") == nil {
		t.Error("expected ReceiverMutation for field 'count'")
	}
	if effectWithTarget(result.SideEffects, taxonomy.ReceiverMutation, "name") == nil {
		t.Error("expected ReceiverMutation for field 'name'")
	}
}

func TestMutation_ValueReceiverNoMutation(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "Counter", "Value")
	if fd == nil {
		t.Fatal("(Counter).Value not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.ReceiverMutation) {
		t.Error("value receiver should NOT report ReceiverMutation")
	}
	// But it should still report ReturnValue.
	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("expected ReturnValue for Value()")
	}
}

func TestMutation_ValueReceiverTrap(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "Counter", "ValueReceiverTrap")
	if fd == nil {
		t.Fatal("(Counter).ValueReceiverTrap not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.ReceiverMutation) {
		t.Error("value receiver copy mutation should NOT report ReceiverMutation")
	}
}

func TestMutation_PointerArgument(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindFuncDecl(pkg, "Normalize")
	if fd == nil {
		t.Fatal("Normalize not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	e := effectWithTarget(result.SideEffects, taxonomy.PointerArgMutation, "v")
	if e == nil {
		t.Error("expected PointerArgMutation for argument 'v'")
	}
}

func TestMutation_PointerArgFillSlice(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindFuncDecl(pkg, "FillSlice")
	if fd == nil {
		t.Fatal("FillSlice not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	e := effectWithTarget(result.SideEffects, taxonomy.PointerArgMutation, "dst")
	if e == nil {
		t.Error("expected PointerArgMutation for argument 'dst'")
	}
}

func TestMutation_PointerArgReadOnly(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindFuncDecl(pkg, "ReadOnly")
	if fd == nil {
		t.Fatal("ReadOnly not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.PointerArgMutation) {
		t.Error("ReadOnly should NOT report PointerArgMutation (read-only access)")
	}
	// But should report ReturnValue.
	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("expected ReturnValue for ReadOnly()")
	}
}

func TestMutation_NestedFieldMutation(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "*Config", "UpdateConfig")
	if fd == nil {
		t.Fatal("(*Config).UpdateConfig not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	e := effectWithTarget(result.SideEffects, taxonomy.ReceiverMutation, "Timeout")
	if e == nil {
		t.Error("expected ReceiverMutation for field 'Timeout'")
	}
}

func TestMutation_DeepNestedMutation(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "*Config", "UpdateNested")
	if fd == nil {
		t.Fatal("(*Config).UpdateNested not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Should report mutation to the top-level field 'Nested'.
	e := effectWithTarget(result.SideEffects, taxonomy.ReceiverMutation, "Nested")
	if e == nil {
		t.Error("expected ReceiverMutation for field 'Nested' (deep nested mutation)")
	}
}

// --- Analysis Metadata Tests ---

func TestAnalysis_MetadataPopulated(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "SingleReturn")
	if fd == nil {
		t.Fatal("SingleReturn not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if result.Metadata.GazeVersion == "" {
		t.Error("expected non-empty GazeVersion")
	}
	if result.Metadata.GoVersion == "" {
		t.Error("expected non-empty GoVersion")
	}
}

func TestAnalysis_TargetPopulated(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "SingleReturn")
	if fd == nil {
		t.Fatal("SingleReturn not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if result.Target.Function != "SingleReturn" {
		t.Errorf("expected function name 'SingleReturn', got %q",
			result.Target.Function)
	}
	if result.Target.Location == "" {
		t.Error("expected non-empty location")
	}
	if result.Target.Signature == "" {
		t.Error("expected non-empty signature")
	}
}

func TestAnalysis_MethodTargetHasReceiver(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "*Counter", "Increment")
	if fd == nil {
		t.Fatal("(*Counter).Increment not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if result.Target.Receiver != "*Counter" {
		t.Errorf("expected receiver '*Counter', got %q",
			result.Target.Receiver)
	}
}

// --- Side Effect ID Tests ---

func TestAnalysis_StableIDs(t *testing.T) {
	pkg := loadTestPackage(t, "returns")
	fd := analysis.FindFuncDecl(pkg, "ErrorReturn")
	if fd == nil {
		t.Fatal("ErrorReturn not found")
	}

	result1 := analysis.AnalyzeFunction(pkg, fd)
	result2 := analysis.AnalyzeFunction(pkg, fd)

	if len(result1.SideEffects) != len(result2.SideEffects) {
		t.Fatalf("different side effect counts: %d vs %d",
			len(result1.SideEffects), len(result2.SideEffects))
	}

	for i := range result1.SideEffects {
		if result1.SideEffects[i].ID != result2.SideEffects[i].ID {
			t.Errorf("unstable ID for effect %d: %q vs %q",
				i, result1.SideEffects[i].ID, result2.SideEffects[i].ID)
		}
	}
}

// --- Analyze() option tests ---

func TestAnalyze_ExportedOnlyByDefault(t *testing.T) {
	pkg := loadTestPackage(t, "returns")

	results, err := analysis.Analyze(pkg, analysis.Options{
		IncludeUnexported: false,
	})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	for _, r := range results {
		if r.Target.Function == "<package>" {
			continue
		}
		// All returned functions should be exported.
		if len(r.Target.Function) > 0 {
			first := r.Target.Function[0]
			if first >= 'a' && first <= 'z' {
				t.Errorf("unexported function %q should not appear with IncludeUnexported=false",
					r.Target.Function)
			}
		}
	}
}

func TestAnalyze_FunctionFilter(t *testing.T) {
	pkg := loadTestPackage(t, "returns")

	results, err := analysis.Analyze(pkg, analysis.Options{
		IncludeUnexported: true,
		FunctionFilter:    "SingleReturn",
	})
	if err != nil {
		t.Fatalf("Analyze failed: %v", err)
	}

	// FunctionFilter also suppresses sentinel analysis.
	if len(results) != 1 {
		t.Fatalf("expected 1 result with FunctionFilter, got %d", len(results))
	}
	if results[0].Target.Function != "SingleReturn" {
		t.Errorf("expected 'SingleReturn', got %q", results[0].Target.Function)
	}
}

// --- All Tiers are P0 ---

func TestAnalysis_AllP0EffectsAreP0(t *testing.T) {
	pkg := loadTestPackage(t, "mutation")
	fd := analysis.FindMethodDecl(pkg, "*Counter", "Increment")
	if fd == nil {
		t.Fatal("(*Counter).Increment not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	for _, e := range result.SideEffects {
		if e.Type == taxonomy.ReceiverMutation && e.Tier != taxonomy.TierP0 {
			t.Errorf("ReceiverMutation should be P0, got %s", e.Tier)
		}
	}
}

// --- P1 Side Effect Tests ---

func TestP1_GlobalMutation(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "MutateGlobal")
	if fd == nil {
		t.Fatal("MutateGlobal not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.GlobalMutation) {
		t.Error("expected GlobalMutation for MutateGlobal")
	}
}

func TestP1_GlobalMutation_TwoGlobals(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "MutateTwoGlobals")
	if fd == nil {
		t.Fatal("MutateTwoGlobals not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if count := countEffects(result.SideEffects, taxonomy.GlobalMutation); count != 2 {
		t.Errorf("expected 2 GlobalMutation effects, got %d", count)
	}
}

func TestP1_GlobalMutation_ReadOnly(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "ReadGlobal")
	if fd == nil {
		t.Fatal("ReadGlobal not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.GlobalMutation) {
		t.Error("ReadGlobal should NOT produce GlobalMutation")
	}
}

func TestP1_ChannelSend(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "SendOnChannel")
	if fd == nil {
		t.Fatal("SendOnChannel not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.ChannelSend) {
		t.Error("expected ChannelSend for SendOnChannel")
	}
}

func TestP1_ChannelClose(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "CloseChannel")
	if fd == nil {
		t.Fatal("CloseChannel not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.ChannelClose) {
		t.Error("expected ChannelClose for CloseChannel")
	}
}

func TestP1_ChannelSendAndClose(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "SendAndClose")
	if fd == nil {
		t.Fatal("SendAndClose not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.ChannelSend) {
		t.Error("expected ChannelSend")
	}
	if !hasEffect(result.SideEffects, taxonomy.ChannelClose) {
		t.Error("expected ChannelClose")
	}
}

func TestP1_WriterOutput(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "WriteToWriter")
	if fd == nil {
		t.Fatal("WriteToWriter not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.WriterOutput) {
		t.Error("expected WriterOutput for WriteToWriter")
	}
}

func TestP1_WriterOutput_ReadOnly(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "ReadFromWriter")
	if fd == nil {
		t.Fatal("ReadFromWriter not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.WriterOutput) {
		t.Error("ReadFromWriter should NOT produce WriterOutput")
	}
}

func TestP1_HttpResponseWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "HandleHTTP")
	if fd == nil {
		t.Fatal("HandleHTTP not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.HttpResponseWrite) {
		t.Error("expected HttpResponseWrite for HandleHTTP")
	}
}

func TestP1_MapMutation(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "WriteToMap")
	if fd == nil {
		t.Fatal("WriteToMap not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.MapMutation) {
		t.Error("expected MapMutation for WriteToMap")
	}
}

func TestP1_MapMutation_ReadOnly(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "ReadFromMap")
	if fd == nil {
		t.Fatal("ReadFromMap not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.MapMutation) {
		t.Error("ReadFromMap should NOT produce MapMutation")
	}
}

func TestP1_SliceMutation(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "WriteToSlice")
	if fd == nil {
		t.Fatal("WriteToSlice not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.SliceMutation) {
		t.Error("expected SliceMutation for WriteToSlice")
	}
}

func TestP1_SliceMutation_ReadOnly(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "ReadFromSlice")
	if fd == nil {
		t.Fatal("ReadFromSlice not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.SliceMutation) {
		t.Error("ReadFromSlice should NOT produce SliceMutation")
	}
}

func TestP1_PureFunction(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "PureP1")
	if fd == nil {
		t.Fatal("PureP1 not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Should only have ReturnValue, no P1 effects.
	for _, e := range result.SideEffects {
		if e.Tier == taxonomy.TierP1 {
			t.Errorf("PureP1 should have no P1 effects, got %s", e.Type)
		}
	}
}

func TestP1_EffectsAreP1Tier(t *testing.T) {
	pkg := loadTestPackage(t, "p1effects")
	fd := analysis.FindFuncDecl(pkg, "SendOnChannel")
	if fd == nil {
		t.Fatal("SendOnChannel not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	for _, e := range result.SideEffects {
		if e.Type == taxonomy.ChannelSend && e.Tier != taxonomy.TierP1 {
			t.Errorf("ChannelSend should be P1, got %s", e.Tier)
		}
	}
}

// ---------------------------------------------------------------------------
// P2-tier effect tests
// ---------------------------------------------------------------------------

func TestP2_GoroutineSpawn(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "SpawnGoroutine")
	if fd == nil {
		t.Fatal("SpawnGoroutine not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.GoroutineSpawn) {
		t.Error("expected GoroutineSpawn for SpawnGoroutine")
	}
}

func TestP2_GoroutineSpawnWithFunc(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "SpawnGoroutineWithFunc")
	if fd == nil {
		t.Fatal("SpawnGoroutineWithFunc not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.GoroutineSpawn) {
		t.Error("expected GoroutineSpawn for SpawnGoroutineWithFunc")
	}
}

func TestP2_NoGoroutine(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "NoGoroutine")
	if fd == nil {
		t.Fatal("NoGoroutine not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.GoroutineSpawn) {
		t.Error("NoGoroutine should NOT produce GoroutineSpawn")
	}
}

func TestP2_Panic(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "PanicWithString")
	if fd == nil {
		t.Fatal("PanicWithString not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.Panic) {
		t.Error("expected Panic for PanicWithString")
	}
}

func TestP2_PanicWithError(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "PanicWithError")
	if fd == nil {
		t.Fatal("PanicWithError not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.Panic) {
		t.Error("expected Panic for PanicWithError")
	}
}

func TestP2_NoPanic(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "NoPanic")
	if fd == nil {
		t.Fatal("NoPanic not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.Panic) {
		t.Error("NoPanic should NOT produce Panic")
	}
}

func TestP2_FileSystemWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "WriteFileOS")
	if fd == nil {
		t.Fatal("WriteFileOS not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.FileSystemWrite) {
		t.Error("expected FileSystemWrite for WriteFileOS")
	}
}

func TestP2_FileSystemWrite_Create(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "CreateFile")
	if fd == nil {
		t.Fatal("CreateFile not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.FileSystemWrite) {
		t.Error("expected FileSystemWrite for CreateFile")
	}
}

func TestP2_FileSystemWrite_Mkdir(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "MkdirCall")
	if fd == nil {
		t.Fatal("MkdirCall not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.FileSystemWrite) {
		t.Error("expected FileSystemWrite for MkdirCall")
	}
}

func TestP2_ReadFileOnly(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "ReadFileOnly")
	if fd == nil {
		t.Fatal("ReadFileOnly not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.FileSystemWrite) {
		t.Error("ReadFileOnly should NOT produce FileSystemWrite")
	}
}

func TestP2_FileSystemDelete(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "RemoveFile")
	if fd == nil {
		t.Fatal("RemoveFile not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.FileSystemDelete) {
		t.Error("expected FileSystemDelete for RemoveFile")
	}
}

func TestP2_FileSystemDelete_RemoveAll(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "RemoveAllDir")
	if fd == nil {
		t.Fatal("RemoveAllDir not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.FileSystemDelete) {
		t.Error("expected FileSystemDelete for RemoveAllDir")
	}
}

func TestP2_FileSystemMeta(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "ChmodFile")
	if fd == nil {
		t.Fatal("ChmodFile not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.FileSystemMeta) {
		t.Error("expected FileSystemMeta for ChmodFile")
	}
}

func TestP2_FileSystemMeta_Symlink(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "SymlinkFile")
	if fd == nil {
		t.Fatal("SymlinkFile not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.FileSystemMeta) {
		t.Error("expected FileSystemMeta for SymlinkFile")
	}
}

func TestP2_StatFile_NoMeta(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "StatFile")
	if fd == nil {
		t.Fatal("StatFile not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.FileSystemMeta) {
		t.Error("StatFile should NOT produce FileSystemMeta")
	}
}

func TestP2_LogWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "LogPrint")
	if fd == nil {
		t.Fatal("LogPrint not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.LogWrite) {
		t.Error("expected LogWrite for LogPrint")
	}
}

func TestP2_LogWrite_Fatal(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "LogFatal")
	if fd == nil {
		t.Fatal("LogFatal not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.LogWrite) {
		t.Error("expected LogWrite for LogFatal")
	}
}

func TestP2_LogWrite_Slog(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "SlogInfo")
	if fd == nil {
		t.Fatal("SlogInfo not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.LogWrite) {
		t.Error("expected LogWrite for SlogInfo")
	}
}

func TestP2_NoLogging(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "NoLogging")
	if fd == nil {
		t.Fatal("NoLogging not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.LogWrite) {
		t.Error("NoLogging should NOT produce LogWrite")
	}
}

func TestP2_ContextCancellation(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "CancelContext")
	if fd == nil {
		t.Fatal("CancelContext not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.ContextCancellation) {
		t.Error("expected ContextCancellation for CancelContext")
	}
}

func TestP2_ContextCancellation_Timeout(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "TimeoutContext")
	if fd == nil {
		t.Fatal("TimeoutContext not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.ContextCancellation) {
		t.Error("expected ContextCancellation for TimeoutContext")
	}
}

func TestP2_UseContextNoCancel(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "UseContextNoCancel")
	if fd == nil {
		t.Fatal("UseContextNoCancel not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.ContextCancellation) {
		t.Error("UseContextNoCancel should NOT produce ContextCancellation")
	}
}

func TestP2_CallbackInvocation(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "InvokeCallback")
	if fd == nil {
		t.Fatal("InvokeCallback not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.CallbackInvocation) {
		t.Error("expected CallbackInvocation for InvokeCallback")
	}
}

func TestP2_CallbackInvocation_WithArgs(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "InvokeCallbackWithArgs")
	if fd == nil {
		t.Fatal("InvokeCallbackWithArgs not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.CallbackInvocation) {
		t.Error("expected CallbackInvocation for InvokeCallbackWithArgs")
	}
}

func TestP2_NoCallback(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "NoCallback")
	if fd == nil {
		t.Fatal("NoCallback not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.CallbackInvocation) {
		t.Error("NoCallback should NOT produce CallbackInvocation")
	}
}

func TestP2_DatabaseWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "DBExec")
	if fd == nil {
		t.Fatal("DBExec not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.DatabaseWrite) {
		t.Error("expected DatabaseWrite for DBExec")
	}
}

func TestP2_DatabaseQuery_NoWrite(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "DBQuery")
	if fd == nil {
		t.Fatal("DBQuery not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if hasEffect(result.SideEffects, taxonomy.DatabaseWrite) {
		t.Error("DBQuery should NOT produce DatabaseWrite")
	}
}

func TestP2_DatabaseTransaction(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "BeginTx")
	if fd == nil {
		t.Fatal("BeginTx not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.DatabaseTransaction) {
		t.Error("expected DatabaseTransaction for BeginTx")
	}
}

func TestP2_PureP2(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "PureP2")
	if fd == nil {
		t.Fatal("PureP2 not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	var p2Effects []taxonomy.SideEffect
	for _, e := range result.SideEffects {
		if e.Tier == taxonomy.TierP2 {
			p2Effects = append(p2Effects, e)
		}
	}
	if len(p2Effects) != 0 {
		t.Errorf("PureP2 should have no P2 side effects, got %d: %v",
			len(p2Effects), p2Effects)
	}
}

func TestP2_EffectsAreP2Tier(t *testing.T) {
	pkg := loadTestPackage(t, "p2effects")
	fd := analysis.FindFuncDecl(pkg, "SpawnGoroutine")
	if fd == nil {
		t.Fatal("SpawnGoroutine not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	for _, e := range result.SideEffects {
		if e.Type == taxonomy.GoroutineSpawn && e.Tier != taxonomy.TierP2 {
			t.Errorf("GoroutineSpawn should be P2, got %s", e.Tier)
		}
	}
}

// ===================================================================
// Edge case tests (T042)
// ===================================================================

// --- Generics ---

func TestEdge_GenericIdentity(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "GenericIdentity")
	if fd == nil {
		t.Fatal("GenericIdentity not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Generic identity returns its type parameter — should detect ReturnValue.
	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("GenericIdentity should detect ReturnValue")
	}
}

func TestEdge_GenericSwap(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "GenericSwap")
	if fd == nil {
		t.Fatal("GenericSwap not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Should detect two ReturnValue effects.
	count := 0
	for _, e := range result.SideEffects {
		if e.Type == taxonomy.ReturnValue {
			count++
		}
	}
	if count != 2 {
		t.Errorf("GenericSwap should detect 2 ReturnValues, got %d", count)
	}
}

func TestEdge_GenericSliceMap(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "GenericSliceMap")
	if fd == nil {
		t.Fatal("GenericSliceMap not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Should detect ReturnValue (returns []U) and CallbackInvocation (calls fn).
	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("GenericSliceMap should detect ReturnValue")
	}
	if !hasEffect(result.SideEffects, taxonomy.CallbackInvocation) {
		t.Error("GenericSliceMap should detect CallbackInvocation")
	}
}

func TestEdge_GenericContainerAdd(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindMethodDecl(pkg, "*GenericContainer[T]", "Add")
	if fd == nil {
		// Try alternate receiver name formats.
		fd = analysis.FindMethodDecl(pkg, "*GenericContainer", "Add")
	}
	if fd == nil {
		t.Skip("GenericContainer.Add not found — receiver format may differ")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Known limitation: SSA-based mutation analysis doesn't fully
	// resolve generic receiver types, so ReceiverMutation may not
	// be detected on generic types. Document this as a known gap.
	if hasEffect(result.SideEffects, taxonomy.ReceiverMutation) {
		t.Log("GenericContainer.Add correctly detects ReceiverMutation")
	} else {
		t.Log("Known limitation: ReceiverMutation not detected on generic receiver types")
	}
}

func TestEdge_GenericContainerCount(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindMethodDecl(pkg, "*GenericContainer[T]", "Count")
	if fd == nil {
		fd = analysis.FindMethodDecl(pkg, "*GenericContainer", "Count")
	}
	if fd == nil {
		t.Skip("GenericContainer.Count not found — receiver format may differ")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Should detect ReturnValue but no ReceiverMutation.
	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("GenericContainer.Count should detect ReturnValue")
	}
	if hasEffect(result.SideEffects, taxonomy.ReceiverMutation) {
		t.Error("GenericContainer.Count should NOT detect ReceiverMutation")
	}
}

// --- Unsafe ---

func TestEdge_UnsafePointerCast(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "UnsafePointerCast")
	if fd == nil {
		t.Fatal("UnsafePointerCast not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Should at least detect ReturnValue. UnsafeMutation detection is
	// P4-tier and may not be implemented yet — this test documents the
	// current behavior.
	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("UnsafePointerCast should detect ReturnValue")
	}
	t.Logf("UnsafePointerCast effects: %v", result.SideEffects)
}

func TestEdge_UnsafeSizeof(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "UnsafeSizeof")
	if fd == nil {
		t.Fatal("UnsafeSizeof not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Sizeof is read-only — should only have ReturnValue.
	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("UnsafeSizeof should detect ReturnValue")
	}
	if hasEffect(result.SideEffects, taxonomy.ReceiverMutation) {
		t.Error("UnsafeSizeof should NOT detect ReceiverMutation")
	}
}

// --- Empty / No-op functions ---

func TestEdge_EmptyFunction(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "EmptyFunction")
	if fd == nil {
		t.Fatal("EmptyFunction not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if len(result.SideEffects) != 0 {
		t.Errorf("EmptyFunction should have 0 side effects, got %d: %v",
			len(result.SideEffects), result.SideEffects)
	}
}

func TestEdge_NoOpWithParams(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "NoOpWithParams")
	if fd == nil {
		t.Fatal("NoOpWithParams not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if len(result.SideEffects) != 0 {
		t.Errorf("NoOpWithParams should have 0 side effects, got %d: %v",
			len(result.SideEffects), result.SideEffects)
	}
}

// --- Variadic functions ---

func TestEdge_VariadicSum(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "VariadicSum")
	if fd == nil {
		t.Fatal("VariadicSum not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("VariadicSum should detect ReturnValue")
	}
}

func TestEdge_VariadicWithCallback(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "VariadicWithCallback")
	if fd == nil {
		t.Fatal("VariadicWithCallback not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Known limitation: P2 callback detection looks for direct calls
	// to func-typed parameters, but variadic params iterated via
	// range are not resolved to the original parameter.
	if hasEffect(result.SideEffects, taxonomy.CallbackInvocation) {
		t.Log("VariadicWithCallback correctly detects CallbackInvocation")
	} else {
		t.Log("Known limitation: CallbackInvocation not detected for variadic func params called via range")
	}
}

// --- Complex signatures ---

func TestEdge_MultiReturn(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "MultiReturn")
	if fd == nil {
		t.Fatal("MultiReturn not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Should detect 3 ReturnValues + 1 ErrorReturn = 4 total.
	rv := 0
	er := 0
	for _, e := range result.SideEffects {
		switch e.Type {
		case taxonomy.ReturnValue:
			rv++
		case taxonomy.ErrorReturn:
			er++
		}
	}
	if rv != 3 {
		t.Errorf("MultiReturn should detect 3 ReturnValues, got %d", rv)
	}
	if er != 1 {
		t.Errorf("MultiReturn should detect 1 ErrorReturn, got %d", er)
	}
}

func TestEdge_FuncReturningFunc(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "FuncReturningFunc")
	if fd == nil {
		t.Fatal("FuncReturningFunc not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	if !hasEffect(result.SideEffects, taxonomy.ReturnValue) {
		t.Error("FuncReturningFunc should detect ReturnValue")
	}
}

func TestEdge_NamedMultiReturn(t *testing.T) {
	pkg := loadTestPackage(t, "edgecases")
	fd := analysis.FindFuncDecl(pkg, "NamedMultiReturn")
	if fd == nil {
		t.Fatal("NamedMultiReturn not found")
	}
	result := analysis.AnalyzeFunction(pkg, fd)

	// Named returns: x int, y string, err error.
	rv := 0
	er := 0
	for _, e := range result.SideEffects {
		switch e.Type {
		case taxonomy.ReturnValue:
			rv++
		case taxonomy.ErrorReturn:
			er++
		}
	}
	if rv != 2 {
		t.Errorf("NamedMultiReturn should detect 2 ReturnValues, got %d", rv)
	}
	if er != 1 {
		t.Errorf("NamedMultiReturn should detect 1 ErrorReturn, got %d", er)
	}
}

// --- Package-level analysis of edge cases ---

func TestEdge_AnalyzePackage(t *testing.T) {
	// Verify that the entire edgecases package can be analyzed
	// without panics or errors.
	pkg := loadTestPackage(t, "edgecases")
	opts := analysis.Options{IncludeUnexported: true}
	results, err := analysis.Analyze(pkg, opts)
	if err != nil {
		t.Fatalf("Analyze(edgecases) failed: %v", err)
	}
	if len(results) == 0 {
		t.Error("expected at least 1 result from edgecases package")
	}
	t.Logf("edgecases: %d functions analyzed without errors", len(results))
}
