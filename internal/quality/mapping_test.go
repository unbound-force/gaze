package quality_test

import (
	"testing"

	"github.com/unbound-force/gaze/v2/internal/analysis"
	"github.com/unbound-force/gaze/v2/internal/quality"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// TestMatchContainerUnwrap_ChainDepth verifies that the container
// unwrap pass traces through 4+ intermediate steps (the MCP pattern)
// and that assertions on the unwrapped data are mapped at confidence 55.
func TestMatchContainerUnwrap_ChainDepth(t *testing.T) {
	pkg := loadPkg(t, "containerunwrap")
	nonTestPkg, err := loadNonTestPackage("containerunwrap")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	testFuncs := quality.FindTestFunctions(pkg)
	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}

	resultMap := make(map[string]*taxonomy.AnalysisResult)
	for i := range results {
		resultMap[results[i].Target.QualifiedName()] = &results[i]
	}

	// Track mappings per test function.
	type testMapping struct {
		testFunc    string
		mapped      int
		unmapped    int
		hasConf55   bool // has at least one container unwrap mapping
		confidences []int
	}
	var mappings []testMapping

	for _, tf := range testFuncs {
		ssaFunc := ssaPkg.Func(tf.Name)
		if ssaFunc == nil {
			continue
		}

		targets, _ := quality.InferTargets(ssaFunc, pkg, quality.DefaultOptions())
		for _, target := range targets {
			result, ok := resultMap[target.FuncName]
			if !ok {
				continue
			}

			sites := quality.DetectAssertions(tf.Decl, pkg, 3)
			mapped, unmapped, _ := quality.MapAssertionsToEffects(
				ssaFunc, target.SSAFunc, sites, result.SideEffects, pkg,
			)

			hasConf55 := false
			var confidences []int
			for _, m := range mapped {
				confidences = append(confidences, m.Confidence)
				if m.Confidence == 55 {
					hasConf55 = true
				}
			}

			mappings = append(mappings, testMapping{
				testFunc:    tf.Name,
				mapped:      len(mapped),
				unmapped:    len(unmapped),
				hasConf55:   hasConf55,
				confidences: confidences,
			})
		}
	}

	// Verify the deep chain test maps its assertions.
	deepChainFound := false
	for _, m := range mappings {
		t.Logf("  %s: mapped=%d unmapped=%d hasConf55=%v confidences=%v",
			m.testFunc, m.mapped, m.unmapped, m.hasConf55, m.confidences)

		if m.testFunc == "TestWrapMCPStyle_DeepChain" {
			deepChainFound = true
			if m.mapped == 0 {
				t.Errorf("TestWrapMCPStyle_DeepChain: expected mapped assertions, got 0")
			}
			// The deep chain data assertions should include at least
			// one mapping at container unwrap confidence (55).
			if !m.hasConf55 {
				t.Errorf("TestWrapMCPStyle_DeepChain: expected at least one container unwrap mapping (confidence 55)")
			}
		}
	}

	if !deepChainFound {
		t.Error("TestWrapMCPStyle_DeepChain not found in mappings")
	}
}

// TestMatchContainerUnwrap_BasicPattern verifies that the basic
// container-unwrap-assert pattern (assign → field → unmarshal → assert)
// produces mappings at confidence 55.
func TestMatchContainerUnwrap_BasicPattern(t *testing.T) {
	pkg := loadPkg(t, "containerunwrap")
	nonTestPkg, err := loadNonTestPackage("containerunwrap")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	testFuncs := quality.FindTestFunctions(pkg)
	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}

	resultMap := make(map[string]*taxonomy.AnalysisResult)
	for i := range results {
		resultMap[results[i].Target.QualifiedName()] = &results[i]
	}

	for _, tf := range testFuncs {
		if tf.Name != "TestWrapJSON_BasicUnmarshal" {
			continue
		}

		ssaFunc := ssaPkg.Func(tf.Name)
		if ssaFunc == nil {
			t.Fatal("SSA function not found for TestWrapJSON_BasicUnmarshal")
		}

		targets, _ := quality.InferTargets(ssaFunc, pkg, quality.DefaultOptions())
		if len(targets) == 0 {
			t.Fatal("no targets inferred for TestWrapJSON_BasicUnmarshal")
		}

		for _, target := range targets {
			result, ok := resultMap[target.FuncName]
			if !ok {
				continue
			}

			sites := quality.DetectAssertions(tf.Decl, pkg, 3)
			mapped, _, _ := quality.MapAssertionsToEffects(
				ssaFunc, target.SSAFunc, sites, result.SideEffects, pkg,
			)

			if len(mapped) == 0 {
				t.Error("expected at least one mapped assertion for TestWrapJSON_BasicUnmarshal")
			}

			// Verify at least one assertion is mapped at container
			// unwrap confidence (55). Other assertions may map at
			// higher confidence via direct/indirect passes.
			hasConf55 := false
			for _, m := range mapped {
				t.Logf("  mapped: %s conf=%d", m.AssertionLocation, m.Confidence)
				if m.Confidence == 55 {
					hasConf55 = true
				}
			}
			if !hasConf55 {
				t.Error("expected at least one container unwrap mapping (confidence 55)")
			}
		}
	}
}

// TestMatchContainerUnwrap_ErrorExclusion verifies FR-009: error
// assertions from transformation calls are NOT mapped to ReturnValue.
func TestMatchContainerUnwrap_ErrorExclusion(t *testing.T) {
	pkg := loadPkg(t, "containerunwrap")
	nonTestPkg, err := loadNonTestPackage("containerunwrap")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	testFuncs := quality.FindTestFunctions(pkg)
	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}

	resultMap := make(map[string]*taxonomy.AnalysisResult)
	for i := range results {
		resultMap[results[i].Target.QualifiedName()] = &results[i]
	}

	for _, tf := range testFuncs {
		if tf.Name != "TestWrapMCPStyle_ErrorExclusion" {
			continue
		}

		ssaFunc := ssaPkg.Func(tf.Name)
		if ssaFunc == nil {
			t.Fatal("SSA function not found for TestWrapMCPStyle_ErrorExclusion")
		}

		targets, _ := quality.InferTargets(ssaFunc, pkg, quality.DefaultOptions())
		for _, target := range targets {
			result, ok := resultMap[target.FuncName]
			if !ok {
				continue
			}

			sites := quality.DetectAssertions(tf.Decl, pkg, 3)
			mapped, unmapped, _ := quality.MapAssertionsToEffects(
				ssaFunc, target.SSAFunc, sites, result.SideEffects, pkg,
			)

			t.Logf("mapped=%d unmapped=%d", len(mapped), len(unmapped))
			for _, m := range mapped {
				t.Logf("  mapped: %s conf=%d effectID=%s",
					m.AssertionLocation, m.Confidence, m.SideEffectID)
			}
			for _, u := range unmapped {
				t.Logf("  unmapped: %s reason=%s",
					u.AssertionLocation, u.UnmappedReason)
			}

			// The data assertion should be mapped.
			if len(mapped) == 0 {
				t.Error("expected at least one mapped assertion (data field)")
			}

			// Find the ReturnValue effect ID to verify FR-009.
			var returnValueEffectID string
			for _, e := range result.SideEffects {
				if e.Type == taxonomy.ReturnValue {
					returnValueEffectID = e.ID
					break
				}
			}

			// FR-009: error assertions from the unmarshal call
			// MUST NOT be mapped to the target function's
			// ReturnValue effect. Verify that no mapped assertion
			// with error_check type points to the ReturnValue.
			for _, m := range mapped {
				if m.AssertionType == taxonomy.AssertionErrorCheck && m.SideEffectID == returnValueEffectID {
					t.Errorf("FR-009 violation: error assertion at %s mapped to ReturnValue (confidence %d) — unmarshal errors must not map to target's ReturnValue",
						m.AssertionLocation, m.Confidence)
				}
			}

			// Verify at least one container unwrap mapping exists
			// (data field assertions should map at confidence 55).
			hasConf55 := false
			for _, m := range mapped {
				if m.Confidence == 55 {
					hasConf55 = true
					break
				}
			}
			if !hasConf55 {
				t.Error("expected at least one container unwrap mapping (confidence 55) for data field assertions")
			}
		}
	}
}

// TestMatchContainerUnwrap_ReturnsNil_WhenDirectMatchExists verifies
// that for an assertion already matchable by direct identity (confidence
// 75), the pipeline returns the direct match and the container unwrap
// pass is never reached (the short-circuit ensures this).
func TestMatchContainerUnwrap_ReturnsNil_WhenDirectMatchExists(t *testing.T) {
	// Use the welltested fixture where assertions directly reference
	// the return variable (no container unwrap needed).
	pkg := loadPkg(t, "welltested")
	nonTestPkg, err := loadNonTestPackage("welltested")
	if err != nil {
		t.Fatalf("loading non-test package: %v", err)
	}

	opts := analysis.Options{Version: "test"}
	results, err := analysis.Analyze(nonTestPkg, opts)
	if err != nil {
		t.Fatalf("analysis failed: %v", err)
	}

	testFuncs := quality.FindTestFunctions(pkg)
	_, ssaPkg, err := quality.BuildTestSSA(pkg)
	if err != nil {
		t.Fatalf("BuildTestSSA failed: %v", err)
	}

	resultMap := make(map[string]*taxonomy.AnalysisResult)
	for i := range results {
		resultMap[results[i].Target.QualifiedName()] = &results[i]
	}

	for _, tf := range testFuncs {
		if tf.Name != "TestAdd" {
			continue
		}

		ssaFunc := ssaPkg.Func(tf.Name)
		if ssaFunc == nil {
			t.Fatal("SSA function not found for TestAdd")
		}

		targets, _ := quality.InferTargets(ssaFunc, pkg, quality.DefaultOptions())
		for _, target := range targets {
			result, ok := resultMap[target.FuncName]
			if !ok {
				continue
			}

			sites := quality.DetectAssertions(tf.Decl, pkg, 3)
			mapped, _, _ := quality.MapAssertionsToEffects(
				ssaFunc, target.SSAFunc, sites, result.SideEffects, pkg,
			)

			if len(mapped) == 0 {
				t.Fatal("expected mapped assertions for TestAdd")
			}

			// All mapped assertions should be at confidence 75
			// (direct identity), NOT 55 (container unwrap).
			for _, m := range mapped {
				if m.Confidence == 55 {
					t.Errorf("TestAdd assertion at %s was mapped by container unwrap (confidence 55) — expected direct match (75)",
						m.AssertionLocation)
				}
				if m.Confidence != 75 {
					t.Errorf("TestAdd assertion at %s: expected confidence 75, got %d",
						m.AssertionLocation, m.Confidence)
				}
			}
		}
	}
}
