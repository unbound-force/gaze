package aireport

// intPtr returns a pointer to v. Used to construct *int values for
// ThresholdResult.Actual and test helpers.
func intPtr(v int) *int { return &v }

// EvaluateThresholds checks ThresholdConfig against ReportPayload summary
// data. It returns a slice of ThresholdResult values (one per non-nil
// threshold) and a boolean indicating whether all thresholds passed.
//
// The typed values in payload.Summary are used directly — no JSON
// unmarshalling is required. Summary fields are populated by the analysis
// pipeline step functions before EvaluateThresholds is called.
//
// Thresholds that are nil in cfg are skipped entirely. When a threshold
// is set but the corresponding metric is unavailable (nil), the result
// is FAIL with Actual = nil — CI gates must not silently pass when data
// is missing.
func EvaluateThresholds(cfg ThresholdConfig, payload *ReportPayload) ([]ThresholdResult, bool) {
	if cfg.MaxCrapload == nil && cfg.MaxGazeCrapload == nil && cfg.MinContractCoverage == nil {
		return nil, true
	}

	var results []ThresholdResult
	allPassed := true

	var summary ReportSummary
	if payload != nil {
		summary = payload.Summary
	}

	if cfg.MaxCrapload != nil {
		limit := *cfg.MaxCrapload
		if summary.CRAPload == nil {
			// Metric unavailable — threshold cannot be evaluated, so it fails.
			// CI gates must not silently pass when data is missing.
			allPassed = false
			results = append(results, ThresholdResult{
				Name:   "CRAPload (unavailable)",
				Actual: nil,
				Limit:  limit,
				Passed: false,
			})
		} else {
			passed := *summary.CRAPload <= limit
			if !passed {
				allPassed = false
			}
			results = append(results, ThresholdResult{
				Name:   "CRAPload",
				Actual: summary.CRAPload,
				Limit:  limit,
				Passed: passed,
			})
		}
	}

	if cfg.MaxGazeCrapload != nil {
		limit := *cfg.MaxGazeCrapload
		if summary.GazeCRAPload == nil {
			// Metric unavailable — threshold cannot be evaluated, so it fails.
			// CI gates must not silently pass when data is missing.
			allPassed = false
			results = append(results, ThresholdResult{
				Name:   "GazeCRAPload (unavailable)",
				Actual: nil,
				Limit:  limit,
				Passed: false,
			})
		} else {
			passed := *summary.GazeCRAPload <= limit
			if !passed {
				allPassed = false
			}
			results = append(results, ThresholdResult{
				Name:   "GazeCRAPload",
				Actual: summary.GazeCRAPload,
				Limit:  limit,
				Passed: passed,
			})
		}
	}

	if cfg.MinContractCoverage != nil {
		limit := *cfg.MinContractCoverage
		if summary.AvgContractCoverage == nil {
			// Metric unavailable — threshold cannot be evaluated, so it fails.
			// CI gates must not silently pass when data is missing.
			allPassed = false
			results = append(results, ThresholdResult{
				Name:   "AvgContractCoverage (unavailable)",
				Actual: nil,
				Limit:  limit,
				Passed: false,
			})
		} else {
			passed := *summary.AvgContractCoverage >= limit
			if !passed {
				allPassed = false
			}
			results = append(results, ThresholdResult{
				Name:   "AvgContractCoverage",
				Actual: summary.AvgContractCoverage,
				Limit:  limit,
				Passed: passed,
			})
		}
	}

	return results, allPassed
}
