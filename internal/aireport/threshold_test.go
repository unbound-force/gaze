package aireport

import (
	"testing"
)

// TestEvaluateThresholds_NilConfig verifies that nil thresholds (not provided)
// result in no results and allPassed=true.
func TestEvaluateThresholds_NilConfig(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{CRAPload: intPtr(10), GazeCRAPload: intPtr(5), AvgContractCoverage: intPtr(30)},
	}
	results, passed := EvaluateThresholds(ThresholdConfig{}, payload)
	if !passed {
		t.Error("expected passed=true when no thresholds set")
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results, got %d", len(results))
	}
}

// TestEvaluateThresholds_ZeroThresholdWithZeroActual verifies that *0 threshold
// with actual=0 passes (zero is a valid live threshold, and 0 <= 0).
func TestEvaluateThresholds_ZeroThresholdWithZeroActual(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{CRAPload: intPtr(0)},
	}
	cfg := ThresholdConfig{MaxCrapload: intPtr(0)}
	results, passed := EvaluateThresholds(cfg, payload)
	if !passed {
		t.Error("expected passed=true when actual=0 and limit=0")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Error("expected result.Passed=true")
	}
	if results[0].Name != "CRAPload" {
		t.Errorf("expected Name=CRAPload, got %q", results[0].Name)
	}
	if results[0].Actual == nil || *results[0].Actual != 0 {
		t.Errorf("expected Actual=0, got %v", results[0].Actual)
	}
	if results[0].Limit != 0 {
		t.Errorf("expected Limit=0, got %d", results[0].Limit)
	}
}

// TestEvaluateThresholds_ZeroThresholdWithPositiveActual verifies that *0 threshold
// with actual>0 fails (zero is a live threshold).
func TestEvaluateThresholds_ZeroThresholdWithPositiveActual(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{CRAPload: intPtr(3)},
	}
	cfg := ThresholdConfig{MaxCrapload: intPtr(0)}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false when actual=3 and limit=0")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("expected result.Passed=false")
	}
	if results[0].Actual == nil || *results[0].Actual != 3 {
		t.Errorf("expected Actual=3, got %v", results[0].Actual)
	}
}

// TestEvaluateThresholds_BelowLimit verifies that actual < limit passes.
func TestEvaluateThresholds_BelowLimit(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{CRAPload: intPtr(3)},
	}
	cfg := ThresholdConfig{MaxCrapload: intPtr(5)}
	results, passed := EvaluateThresholds(cfg, payload)
	if !passed {
		t.Error("expected passed=true when actual < limit")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Error("expected result.Passed=true")
	}
	if results[0].Name != "CRAPload" {
		t.Errorf("expected Name=CRAPload, got %q", results[0].Name)
	}
	if results[0].Actual == nil || *results[0].Actual != 3 {
		t.Errorf("expected Actual=3, got %v", results[0].Actual)
	}
	if results[0].Limit != 5 {
		t.Errorf("expected Limit=5, got %d", results[0].Limit)
	}
}

// TestEvaluateThresholds_AboveLimit verifies that actual > limit fails.
func TestEvaluateThresholds_AboveLimit(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{CRAPload: intPtr(8)},
	}
	cfg := ThresholdConfig{MaxCrapload: intPtr(5)}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false when actual > limit")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("expected result.Passed=false")
	}
	if results[0].Name != "CRAPload" {
		t.Errorf("expected Name=CRAPload, got %q", results[0].Name)
	}
	if results[0].Actual == nil || *results[0].Actual != 8 {
		t.Errorf("expected Actual=8, got %v", results[0].Actual)
	}
	if results[0].Limit != 5 {
		t.Errorf("expected Limit=5, got %d", results[0].Limit)
	}
}

// TestEvaluateThresholds_AllThreeFields verifies that all three threshold
// fields are evaluated independently.
func TestEvaluateThresholds_AllThreeFields(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			CRAPload:            intPtr(4),
			GazeCRAPload:        intPtr(2),
			AvgContractCoverage: intPtr(60),
		},
	}
	cfg := ThresholdConfig{
		MaxCrapload:         intPtr(5),  // pass (4 <= 5)
		MaxGazeCrapload:     intPtr(1),  // fail (2 > 1)
		MinContractCoverage: intPtr(50), // pass (60 >= 50)
	}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false (GazeCRAPload exceeds limit)")
	}
	if len(results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(results))
	}

	byName := make(map[string]ThresholdResult)
	for _, r := range results {
		byName[r.Name] = r
	}

	if !byName["CRAPload"].Passed {
		t.Error("expected CRAPload to pass")
	}
	if byName["CRAPload"].Actual == nil || *byName["CRAPload"].Actual != 4 {
		t.Errorf("expected CRAPload.Actual=4, got %v", byName["CRAPload"].Actual)
	}
	if byName["CRAPload"].Limit != 5 {
		t.Errorf("expected CRAPload.Limit=5, got %d", byName["CRAPload"].Limit)
	}
	if byName["GazeCRAPload"].Passed {
		t.Error("expected GazeCRAPload to fail")
	}
	if byName["GazeCRAPload"].Actual == nil || *byName["GazeCRAPload"].Actual != 2 {
		t.Errorf("expected GazeCRAPload.Actual=2, got %v", byName["GazeCRAPload"].Actual)
	}
	if byName["GazeCRAPload"].Limit != 1 {
		t.Errorf("expected GazeCRAPload.Limit=1, got %d", byName["GazeCRAPload"].Limit)
	}
	if !byName["AvgContractCoverage"].Passed {
		t.Error("expected AvgContractCoverage to pass")
	}
	if byName["AvgContractCoverage"].Actual == nil || *byName["AvgContractCoverage"].Actual != 60 {
		t.Errorf("expected AvgContractCoverage.Actual=60, got %v", byName["AvgContractCoverage"].Actual)
	}
	if byName["AvgContractCoverage"].Limit != 50 {
		t.Errorf("expected AvgContractCoverage.Limit=50, got %d", byName["AvgContractCoverage"].Limit)
	}
}

// TestEvaluateThresholds_BothCRAPloadsFail verifies simultaneous CRAPload
// and GazeCRAPload threshold breaches.
func TestEvaluateThresholds_BothCRAPloadsFail(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			CRAPload:     intPtr(10),
			GazeCRAPload: intPtr(7),
		},
	}
	cfg := ThresholdConfig{
		MaxCrapload:     intPtr(5),
		MaxGazeCrapload: intPtr(3),
	}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false")
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for _, r := range results {
		if r.Passed {
			t.Errorf("expected %s to fail", r.Name)
		}
	}
}

// TestEvaluateThresholds_GazeCRAPloadZeroLiveThreshold verifies the US2
// scenario 7: --max-gaze-crapload=0 with positive actual fails.
func TestEvaluateThresholds_GazeCRAPloadZeroLiveThreshold(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{GazeCRAPload: intPtr(1)},
	}
	cfg := ThresholdConfig{MaxGazeCrapload: intPtr(0)}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false when GazeCRAPload=1 and limit=0")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("expected result.Passed=false")
	}
	if results[0].Name != "GazeCRAPload" {
		t.Errorf("expected Name=GazeCRAPload, got %s", results[0].Name)
	}
}

// TestEvaluateThresholds_NilPayload verifies that nil payload now correctly
// fails thresholds because all summary fields are nil (*int zero value = nil),
// not zero (int zero value = 0). This is the fix for #102: CI gates must not
// silently pass when data is unavailable.
func TestEvaluateThresholds_NilPayload(t *testing.T) {
	cfg := ThresholdConfig{MaxCrapload: intPtr(5)}
	results, passed := EvaluateThresholds(cfg, nil)
	// nil payload → zero-value summary → CRAPload=nil → FAIL (unavailable)
	if passed {
		t.Error("expected passed=false with nil payload (CRAPload is nil/unavailable)")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if results[0].Passed {
		t.Error("expected result.Passed=false")
	}
	if results[0].Actual != nil {
		t.Errorf("expected Actual=nil (unavailable), got %d", *results[0].Actual)
	}
	if results[0].Name != "CRAPload (unavailable)" {
		t.Errorf("expected Name='CRAPload (unavailable)', got %q", results[0].Name)
	}
}

// TestEvaluateThresholds_MinContractCoverageDirection verifies that
// MinContractCoverage uses >= (not <=). actual=60, limit=60 should
// pass (60 >= 60 is true). actual=59, limit=60 should fail.
func TestEvaluateThresholds_MinContractCoverageDirection(t *testing.T) {
	// Boundary: actual == limit → should pass (>= semantics).
	payload := &ReportPayload{
		Summary: ReportSummary{AvgContractCoverage: intPtr(60)},
	}
	cfg := ThresholdConfig{MinContractCoverage: intPtr(60)}
	results, passed := EvaluateThresholds(cfg, payload)
	if !passed {
		t.Error("expected passed=true when actual=60 and limit=60 (>= semantics)")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	if !results[0].Passed {
		t.Error("expected result.Passed=true at boundary")
	}
	if results[0].Name != "AvgContractCoverage" {
		t.Errorf("expected Name=AvgContractCoverage, got %q", results[0].Name)
	}
	if results[0].Actual == nil || *results[0].Actual != 60 {
		t.Errorf("expected Actual=60, got %v", results[0].Actual)
	}
	if results[0].Limit != 60 {
		t.Errorf("expected Limit=60, got %d", results[0].Limit)
	}

	// Below boundary: actual < limit → should fail.
	payload2 := &ReportPayload{
		Summary: ReportSummary{AvgContractCoverage: intPtr(59)},
	}
	results2, passed2 := EvaluateThresholds(cfg, payload2)
	if passed2 {
		t.Error("expected passed=false when actual=59 and limit=60")
	}
	if len(results2) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results2))
	}
	if results2[0].Passed {
		t.Error("expected result.Passed=false below boundary")
	}
	if results2[0].Actual == nil || *results2[0].Actual != 59 {
		t.Errorf("expected Actual=59, got %v", results2[0].Actual)
	}
}

// TestEvaluateThresholds_GazeCRAPload_ThresholdSet_DataUnavailable verifies
// that when the GazeCRAPload threshold is set but the metric is nil
// (unavailable), the result is FAIL with Actual==nil and a descriptive name.
// This is the core fix for #108: CI gates must not silently pass when data
// is missing.
func TestEvaluateThresholds_GazeCRAPload_ThresholdSet_DataUnavailable(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			CRAPload:     intPtr(2),
			GazeCRAPload: nil, // metric unavailable
		},
	}
	cfg := ThresholdConfig{MaxGazeCrapload: intPtr(5)}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false when GazeCRAPload is nil (unavailable)")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Passed {
		t.Error("expected result.Passed=false")
	}
	if r.Actual != nil {
		t.Errorf("expected Actual=nil (unavailable), got %d", *r.Actual)
	}
	if r.Name != "GazeCRAPload (unavailable)" {
		t.Errorf("expected Name='GazeCRAPload (unavailable)', got %q", r.Name)
	}
	if r.Limit != 5 {
		t.Errorf("expected Limit=5, got %d", r.Limit)
	}
}

// TestEvaluateThresholds_GazeCRAPload_ThresholdSet_DataWithinLimit verifies
// that when the GazeCRAPload threshold is set and the metric is within the
// limit, the result is PASS.
func TestEvaluateThresholds_GazeCRAPload_ThresholdSet_DataWithinLimit(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{GazeCRAPload: intPtr(3)},
	}
	cfg := ThresholdConfig{MaxGazeCrapload: intPtr(5)}
	results, passed := EvaluateThresholds(cfg, payload)
	if !passed {
		t.Error("expected passed=true when GazeCRAPload=3 <= limit=5")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.Passed {
		t.Error("expected result.Passed=true")
	}
	if r.Name != "GazeCRAPload" {
		t.Errorf("expected Name='GazeCRAPload', got %q", r.Name)
	}
	if r.Actual == nil || *r.Actual != 3 {
		t.Errorf("expected Actual=3, got %v", r.Actual)
	}
	if r.Limit != 5 {
		t.Errorf("expected Limit=5, got %d", r.Limit)
	}
}

// TestEvaluateThresholds_GazeCRAPload_ThresholdSet_DataExceedsLimit verifies
// that when the GazeCRAPload threshold is set and the metric exceeds the
// limit, the result is FAIL.
func TestEvaluateThresholds_GazeCRAPload_ThresholdSet_DataExceedsLimit(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{GazeCRAPload: intPtr(8)},
	}
	cfg := ThresholdConfig{MaxGazeCrapload: intPtr(5)}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false when GazeCRAPload=8 > limit=5")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Passed {
		t.Error("expected result.Passed=false")
	}
	if r.Name != "GazeCRAPload" {
		t.Errorf("expected Name='GazeCRAPload', got %q", r.Name)
	}
	if r.Actual == nil || *r.Actual != 8 {
		t.Errorf("expected Actual=8, got %v", r.Actual)
	}
	if r.Limit != 5 {
		t.Errorf("expected Limit=5, got %d", r.Limit)
	}
}

// TestEvaluateThresholds_GazeCRAPload_ThresholdNotSet_DataUnavailable verifies
// that when no GazeCRAPload threshold is configured and the metric is nil,
// no result is emitted (threshold skipped entirely).
func TestEvaluateThresholds_GazeCRAPload_ThresholdNotSet_DataUnavailable(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			CRAPload:     intPtr(5),
			GazeCRAPload: nil,
		},
	}
	// Only set CRAPload threshold, not GazeCRAPload.
	cfg := ThresholdConfig{MaxCrapload: intPtr(10)}
	results, passed := EvaluateThresholds(cfg, payload)
	if !passed {
		t.Error("expected passed=true (CRAPload 5 <= 10, GazeCRAPload threshold not set)")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (CRAPload only), got %d", len(results))
	}
	if results[0].Name != "CRAPload" {
		t.Errorf("expected only CRAPload result, got %q", results[0].Name)
	}
}

// TestEvaluateThresholds_GazeCRAPload_ThresholdNotSet_DataAvailable verifies
// that when no GazeCRAPload threshold is configured but the metric is
// available, no GazeCRAPload result is emitted (threshold skipped).
func TestEvaluateThresholds_GazeCRAPload_ThresholdNotSet_DataAvailable(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			CRAPload:     intPtr(5),
			GazeCRAPload: intPtr(3),
		},
	}
	// Only set CRAPload threshold, not GazeCRAPload.
	cfg := ThresholdConfig{MaxCrapload: intPtr(10)}
	results, passed := EvaluateThresholds(cfg, payload)
	if !passed {
		t.Error("expected passed=true (CRAPload 5 <= 10, GazeCRAPload threshold not set)")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result (CRAPload only), got %d", len(results))
	}
	if results[0].Name != "CRAPload" {
		t.Errorf("expected only CRAPload result, got %q", results[0].Name)
	}
}

// TestEvaluateThresholds_CRAPload_Unavailable verifies that when
// CRAPload is nil (step failed / metric unavailable) and MaxCrapload
// is configured, the result is FAIL with Actual=nil and a descriptive
// "(unavailable)" name. This is the core regression test for #102.
func TestEvaluateThresholds_CRAPload_Unavailable(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			CRAPload:     nil, // metric unavailable (step failed)
			GazeCRAPload: intPtr(2),
		},
	}
	cfg := ThresholdConfig{MaxCrapload: intPtr(5)}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false when CRAPload is nil (unavailable)")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Passed {
		t.Error("expected result.Passed=false")
	}
	if r.Actual != nil {
		t.Errorf("expected Actual=nil (unavailable), got %d", *r.Actual)
	}
	if r.Name != "CRAPload (unavailable)" {
		t.Errorf("expected Name='CRAPload (unavailable)', got %q", r.Name)
	}
	if r.Limit != 5 {
		t.Errorf("expected Limit=5, got %d", r.Limit)
	}
}

// TestEvaluateThresholds_AvgContractCoverage_Unavailable verifies that when
// AvgContractCoverage is nil (quality step failed) and MinContractCoverage
// is configured, the result is FAIL with Actual=nil and "(unavailable)" name.
func TestEvaluateThresholds_AvgContractCoverage_Unavailable(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			CRAPload:            intPtr(3),
			AvgContractCoverage: nil, // metric unavailable (step failed)
		},
	}
	cfg := ThresholdConfig{MinContractCoverage: intPtr(50)}
	results, passed := EvaluateThresholds(cfg, payload)
	if passed {
		t.Error("expected passed=false when AvgContractCoverage is nil (unavailable)")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if r.Passed {
		t.Error("expected result.Passed=false")
	}
	if r.Actual != nil {
		t.Errorf("expected Actual=nil (unavailable), got %d", *r.Actual)
	}
	if r.Name != "AvgContractCoverage (unavailable)" {
		t.Errorf("expected Name='AvgContractCoverage (unavailable)', got %q", r.Name)
	}
	if r.Limit != 50 {
		t.Errorf("expected Limit=50, got %d", r.Limit)
	}
}

// TestEvaluateThresholds_CRAPload_ZeroIsValid verifies that CRAPload=0 from
// a successful analysis (intPtr(0)) passes the threshold, confirming that
// zero-from-success is distinguishable from nil-from-failure.
func TestEvaluateThresholds_CRAPload_ZeroIsValid(t *testing.T) {
	payload := &ReportPayload{
		Summary: ReportSummary{CRAPload: intPtr(0)},
	}
	cfg := ThresholdConfig{MaxCrapload: intPtr(5)}
	results, passed := EvaluateThresholds(cfg, payload)
	if !passed {
		t.Error("expected passed=true when CRAPload=0 (valid zero) and limit=5")
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(results))
	}
	r := results[0]
	if !r.Passed {
		t.Error("expected result.Passed=true")
	}
	if r.Name != "CRAPload" {
		t.Errorf("expected Name='CRAPload' (no unavailable suffix), got %q", r.Name)
	}
	if r.Actual == nil {
		t.Fatal("expected Actual=intPtr(0), got nil")
	}
	if *r.Actual != 0 {
		t.Errorf("expected Actual=0, got %d", *r.Actual)
	}
}

// BenchmarkEvaluateThresholds measures the overhead of threshold evaluation.
// EvaluateThresholds is a pure in-memory function with no I/O; its overhead
// must be negligible (well under 1 ms per invocation).
func BenchmarkEvaluateThresholds(b *testing.B) {
	payload := &ReportPayload{
		Summary: ReportSummary{
			CRAPload:            intPtr(8),
			GazeCRAPload:        intPtr(3),
			AvgContractCoverage: intPtr(72),
		},
	}
	cfg := ThresholdConfig{
		MaxCrapload:         intPtr(10),
		MaxGazeCrapload:     intPtr(5),
		MinContractCoverage: intPtr(60),
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EvaluateThresholds(cfg, payload)
	}
}
