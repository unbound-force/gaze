package aireport

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// buildFullPayload constructs a fully-populated ReportPayload for testing.
// Each section contains realistic data with known field values.
func buildFullPayload(t *testing.T) *ReportPayload {
	t.Helper()

	// CRAP section with worst offender lists and recommended actions.
	crapJSON := mustMarshal(t, map[string]interface{}{
		"scores": []map[string]interface{}{
			{
				"package": "github.com/example/pkg", "function": "DoWork",
				"file": "pkg/work.go", "line": 10, "complexity": 8,
				"line_coverage": 75.0, "crap": 12.5,
			},
		},
		"summary": map[string]interface{}{
			"total_functions": 1, "avg_complexity": 8.0,
			"avg_line_coverage": 75.0, "avg_crap": 12.5,
			"crapload": 1, "crap_threshold": 15.0,
			"worst_crap": []map[string]interface{}{
				{"package": "github.com/example/pkg", "function": "DoWork",
					"file": "pkg/work.go", "line": 10, "complexity": 8,
					"line_coverage": 75.0, "crap": 12.5},
			},
			"worst_gaze_crap": []map[string]interface{}{
				{"package": "github.com/example/pkg", "function": "DoWork",
					"file": "pkg/work.go", "line": 10, "complexity": 8,
					"line_coverage": 75.0, "crap": 12.5},
			},
			"recommended_actions": []map[string]interface{}{
				{"function": "DoWork", "package": "github.com/example/pkg",
					"file": "pkg/work.go", "line": 10,
					"fix_strategy": "add_tests", "crap": 12.5, "complexity": 8},
			},
		},
	})

	// Quality section with full SideEffect objects in gaps and ambiguous effects.
	qualityJSON := mustMarshal(t, map[string]interface{}{
		"quality_reports": []map[string]interface{}{
			{
				"test_function": "TestDoWork",
				"test_location": "pkg/work_test.go:15",
				"target_function": map[string]interface{}{
					"package": "github.com/example/pkg", "function": "DoWork",
					"signature": "func DoWork() error", "location": "pkg/work.go:10",
				},
				"contract_coverage": map[string]interface{}{
					"percentage": 50.0, "covered_count": 1, "total_contractual": 2,
					"gaps": []map[string]interface{}{
						{"id": "se-aabbccdd", "type": "ErrorReturn", "tier": "P0",
							"location": "pkg/work.go:20", "description": "returns error",
							"target": "error"},
					},
					"gap_hints": []string{"assert err != nil"},
					"discarded_returns": []map[string]interface{}{
						{"id": "se-11223344", "type": "ReturnValue", "tier": "P0",
							"location": "pkg/work.go:25", "description": "returns int",
							"target": "int"},
					},
					"discarded_return_hints": []string{"capture return value"},
				},
				"over_specification": map[string]interface{}{
					"count": 0, "ratio": 0.0,
					"incidental_assertions": []interface{}{},
					"suggestions":           []interface{}{},
				},
				"ambiguous_effects": []map[string]interface{}{
					{"id": "se-55667788", "type": "LogWrite", "tier": "P2",
						"location": "pkg/work.go:30", "description": "writes to log",
						"target": "logger",
						"classification": map[string]interface{}{
							"label": "ambiguous", "confidence": 55,
							"signals": []map[string]interface{}{
								{"source": "naming", "weight": 10},
							},
							"reasoning": "unclear intent",
						}},
				},
				"unmapped_assertions":            []interface{}{},
				"assertion_count":                3,
				"assertion_detection_confidence": 85,
				"metadata": map[string]interface{}{
					"gaze_version": "dev", "language": "go", "language_version": "go1.24",
					"duration_ms": 100, "warnings": []interface{}{},
				},
			},
		},
		"quality_summary": map[string]interface{}{
			"total_tests": 1, "average_contract_coverage": 50.0,
			"total_over_specifications":      0,
			"assertion_detection_confidence": 85,
			"ssa_degraded":                   false,
			"worst_coverage_tests": []map[string]interface{}{
				{
					"test_function": "TestDoWork",
					"test_location": "pkg/work_test.go:15",
					"target_function": map[string]interface{}{
						"package": "github.com/example/pkg", "function": "DoWork",
						"signature": "func DoWork() error", "location": "pkg/work.go:10",
					},
					"contract_coverage": map[string]interface{}{
						"percentage": 50.0, "covered_count": 1, "total_contractual": 2,
						"gaps": []interface{}{}, "discarded_returns": []interface{}{},
					},
					"over_specification": map[string]interface{}{
						"count": 0, "ratio": 0.0,
						"incidental_assertions": []interface{}{},
						"suggestions":           []interface{}{},
					},
					"ambiguous_effects":              []interface{}{},
					"unmapped_assertions":            []interface{}{},
					"assertion_count":                3,
					"assertion_detection_confidence": 85,
					"metadata": map[string]interface{}{
						"gaze_version": "dev", "language": "go", "language_version": "go1.24",
						"duration_ms": 100, "warnings": []interface{}{},
					},
				},
			},
		},
	})

	// Classify section with signals on side effects.
	classifyJSON := mustMarshal(t, map[string]interface{}{
		"version": "dev",
		"results": []map[string]interface{}{
			{
				"target": map[string]interface{}{
					"package": "github.com/example/pkg", "function": "DoWork",
					"signature": "func DoWork() error", "location": "pkg/work.go:10",
				},
				"side_effects": []map[string]interface{}{
					{
						"id": "se-aabbccdd", "type": "ErrorReturn", "tier": "P0",
						"location": "pkg/work.go:20", "description": "returns error",
						"target": "error",
						"classification": map[string]interface{}{
							"label": "contractual", "confidence": 90,
							"signals": []map[string]interface{}{
								{"source": "interface", "weight": 25},
								{"source": "naming", "weight": 15},
								{"source": "godoc", "weight": 20,
									"source_file": "pkg/work.go",
									"excerpt":     "DoWork returns an error",
									"reasoning":   "GoDoc mentions error return"},
							},
							"reasoning": "error return is contractual",
						},
					},
				},
				"metadata": map[string]interface{}{
					"gaze_version": "dev", "language": "go", "language_version": "go1.24",
					"duration_ms": 50, "warnings": []interface{}{},
				},
			},
		},
	})

	// Docscan section with content (envelope format).
	docscanJSON := mustMarshal(t, map[string]interface{}{
		"documents": []map[string]interface{}{
			{"path": "README.md", "content": "# Project\nSome content here.", "priority": 2},
			{"path": "CONTRIBUTING.md", "content": "# Contributing\nGuidelines.", "priority": 3},
		},
		"api_coverage": nil,
	})

	return &ReportPayload{
		Summary: ReportSummary{
			CRAPload:            intPtr(1),
			GazeCRAPload:        intPtr(0),
			AvgContractCoverage: intPtr(50),
			SSADegraded:         false,
			Contractual:         1,
			Ambiguous:           1,
			Incidental:          0,
		},
		CRAP:     crapJSON,
		Quality:  qualityJSON,
		Classify: classifyJSON,
		Docscan:  docscanJSON,
		Errors:   PayloadErrors{},
	}
}

// mustMarshal marshals v to json.RawMessage, failing the test on error.
func mustMarshal(t *testing.T, v interface{}) json.RawMessage {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("mustMarshal: %v", err)
	}
	return json.RawMessage(data)
}

// TestCompactForAI_ValidJSON verifies that a fully populated payload
// produces valid JSON from CompactForAI.
func TestCompactForAI_ValidJSON(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	if !json.Valid(data) {
		t.Fatalf("CompactForAI produced invalid JSON: %s", data)
	}

	// Verify it can be unmarshalled into a generic map.
	var m map[string]interface{}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal compact JSON: %v", err)
	}

	// Verify all top-level keys are present.
	for _, key := range []string{"summary", "crap", "quality", "classify", "docscan", "errors"} {
		if _, ok := m[key]; !ok {
			t.Errorf("missing top-level key %q in compact JSON", key)
		}
	}
}

// TestCompactForAI_Summary verifies that summary fields appear with
// correct JSON keys in the compact output.
func TestCompactForAI_Summary(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var summary map[string]interface{}
	if err := json.Unmarshal(m["summary"], &summary); err != nil {
		t.Fatalf("Unmarshal summary: %v", err)
	}

	expectedKeys := []string{
		"crapload", "gaze_crapload", "avg_contract_coverage",
		"ssa_degraded", "ssa_degraded_packages",
		"contractual", "ambiguous", "incidental",
	}
	for _, key := range expectedKeys {
		if _, ok := summary[key]; !ok {
			t.Errorf("missing summary key %q", key)
		}
	}

	// Verify values.
	v, ok := summary["crapload"].(float64)
	if !ok {
		t.Fatal("crapload is not a number")
	}
	if int(v) != 1 {
		t.Errorf("crapload = %v, want 1", v)
	}
	v, ok = summary["avg_contract_coverage"].(float64)
	if !ok {
		t.Fatal("avg_contract_coverage is not a number")
	}
	if int(v) != 50 {
		t.Errorf("avg_contract_coverage = %v, want 50", v)
	}
	v, ok = summary["contractual"].(float64)
	if !ok {
		t.Fatal("contractual is not a number")
	}
	if int(v) != 1 {
		t.Errorf("contractual = %v, want 1", v)
	}
}

// TestCompactForAI_DocscanContentStripped verifies that docscan entries
// have their Content field stripped, keeping only Path and Priority.
func TestCompactForAI_DocscanContentStripped(t *testing.T) {
	// Build payload with 3 docs, each with 10KB content (envelope format).
	bigContent := strings.Repeat("x", 10*1024)
	docscanJSON := mustMarshal(t, map[string]interface{}{
		"documents": []map[string]interface{}{
			{"path": "README.md", "content": bigContent, "priority": 2},
			{"path": "CONTRIBUTING.md", "content": bigContent, "priority": 3},
			{"path": "docs/ARCH.md", "content": bigContent, "priority": 3},
		},
		"api_coverage": nil,
	})

	payload := &ReportPayload{
		Docscan: docscanJSON,
		Errors:  PayloadErrors{},
	}

	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	// Content must not appear in compact output.
	if strings.Contains(string(data), bigContent[:100]) {
		t.Error("compact output still contains docscan content")
	}

	// Parse docscan envelope.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var docscanEnv map[string]json.RawMessage
	if err := json.Unmarshal(m["docscan"], &docscanEnv); err != nil {
		t.Fatalf("Unmarshal docscan envelope: %v", err)
	}

	var docs []map[string]interface{}
	if err := json.Unmarshal(docscanEnv["documents"], &docs); err != nil {
		t.Fatalf("Unmarshal docscan documents: %v", err)
	}

	if len(docs) != 3 {
		t.Fatalf("docscan documents length = %d, want 3", len(docs))
	}

	for i, doc := range docs {
		if _, ok := doc["content"]; ok {
			t.Errorf("doc[%d] still has content field", i)
		}
		if _, ok := doc["path"]; !ok {
			t.Errorf("doc[%d] missing path field", i)
		}
		if _, ok := doc["priority"]; !ok {
			t.Errorf("doc[%d] missing priority field", i)
		}
	}
}

// TestCompactForAI_DocscanEmpty verifies that an empty docscan array
// stays [] (not null) in compact output.
func TestCompactForAI_DocscanEmpty(t *testing.T) {
	payload := &ReportPayload{
		Docscan: json.RawMessage(`{"documents":[],"api_coverage":null}`),
		Errors:  PayloadErrors{},
	}

	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Docscan should be an envelope with empty documents.
	var docscanEnv map[string]json.RawMessage
	if err := json.Unmarshal(m["docscan"], &docscanEnv); err != nil {
		t.Fatalf("Unmarshal docscan envelope: %v", err)
	}
	if string(docscanEnv["documents"]) != "[]" {
		t.Errorf("docscan.documents = %s, want []", docscanEnv["documents"])
	}
}

// TestCompactForAI_DocscanNil verifies that nil docscan stays null
// in compact output.
func TestCompactForAI_DocscanNil(t *testing.T) {
	payload := &ReportPayload{
		Docscan: nil,
		Errors:  PayloadErrors{},
	}

	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if string(m["docscan"]) != "null" {
		t.Errorf("docscan = %s, want null", m["docscan"])
	}
}

// TestCompactForAI_QualityGapsReducedToIDs verifies that contract
// coverage gaps are replaced with gap_ids (string array).
func TestCompactForAI_QualityGapsReducedToIDs(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	// Parse quality reports.
	reports := extractQualityReports(t, data)
	if len(reports) == 0 {
		t.Fatal("no quality reports in compact output")
	}

	cc, ok := reports[0]["contract_coverage"].(map[string]interface{})
	if !ok {
		t.Fatal("contract_coverage is not an object")
	}

	// Must have gap_ids, not gaps.
	if _, ok := cc["gaps"]; ok {
		t.Error("compact quality still has 'gaps' field (should be 'gap_ids')")
	}
	gapIDs, ok := cc["gap_ids"].([]interface{})
	if !ok {
		t.Fatal("compact quality missing 'gap_ids' field")
	}
	if len(gapIDs) != 1 {
		t.Fatalf("gap_ids length = %d, want 1", len(gapIDs))
	}
	gapID, ok := gapIDs[0].(string)
	if !ok {
		t.Fatal("gap_ids[0] is not a string")
	}
	if gapID != "se-aabbccdd" {
		t.Errorf("gap_ids[0] = %v, want se-aabbccdd", gapID)
	}
}

// TestCompactForAI_QualityDiscardedReturnsReducedToIDs verifies that
// discarded_returns are replaced with discarded_return_ids.
func TestCompactForAI_QualityDiscardedReturnsReducedToIDs(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	reports := extractQualityReports(t, data)
	if len(reports) == 0 {
		t.Fatal("no quality reports in compact output")
	}

	cc, ok := reports[0]["contract_coverage"].(map[string]interface{})
	if !ok {
		t.Fatal("contract_coverage is not an object")
	}

	if _, ok := cc["discarded_returns"]; ok {
		t.Error("compact quality still has 'discarded_returns' field")
	}
	drIDs, ok := cc["discarded_return_ids"].([]interface{})
	if !ok {
		t.Fatal("compact quality missing 'discarded_return_ids' field")
	}
	if len(drIDs) != 1 {
		t.Fatalf("discarded_return_ids length = %d, want 1", len(drIDs))
	}
	drID, ok := drIDs[0].(string)
	if !ok {
		t.Fatal("discarded_return_ids[0] is not a string")
	}
	if drID != "se-11223344" {
		t.Errorf("discarded_return_ids[0] = %v, want se-11223344", drIDs[0])
	}
}

// TestCompactForAI_QualityAmbiguousEffectsReducedToIDs verifies that
// ambiguous_effects are replaced with ambiguous_effect_ids.
func TestCompactForAI_QualityAmbiguousEffectsReducedToIDs(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	reports := extractQualityReports(t, data)
	if len(reports) == 0 {
		t.Fatal("no quality reports in compact output")
	}

	if _, ok := reports[0]["ambiguous_effects"]; ok {
		t.Error("compact quality still has 'ambiguous_effects' field")
	}
	aeIDs, ok := reports[0]["ambiguous_effect_ids"].([]interface{})
	if !ok {
		t.Fatal("compact quality missing 'ambiguous_effect_ids' field")
	}
	if len(aeIDs) != 1 {
		t.Fatalf("ambiguous_effect_ids length = %d, want 1", len(aeIDs))
	}
	aeID, ok := aeIDs[0].(string)
	if !ok {
		t.Fatal("ambiguous_effect_ids[0] is not a string")
	}
	if aeID != "se-55667788" {
		t.Errorf("ambiguous_effect_ids[0] = %v, want se-55667788", aeID)
	}
}

// TestCompactForAI_QualityHintsPreserved verifies that gap_hints and
// discarded_return_hints are preserved in compact output.
func TestCompactForAI_QualityHintsPreserved(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	reports := extractQualityReports(t, data)
	if len(reports) == 0 {
		t.Fatal("no quality reports in compact output")
	}

	cc, ok := reports[0]["contract_coverage"].(map[string]interface{})
	if !ok {
		t.Fatal("contract_coverage is not an object")
	}

	gapHints, ok := cc["gap_hints"].([]interface{})
	if !ok {
		t.Fatal("compact quality missing 'gap_hints' field")
	}
	if len(gapHints) != 1 {
		t.Fatalf("gap_hints length = %d, want 1", len(gapHints))
	}
	hint, ok := gapHints[0].(string)
	if !ok || hint != "assert err != nil" {
		t.Errorf("gap_hints = %v, want [\"assert err != nil\"]", gapHints)
	}

	drHints, ok := cc["discarded_return_hints"].([]interface{})
	if !ok {
		t.Fatal("compact quality missing 'discarded_return_hints' field")
	}
	if len(drHints) != 1 {
		t.Fatalf("discarded_return_hints length = %d, want 1", len(drHints))
	}
	drHint, ok := drHints[0].(string)
	if !ok || drHint != "capture return value" {
		t.Errorf("discarded_return_hints = %v, want [\"capture return value\"]", drHints)
	}
}

// TestCompactForAI_QualityCrossRefResilience verifies that when
// classify is nil, quality gaps still produce IDs from the quality
// data alone (no cross-reference dependency).
func TestCompactForAI_QualityCrossRefResilience(t *testing.T) {
	payload := buildFullPayload(t)
	payload.Classify = nil // Simulate classify step failure.

	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	reports := extractQualityReports(t, data)
	if len(reports) == 0 {
		t.Fatal("no quality reports in compact output")
	}

	cc, ok := reports[0]["contract_coverage"].(map[string]interface{})
	if !ok {
		t.Fatal("contract_coverage is not an object")
	}
	gapIDs, ok := cc["gap_ids"].([]interface{})
	if !ok {
		t.Fatal("compact quality missing 'gap_ids' when classify is nil")
	}
	if len(gapIDs) != 1 {
		t.Errorf("gap_ids length = %d, want 1 (resilient to nil classify)", len(gapIDs))
	}
}

// TestCompactForAI_ClassifySignalsStripped verifies that classification
// signals are omitted while label, confidence, and reasoning are preserved.
func TestCompactForAI_ClassifySignalsStripped(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	var classify map[string]interface{}
	if err := json.Unmarshal(m["classify"], &classify); err != nil {
		t.Fatalf("Unmarshal classify: %v", err)
	}

	results, ok := classify["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("no classify results in compact output")
	}

	result, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatal("classify result is not an object")
	}
	effects, ok := result["side_effects"].([]interface{})
	if !ok || len(effects) == 0 {
		t.Fatal("no side effects in compact classify")
	}

	effect, ok := effects[0].(map[string]interface{})
	if !ok {
		t.Fatal("side effect is not an object")
	}
	cls, ok := effect["classification"].(map[string]interface{})
	if !ok {
		t.Fatal("classification is not an object")
	}

	// Signals must be absent.
	if _, ok := cls["signals"]; ok {
		t.Error("compact classify still has 'signals' field")
	}

	// Label, confidence, reasoning must be present.
	label, ok := cls["label"].(string)
	if !ok || label != "contractual" {
		t.Errorf("label = %v, want contractual", cls["label"])
	}
	conf, ok := cls["confidence"].(float64)
	if !ok || int(conf) != 90 {
		t.Errorf("confidence = %v, want 90", cls["confidence"])
	}
	reasoning, ok := cls["reasoning"].(string)
	if !ok || reasoning != "error return is contractual" {
		t.Errorf("reasoning = %v, want 'error return is contractual'", cls["reasoning"])
	}
}

// TestCompactForAI_CRAPWorstOffendersOmitted verifies that worst_crap,
// worst_gaze_crap, and recommended_actions are omitted from the compact
// CRAP summary.
func TestCompactForAI_CRAPWorstOffendersOmitted(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	s := string(data)
	if strings.Contains(s, "worst_crap") {
		t.Error("compact CRAP still contains 'worst_crap'")
	}
	if strings.Contains(s, "worst_gaze_crap") {
		t.Error("compact CRAP still contains 'worst_gaze_crap'")
	}
	if strings.Contains(s, "recommended_actions") {
		t.Error("compact CRAP still contains 'recommended_actions'")
	}

	// Verify other summary fields are preserved.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var crapData map[string]json.RawMessage
	if err := json.Unmarshal(m["crap"], &crapData); err != nil {
		t.Fatalf("Unmarshal crap: %v", err)
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(crapData["summary"], &summary); err != nil {
		t.Fatalf("Unmarshal crap summary: %v", err)
	}
	craploadVal, ok := summary["crapload"].(float64)
	if !ok || int(craploadVal) != 1 {
		t.Errorf("crap summary crapload = %v, want 1", summary["crapload"])
	}
}

// TestCompactForAI_QualitySummaryDedup verifies that worst_coverage_tests
// is omitted from the compact quality summary.
func TestCompactForAI_QualitySummaryDedup(t *testing.T) {
	payload := buildFullPayload(t)
	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	if strings.Contains(string(data), "worst_coverage_tests") {
		t.Error("compact quality still contains 'worst_coverage_tests'")
	}

	// Verify quality summary scalar fields are preserved.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	var quality map[string]json.RawMessage
	if err := json.Unmarshal(m["quality"], &quality); err != nil {
		t.Fatalf("Unmarshal quality: %v", err)
	}
	var summary map[string]interface{}
	if err := json.Unmarshal(quality["quality_summary"], &summary); err != nil {
		t.Fatalf("Unmarshal quality summary: %v", err)
	}
	totalTests, ok := summary["total_tests"].(float64)
	if !ok || int(totalTests) != 1 {
		t.Errorf("quality summary total_tests = %v, want 1", summary["total_tests"])
	}
}

// TestCompactForAI_AllStepsFailed verifies that when all steps failed
// (all data fields nil, all error fields populated), CompactForAI
// produces valid JSON.
func TestCompactForAI_AllStepsFailed(t *testing.T) {
	crapErr := "crap failed"
	qualErr := "quality failed"
	classErr := "classify failed"
	docErr := "docscan failed"

	payload := &ReportPayload{
		CRAP:     nil,
		Quality:  nil,
		Classify: nil,
		Docscan:  nil,
		Errors: PayloadErrors{
			CRAP:     &crapErr,
			Quality:  &qualErr,
			Classify: &classErr,
			Docscan:  &docErr,
		},
	}

	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	if !json.Valid(data) {
		t.Fatalf("invalid JSON from all-failed payload: %s", data)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	// Data fields must be null.
	for _, key := range []string{"crap", "quality", "classify", "docscan"} {
		if string(m[key]) != "null" {
			t.Errorf("%s = %s, want null", key, m[key])
		}
	}

	// Errors must be populated.
	var errs map[string]interface{}
	if err := json.Unmarshal(m["errors"], &errs); err != nil {
		t.Fatalf("Unmarshal errors: %v", err)
	}
	for _, key := range []string{"crap", "quality", "classify", "docscan"} {
		if errs[key] == nil {
			t.Errorf("errors.%s should be non-null", key)
		}
	}
}

// TestCompactForAI_StepFailurePreserved verifies that when a single
// step fails (Quality nil with error), the quality field is null and
// the error is present, while other fields are compacted normally.
func TestCompactForAI_StepFailurePreserved(t *testing.T) {
	qualErr := "SSA construction failed"
	payload := buildFullPayload(t)
	payload.Quality = nil
	payload.Errors.Quality = &qualErr

	data, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}

	if string(m["quality"]) != "null" {
		t.Errorf("quality = %s, want null", m["quality"])
	}

	// CRAP should still be compacted (not null).
	if string(m["crap"]) == "null" {
		t.Error("crap should not be null when only quality failed")
	}

	// Error should be present.
	var errs map[string]interface{}
	if err := json.Unmarshal(m["errors"], &errs); err != nil {
		t.Fatalf("Unmarshal errors: %v", err)
	}
	qualErrVal, ok := errs["quality"].(string)
	if !ok {
		t.Fatal("errors.quality should be a non-null string")
	}
	if qualErrVal != qualErr {
		t.Errorf("errors.quality = %v, want %q", qualErrVal, qualErr)
	}
}

// TestCompactForAI_SizeBudget verifies that a synthetic payload with
// ~200 functions, ~100 test-target pairs, and ~30 docs compacts under
// 300KB.
func TestCompactForAI_SizeBudget(t *testing.T) {
	// Build a large CRAP section with 200 scores.
	scores := make([]map[string]interface{}, 200)
	for i := range scores {
		scores[i] = map[string]interface{}{
			"package":  fmt.Sprintf("github.com/example/pkg%d", i),
			"function": fmt.Sprintf("Func%d", i),
			"file":     fmt.Sprintf("pkg%d/file.go", i),
			"line":     i + 1, "complexity": 5 + (i % 20),
			"line_coverage": 50.0 + float64(i%50),
			"crap":          10.0 + float64(i%30),
		}
	}
	crapJSON := mustMarshal(t, map[string]interface{}{
		"scores": scores,
		"summary": map[string]interface{}{
			"total_functions": 200, "avg_complexity": 15.0,
			"avg_line_coverage": 75.0, "avg_crap": 20.0,
			"crapload": 50, "crap_threshold": 15.0,
			"worst_crap":          scores[:10],
			"worst_gaze_crap":     scores[:10],
			"recommended_actions": scores[:20],
		},
	})

	// Build quality with 100 test-target pairs, each with gaps.
	qualReports := make([]map[string]interface{}, 100)
	for i := range qualReports {
		qualReports[i] = map[string]interface{}{
			"test_function": fmt.Sprintf("TestFunc%d", i),
			"test_location": fmt.Sprintf("pkg/file_test.go:%d", i+1),
			"target_function": map[string]interface{}{
				"package":   fmt.Sprintf("github.com/example/pkg%d", i),
				"function":  fmt.Sprintf("Func%d", i),
				"signature": fmt.Sprintf("func Func%d() error", i),
				"location":  fmt.Sprintf("pkg/file.go:%d", i+1),
			},
			"contract_coverage": map[string]interface{}{
				"percentage": 50.0, "covered_count": 1, "total_contractual": 2,
				"gaps": []map[string]interface{}{
					{"id": fmt.Sprintf("se-%08x", i), "type": "ErrorReturn",
						"tier": "P0", "location": fmt.Sprintf("pkg/file.go:%d", i+10),
						"description": "returns error", "target": "error"},
				},
				"gap_hints":         []string{"assert err"},
				"discarded_returns": []interface{}{},
			},
			"over_specification": map[string]interface{}{
				"count": 0, "ratio": 0.0,
				"incidental_assertions": []interface{}{},
				"suggestions":           []interface{}{},
			},
			"ambiguous_effects":              []interface{}{},
			"unmapped_assertions":            []interface{}{},
			"assertion_count":                2,
			"assertion_detection_confidence": 90,
			"metadata": map[string]interface{}{
				"gaze_version": "dev", "language": "go", "language_version": "go1.24",
				"duration_ms": 10, "warnings": []interface{}{},
			},
		}
	}
	qualityJSON := mustMarshal(t, map[string]interface{}{
		"quality_reports": qualReports,
		"quality_summary": map[string]interface{}{
			"total_tests": 100, "average_contract_coverage": 50.0,
			"total_over_specifications":      0,
			"assertion_detection_confidence": 90,
			"ssa_degraded":                   false,
			"worst_coverage_tests":           qualReports[:5],
		},
	})

	// Build docscan with 30 docs, each with 10KB content (envelope format).
	docs := make([]map[string]interface{}, 30)
	bigContent := strings.Repeat("Documentation content. ", 500) // ~11KB
	for i := range docs {
		docs[i] = map[string]interface{}{
			"path":     fmt.Sprintf("docs/doc%d.md", i),
			"content":  bigContent,
			"priority": 3,
		}
	}
	docscanJSON := mustMarshal(t, map[string]interface{}{
		"documents":    docs,
		"api_coverage": nil,
	})

	// Classify with 200 results, each with 3 signals.
	classifyResults := make([]map[string]interface{}, 200)
	for i := range classifyResults {
		classifyResults[i] = map[string]interface{}{
			"target": map[string]interface{}{
				"package":   fmt.Sprintf("github.com/example/pkg%d", i),
				"function":  fmt.Sprintf("Func%d", i),
				"signature": fmt.Sprintf("func Func%d() error", i),
				"location":  fmt.Sprintf("pkg/file.go:%d", i+1),
			},
			"side_effects": []map[string]interface{}{
				{
					"id": fmt.Sprintf("se-%08x", i), "type": "ErrorReturn",
					"tier": "P0", "location": fmt.Sprintf("pkg/file.go:%d", i+10),
					"description": "returns error", "target": "error",
					"classification": map[string]interface{}{
						"label": "contractual", "confidence": 90,
						"signals": []map[string]interface{}{
							{"source": "interface", "weight": 25},
							{"source": "naming", "weight": 15},
							{"source": "godoc", "weight": 20,
								"source_file": "pkg/file.go",
								"excerpt":     "Returns an error when the operation fails",
								"reasoning":   "GoDoc explicitly documents error return"},
						},
						"reasoning": "error return is contractual",
					},
				},
			},
			"metadata": map[string]interface{}{
				"gaze_version": "dev", "language": "go", "language_version": "go1.24",
				"duration_ms": 10, "warnings": []interface{}{},
			},
		}
	}
	classifyJSON := mustMarshal(t, map[string]interface{}{
		"version": "dev",
		"results": classifyResults,
	})

	payload := &ReportPayload{
		Summary:  ReportSummary{CRAPload: intPtr(50), GazeCRAPload: intPtr(10), AvgContractCoverage: intPtr(50)},
		CRAP:     crapJSON,
		Quality:  qualityJSON,
		Classify: classifyJSON,
		Docscan:  docscanJSON,
		Errors:   PayloadErrors{},
	}

	// Verify full marshal is large.
	fullData, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal: %v", err)
	}
	t.Logf("Full payload size: %d bytes", len(fullData))

	// Compact must be under 300KB.
	compactData, err := payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}
	t.Logf("Compact payload size: %d bytes", len(compactData))

	if len(compactData) > 300*1024 {
		t.Errorf("compact payload = %d bytes, want <= %d bytes (300KB)",
			len(compactData), 300*1024)
	}

	// Compact must be significantly smaller than full.
	if len(compactData) >= len(fullData) {
		t.Errorf("compact (%d) should be smaller than full (%d)",
			len(compactData), len(fullData))
	}
}

// TestCompactForAI_FullMarshalUnchanged verifies that json.Marshal on
// the same payload still produces the full output with content, signals,
// and worst offender lists — CompactForAI does not mutate the payload.
func TestCompactForAI_FullMarshalUnchanged(t *testing.T) {
	payload := buildFullPayload(t)

	// Capture full marshal before compact.
	fullBefore, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal (before): %v", err)
	}

	// Run compact.
	_, err = payload.CompactForAI()
	if err != nil {
		t.Fatalf("CompactForAI: %v", err)
	}

	// Full marshal after compact must be identical.
	fullAfter, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("json.Marshal (after): %v", err)
	}

	if string(fullBefore) != string(fullAfter) {
		t.Error("json.Marshal output changed after CompactForAI — payload was mutated")
	}

	// Verify full output still contains content, signals, worst lists.
	s := string(fullAfter)
	if !strings.Contains(s, "content") {
		t.Error("full marshal missing 'content' (docscan)")
	}
	if !strings.Contains(s, "signals") {
		t.Error("full marshal missing 'signals' (classify)")
	}
	if !strings.Contains(s, "worst_crap") {
		t.Error("full marshal missing 'worst_crap' (CRAP)")
	}
}

// TestExtractEffectIDs verifies the nil/empty/populated behavior of
// extractEffectIDs, which must preserve the distinction between nil
// (JSON null) and empty (JSON []) slices.
func TestExtractEffectIDs(t *testing.T) {
	tests := []struct {
		name   string
		input  []taxonomy.SideEffect
		want   []string // nil means expect nil output
		wantNl bool     // true = expect nil result (vs non-nil empty)
	}{
		{
			name:   "nil input returns nil output",
			input:  nil,
			want:   nil,
			wantNl: true,
		},
		{
			name:   "empty slice returns empty non-nil slice",
			input:  []taxonomy.SideEffect{},
			want:   []string{},
			wantNl: false,
		},
		{
			name: "populated slice extracts ID strings",
			input: []taxonomy.SideEffect{
				{ID: "se-aabbccdd", Type: "ErrorReturn", Tier: "P0"},
				{ID: "se-11223344", Type: "ReturnValue", Tier: "P0"},
				{ID: "se-55667788", Type: "LogWrite", Tier: "P2"},
			},
			want:   []string{"se-aabbccdd", "se-11223344", "se-55667788"},
			wantNl: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractEffectIDs(tt.input)

			// Check nil vs non-nil distinction.
			if tt.wantNl {
				if got != nil {
					t.Fatalf("extractEffectIDs(%v) = %v, want nil", tt.input, got)
				}
				return
			}
			if got == nil {
				t.Fatalf("extractEffectIDs(%v) = nil, want non-nil empty slice", tt.input)
			}

			// Check length.
			if len(got) != len(tt.want) {
				t.Fatalf("extractEffectIDs length = %d, want %d", len(got), len(tt.want))
			}

			// Check values.
			for i, wantID := range tt.want {
				if got[i] != wantID {
					t.Errorf("extractEffectIDs[%d] = %q, want %q", i, got[i], wantID)
				}
			}
		})
	}
}

// TestRunTextPath_CompactPayloadReceived verifies that the compact payload
// produced by CompactForAI is what the AI adapter actually receives when
// runTextPath is called. This is an integration test between the compact
// encoding and the text path plumbing.
func TestRunTextPath_CompactPayloadReceived(t *testing.T) {
	payload := buildFullPayload(t)

	fa := &FakeAdapter{Response: "# Report\n\nFormatted output."}
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	opts := RunnerOptions{
		Format:  "text",
		Adapter: fa,
		Stdout:  &stdout,
		Stderr:  &stderr,
	}

	err := runTextPath(payload, opts)
	if err != nil {
		t.Fatalf("runTextPath: %v", err)
	}

	// FakeAdapter must have been called exactly once.
	if len(fa.Calls) != 1 {
		t.Fatalf("expected 1 adapter call, got %d", len(fa.Calls))
	}

	adapterPayload := fa.Calls[0].Payload

	// The adapter payload must be valid JSON.
	if !json.Valid(adapterPayload) {
		t.Fatalf("adapter received invalid JSON: %s", adapterPayload)
	}

	// Parse into a generic map for field assertions.
	var m map[string]json.RawMessage
	if err := json.Unmarshal(adapterPayload, &m); err != nil {
		t.Fatalf("Unmarshal adapter payload: %v", err)
	}

	// Assertion 1: no "content" in docscan document entries.
	var docscanEnv map[string]json.RawMessage
	if err := json.Unmarshal(m["docscan"], &docscanEnv); err != nil {
		t.Fatalf("Unmarshal docscan envelope: %v", err)
	}
	var docs []map[string]interface{}
	if err := json.Unmarshal(docscanEnv["documents"], &docs); err != nil {
		t.Fatalf("Unmarshal docscan documents: %v", err)
	}
	for i, doc := range docs {
		if _, ok := doc["content"]; ok {
			t.Errorf("docscan[%d] still has 'content' field in adapter payload", i)
		}
	}

	// Assertion 2: no "signals" in classify side effect classifications.
	var classify map[string]interface{}
	if err := json.Unmarshal(m["classify"], &classify); err != nil {
		t.Fatalf("Unmarshal classify: %v", err)
	}
	results, ok := classify["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("no classify results in adapter payload")
	}
	result, ok := results[0].(map[string]interface{})
	if !ok {
		t.Fatal("classify result is not an object")
	}
	effects, ok := result["side_effects"].([]interface{})
	if !ok || len(effects) == 0 {
		t.Fatal("no side effects in classify result")
	}
	effect, ok := effects[0].(map[string]interface{})
	if !ok {
		t.Fatal("side effect is not an object")
	}
	cls, ok := effect["classification"].(map[string]interface{})
	if !ok {
		t.Fatal("classification is not an object")
	}
	if _, ok := cls["signals"]; ok {
		t.Error("classify classification still has 'signals' in adapter payload")
	}

	// Assertion 3: no "worst_crap" in CRAP summary.
	if strings.Contains(string(m["crap"]), `"worst_crap"`) {
		t.Error("CRAP section still contains 'worst_crap' in adapter payload")
	}

	// Assertion 4: "summary" key present at top level.
	if _, ok := m["summary"]; !ok {
		t.Error("adapter payload missing top-level 'summary' key")
	}
}

// TestCompactForAI_MalformedInput verifies that CompactForAI returns a
// descriptive error when a section contains malformed JSON, and that the
// error message identifies which section failed.
func TestCompactForAI_MalformedInput(t *testing.T) {
	malformed := json.RawMessage(`{not valid json}`)

	tests := []struct {
		name    string
		payload *ReportPayload
		wantSub string // substring expected in error message
	}{
		{
			name: "malformed CRAP",
			payload: &ReportPayload{
				CRAP:   malformed,
				Errors: PayloadErrors{},
			},
			wantSub: "CRAP",
		},
		{
			name: "malformed quality",
			payload: &ReportPayload{
				Quality: malformed,
				Errors:  PayloadErrors{},
			},
			wantSub: "quality",
		},
		{
			name: "malformed classify",
			payload: &ReportPayload{
				Classify: malformed,
				Errors:   PayloadErrors{},
			},
			wantSub: "classify",
		},
		{
			name: "malformed docscan",
			payload: &ReportPayload{
				Docscan: malformed,
				Errors:  PayloadErrors{},
			},
			wantSub: "docscan",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.payload.CompactForAI()
			if err == nil {
				t.Fatalf("CompactForAI with malformed %s: expected error, got nil", tt.name)
			}
			errMsg := err.Error()
			// Error message must identify the section that failed.
			// The compact.go wraps errors with section context like
			// "compacting CRAP:", "compacting quality:", etc.
			if !strings.Contains(strings.ToLower(errMsg), strings.ToLower(tt.wantSub)) {
				t.Errorf("error %q does not contain section name %q", errMsg, tt.wantSub)
			}
		})
	}
}

// extractQualityReports parses the compact JSON and returns the
// quality_reports array as generic maps.
func extractQualityReports(t *testing.T, data []byte) []map[string]interface{} {
	t.Helper()

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Unmarshal top-level: %v", err)
	}

	var quality map[string]json.RawMessage
	if err := json.Unmarshal(m["quality"], &quality); err != nil {
		t.Fatalf("Unmarshal quality: %v", err)
	}

	var reports []map[string]interface{}
	if err := json.Unmarshal(quality["quality_reports"], &reports); err != nil {
		t.Fatalf("Unmarshal quality_reports: %v", err)
	}

	return reports
}
