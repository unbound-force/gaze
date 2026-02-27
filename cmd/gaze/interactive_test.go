package main

import (
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// TestRenderAnalyzeContent_EmptyResults verifies that an empty slice
// produces output indicating zero functions and zero side effects (FR-016).
func TestRenderAnalyzeContent_EmptyResults(t *testing.T) {
	output := renderAnalyzeContent([]taxonomy.AnalysisResult{})

	if !strings.Contains(output, "0 function(s)") {
		t.Errorf("expected output to contain '0 function(s)', got:\n%s", output)
	}
	if !strings.Contains(output, "0 side effect(s)") {
		t.Errorf("expected output to contain '0 side effect(s)', got:\n%s", output)
	}
}

// TestRenderAnalyzeContent_WithSideEffects verifies that results with
// side effects include the function's qualified name, tier, and effect
// type description in the output (FR-016).
func TestRenderAnalyzeContent_WithSideEffects(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "example.com/pkg",
				Function: "DoSomething",
				Location: "pkg.go:10",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-001",
					Type:        taxonomy.GlobalMutation,
					Tier:        taxonomy.TierP1,
					Description: "modifies global counter",
					Location:    "pkg.go:15",
					Target:      "counter",
				},
			},
		},
	}

	output := renderAnalyzeContent(results)

	// Qualified name for a function without receiver is just the function name.
	if !strings.Contains(output, "DoSomething") {
		t.Errorf("expected output to contain qualified name 'DoSomething', got:\n%s", output)
	}
	if !strings.Contains(output, "1 function(s)") {
		t.Errorf("expected output to contain '1 function(s)', got:\n%s", output)
	}
	if !strings.Contains(output, "1 side effect(s)") {
		t.Errorf("expected output to contain '1 side effect(s)', got:\n%s", output)
	}
	if !strings.Contains(output, "P1") {
		t.Errorf("expected output to contain tier 'P1', got:\n%s", output)
	}
	if !strings.Contains(output, "GlobalMutation") {
		t.Errorf("expected output to contain effect type 'GlobalMutation', got:\n%s", output)
	}
	if !strings.Contains(output, "modifies global counter") {
		t.Errorf("expected output to contain description 'modifies global counter', got:\n%s", output)
	}
}

// TestRenderAnalyzeContent_WithReceiver verifies that a method with a
// receiver shows the qualified name in "(Receiver).Method" format (FR-016).
func TestRenderAnalyzeContent_WithReceiver(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "example.com/pkg",
				Function: "Save",
				Receiver: "*Store",
				Location: "store.go:20",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-002",
					Type:        taxonomy.FileSystemWrite,
					Tier:        taxonomy.TierP2,
					Description: "writes to disk",
					Location:    "store.go:25",
					Target:      "/data",
				},
			},
		},
	}

	output := renderAnalyzeContent(results)

	// QualifiedName() for a method with receiver "*Store" returns "(*Store).Save".
	if !strings.Contains(output, "(*Store).Save") {
		t.Errorf("expected output to contain '(*Store).Save', got:\n%s", output)
	}
	if !strings.Contains(output, "P2") {
		t.Errorf("expected output to contain tier 'P2', got:\n%s", output)
	}
	if !strings.Contains(output, "FileSystemWrite") {
		t.Errorf("expected output to contain effect type 'FileSystemWrite', got:\n%s", output)
	}
}

// TestRenderAnalyzeContent_MultipleTiers verifies that multiple side
// effects with different tiers are all rendered (FR-016).
func TestRenderAnalyzeContent_MultipleTiers(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "example.com/pkg",
				Function: "Process",
				Location: "proc.go:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-010",
					Type:        taxonomy.ReturnValue,
					Tier:        taxonomy.TierP0,
					Description: "returns result",
					Location:    "proc.go:5",
					Target:      "int",
				},
				{
					ID:          "se-011",
					Type:        taxonomy.ChannelSend,
					Tier:        taxonomy.TierP1,
					Description: "sends on channel",
					Location:    "proc.go:10",
					Target:      "ch",
				},
				{
					ID:          "se-012",
					Type:        taxonomy.GoroutineSpawn,
					Tier:        taxonomy.TierP2,
					Description: "spawns goroutine",
					Location:    "proc.go:15",
					Target:      "worker",
				},
			},
		},
	}

	output := renderAnalyzeContent(results)

	if !strings.Contains(output, "3 side effect(s)") {
		t.Errorf("expected output to contain '3 side effect(s)', got:\n%s", output)
	}
	for _, tier := range []string{"P0", "P1", "P2"} {
		if !strings.Contains(output, tier) {
			t.Errorf("expected output to contain tier %q, got:\n%s", tier, output)
		}
	}
	for _, typ := range []string{"ReturnValue", "ChannelSend", "GoroutineSpawn"} {
		if !strings.Contains(output, typ) {
			t.Errorf("expected output to contain effect type %q, got:\n%s", typ, output)
		}
	}
}

// TestRenderAnalyzeContent_DescriptionTruncation verifies that
// descriptions longer than 50 characters are truncated with "..."
// in the rendered output (FR-017).
func TestRenderAnalyzeContent_DescriptionTruncation(t *testing.T) {
	longDesc := "this is a very long description that exceeds fifty characters by a lot"
	if len(longDesc) <= 50 {
		t.Fatalf("test setup: description must be >50 chars, got %d", len(longDesc))
	}

	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "example.com/pkg",
				Function: "LongDesc",
				Location: "long.go:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-100",
					Type:        taxonomy.GlobalMutation,
					Tier:        taxonomy.TierP1,
					Description: longDesc,
					Location:    "long.go:5",
					Target:      "x",
				},
			},
		},
	}

	output := renderAnalyzeContent(results)

	// The full description should NOT appear — it should be truncated.
	if strings.Contains(output, longDesc) {
		t.Error("expected long description to be truncated, but full description found in output")
	}

	// The truncated form should be first 47 chars + "...".
	truncated := longDesc[:47] + "..."
	if !strings.Contains(output, truncated) {
		t.Errorf("expected output to contain truncated description %q, got:\n%s", truncated, output)
	}
}

// TestRenderAnalyzeContent_ShortDescriptionNotTruncated verifies that
// descriptions at exactly 50 characters are NOT truncated (FR-017).
func TestRenderAnalyzeContent_ShortDescriptionNotTruncated(t *testing.T) {
	// Exactly 50 characters — should not be truncated.
	desc50 := "12345678901234567890123456789012345678901234567890"
	if len(desc50) != 50 {
		t.Fatalf("test setup: description must be exactly 50 chars, got %d", len(desc50))
	}

	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "example.com/pkg",
				Function: "Exact50",
				Location: "exact.go:1",
			},
			SideEffects: []taxonomy.SideEffect{
				{
					ID:          "se-200",
					Type:        taxonomy.MapMutation,
					Tier:        taxonomy.TierP1,
					Description: desc50,
					Location:    "exact.go:5",
					Target:      "m",
				},
			},
		},
	}

	output := renderAnalyzeContent(results)

	// Exactly 50 chars should appear in full without truncation.
	if !strings.Contains(output, desc50) {
		t.Errorf("expected output to contain full 50-char description, got:\n%s", output)
	}
}

// TestRenderAnalyzeContent_NoSideEffects verifies that a result with
// zero side effects shows "No side effects detected" (FR-017).
func TestRenderAnalyzeContent_NoSideEffects(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			Target: taxonomy.FunctionTarget{
				Package:  "example.com/pkg",
				Function: "Pure",
				Location: "pure.go:1",
			},
			SideEffects: nil,
		},
	}

	output := renderAnalyzeContent(results)

	if !strings.Contains(output, "Pure") {
		t.Errorf("expected output to contain function name 'Pure', got:\n%s", output)
	}
	if !strings.Contains(output, "No side effects detected") {
		t.Errorf("expected output to contain 'No side effects detected', got:\n%s", output)
	}
	if !strings.Contains(output, "1 function(s)") {
		t.Errorf("expected output to contain '1 function(s)', got:\n%s", output)
	}
	if !strings.Contains(output, "0 side effect(s)") {
		t.Errorf("expected output to contain '0 side effect(s)', got:\n%s", output)
	}
}
