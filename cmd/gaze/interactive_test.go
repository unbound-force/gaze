package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// fakeBinaryPath is the path to the compiled fake_analyzer binary.
// Built once in TestMain from the protocol testdata.
var fakeBinaryPath string

// TestMain forces lipgloss to use ASCII (no-color) rendering so that
// string content assertions in renderAnalyzeContent tests are
// deterministic across terminal environments and CI configurations.
// It also builds the fake_analyzer binary for external analyzer tests.
func TestMain(m *testing.M) {
	lipgloss.DefaultRenderer().SetColorProfile(termenv.Ascii)

	// Build the fake analyzer binary for external analyzer integration
	// tests (TestCrapWithExternalAnalyzer, etc.).
	tmpDir, err := os.MkdirTemp("", "gaze-cli-test-*")
	if err != nil {
		panic("creating temp dir: " + err.Error())
	}
	fakeBinaryPath = filepath.Join(tmpDir, "fake_analyzer")
	cmd := exec.Command("go", "build", "-o", fakeBinaryPath,
		"../../internal/protocol/testdata/fake_analyzer/")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("building fake_analyzer: " + err.Error())
	}

	code := m.Run()
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

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

// TestUpdate verifies the Bubble Tea Update method on analyzeModel,
// covering viewport initialization, resize, quit key, help toggle,
// and unhandled message passthrough.
func TestUpdate(t *testing.T) {
	t.Run("WindowSizeMsg_InitializesViewport", func(t *testing.T) {
		m := newAnalyzeModel(nil)
		if m.ready {
			t.Fatal("expected ready to be false before first WindowSizeMsg")
		}

		result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		updated, ok := result.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}

		if !updated.ready {
			t.Error("expected ready to be true after WindowSizeMsg")
		}
		if updated.viewport.Width != 80 {
			t.Errorf("expected viewport width 80, got %d", updated.viewport.Width)
		}
		// footerHeight is 2, so viewport height = 24 - 2 = 22.
		if updated.viewport.Height != 22 {
			t.Errorf("expected viewport height 22 (24 - 2 footerHeight), got %d", updated.viewport.Height)
		}
	})

	t.Run("WindowSizeMsg_Resize", func(t *testing.T) {
		m := newAnalyzeModel(nil)

		// First WindowSizeMsg initializes the viewport.
		result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m2, ok := result.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}
		if !m2.ready {
			t.Fatal("expected ready after first WindowSizeMsg")
		}

		// Second WindowSizeMsg resizes the existing viewport.
		result2, _ := m2.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		updated, ok := result2.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}

		if !updated.ready {
			t.Error("expected ready to remain true after resize")
		}
		if updated.viewport.Width != 120 {
			t.Errorf("expected viewport width 120, got %d", updated.viewport.Width)
		}
		// footerHeight is 2, so viewport height = 40 - 2 = 38.
		if updated.viewport.Height != 38 {
			t.Errorf("expected viewport height 38 (40 - 2 footerHeight), got %d", updated.viewport.Height)
		}
	})

	t.Run("KeyMsg_Quit", func(t *testing.T) {
		m := newAnalyzeModel(nil)

		// Initialize viewport first so key handling proceeds normally.
		result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m2, ok := result.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}

		// Send the quit key ('q').
		_, cmd := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
		if cmd == nil {
			t.Fatal("expected non-nil command for quit key")
		}
		msg := cmd()
		if _, ok := msg.(tea.QuitMsg); !ok {
			t.Errorf("expected tea.QuitMsg, got %T", msg)
		}
	})

	t.Run("KeyMsg_HelpToggle", func(t *testing.T) {
		m := newAnalyzeModel(nil)

		// Initialize viewport.
		result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m2, ok := result.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}
		if m2.help.ShowAll {
			t.Fatal("expected ShowAll to be false initially")
		}

		// First '?' press — toggle help on.
		result3, _ := m2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		m3, ok := result3.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}
		if !m3.help.ShowAll {
			t.Error("expected ShowAll to be true after pressing '?'")
		}

		// Second '?' press — toggle help off.
		result4, _ := m3.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
		m4, ok := result4.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}
		if m4.help.ShowAll {
			t.Error("expected ShowAll to be false after second '?' press")
		}
	})

	t.Run("UnhandledMsg", func(t *testing.T) {
		type customMsg struct{}

		m := newAnalyzeModel(nil)

		// Initialize viewport.
		result, _ := m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
		m2, ok := result.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}
		readyBefore := m2.ready

		// Send an unhandled message type — should not panic or change ready.
		result3, cmd := m2.Update(customMsg{})
		m3, ok := result3.(analyzeModel)
		if !ok {
			t.Fatal("expected analyzeModel from Update")
		}

		if m3.ready != readyBefore {
			t.Error("ready flag changed after unhandled message")
		}
		// cmd may be non-nil (viewport may return a cmd), but no panic.
		_ = cmd
	})
}
