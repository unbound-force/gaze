// Package taxonomy defines the side effect type system, core data
// structures, and stable ID generation for Gaze analysis results.
package taxonomy

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// SideEffectType enumerates all observable side effect categories.
type SideEffectType string

// P0 — Must Detect (zero false negatives, zero false positives).
const (
	ReturnValue        SideEffectType = "ReturnValue"
	ErrorReturn        SideEffectType = "ErrorReturn"
	SentinelError      SideEffectType = "SentinelError"
	ReceiverMutation   SideEffectType = "ReceiverMutation"
	PointerArgMutation SideEffectType = "PointerArgMutation"
)

// P1 — High Value.
const (
	SliceMutation          SideEffectType = "SliceMutation"
	MapMutation            SideEffectType = "MapMutation"
	GlobalMutation         SideEffectType = "GlobalMutation"
	WriterOutput           SideEffectType = "WriterOutput"
	HTTPResponseWrite      SideEffectType = "HTTPResponseWrite"
	ChannelSend            SideEffectType = "ChannelSend"
	ChannelClose           SideEffectType = "ChannelClose"
	DeferredReturnMutation SideEffectType = "DeferredReturnMutation"
)

// P2 — Important.
const (
	FileSystemWrite     SideEffectType = "FileSystemWrite"
	FileSystemDelete    SideEffectType = "FileSystemDelete"
	FileSystemMeta      SideEffectType = "FileSystemMeta"
	DatabaseWrite       SideEffectType = "DatabaseWrite"
	DatabaseTransaction SideEffectType = "DatabaseTransaction"
	GoroutineSpawn      SideEffectType = "GoroutineSpawn"
	Panic               SideEffectType = "Panic"
	CallbackInvocation  SideEffectType = "CallbackInvocation"
	LogWrite            SideEffectType = "LogWrite"
	ContextCancellation SideEffectType = "ContextCancellation"
)

// P3 — Nice to Have.
const (
	StdoutWrite     SideEffectType = "StdoutWrite"
	StderrWrite     SideEffectType = "StderrWrite"
	EnvVarMutation  SideEffectType = "EnvVarMutation"
	MutexOp         SideEffectType = "MutexOp"
	WaitGroupOp     SideEffectType = "WaitGroupOp"
	AtomicOp        SideEffectType = "AtomicOp"
	TimeDependency  SideEffectType = "TimeDependency"
	ProcessExit     SideEffectType = "ProcessExit"
	RecoverBehavior SideEffectType = "RecoverBehavior"
)

// P4 — Exotic.
const (
	ReflectionMutation     SideEffectType = "ReflectionMutation"
	UnsafeMutation         SideEffectType = "UnsafeMutation"
	CgoCall                SideEffectType = "CgoCall"
	FinalizerRegistration  SideEffectType = "FinalizerRegistration"
	SyncPoolOp             SideEffectType = "SyncPoolOp"
	ClosureCaptureMutation SideEffectType = "ClosureCaptureMutation"
)

// Tier represents the priority tier for a side effect type.
type Tier string

// Priority tier constants for side effect classification.
const (
	TierP0 Tier = "P0"
	TierP1 Tier = "P1"
	TierP2 Tier = "P2"
	TierP3 Tier = "P3"
	TierP4 Tier = "P4"
)

// ClassificationLabel represents the contractual classification of
// a side effect.
type ClassificationLabel string

// Classification label constants.
const (
	Contractual ClassificationLabel = "contractual"
	Incidental  ClassificationLabel = "incidental"
	Ambiguous   ClassificationLabel = "ambiguous"
)

// Signal represents a single piece of evidence contributing to a
// classification confidence score.
type Signal struct {
	// Source identifies the signal type (e.g., "interface",
	// "caller", "naming", "godoc", "readme", "architecture_doc").
	Source string `json:"source"`

	// Weight is the numeric contribution to the confidence score.
	// Can be negative for contradiction penalties.
	Weight int `json:"weight"`

	// SourceFile is the file path that provided this signal.
	// Omitted from JSON when empty (non-verbose mode).
	SourceFile string `json:"source_file,omitempty"`

	// Excerpt is the relevant text from the source.
	// Omitted from JSON when empty (non-verbose mode).
	Excerpt string `json:"excerpt,omitempty"`

	// Reasoning explains why this signal was applied.
	// Omitted from JSON when empty (non-verbose mode).
	Reasoning string `json:"reasoning,omitempty"`
}

// Classification represents the contractual classification of a
// single side effect, including the confidence score and the
// signals that contributed to it.
type Classification struct {
	// Label is the classification result.
	Label ClassificationLabel `json:"label"`

	// Confidence is the numeric confidence score (0-100).
	Confidence int `json:"confidence"`

	// Signals is the list of evidence that contributed to the
	// confidence score.
	Signals []Signal `json:"signals"`

	// Reasoning is a human-readable summary of the classification.
	Reasoning string `json:"reasoning,omitempty"`
}

// SideEffect represents a single detected observable change in a
// function.
type SideEffect struct {
	// ID is a stable identifier for diffing across runs.
	// Generated from sha256(pkg+func+type+location).
	ID string `json:"id"`

	// Type is the category of side effect from the taxonomy.
	Type SideEffectType `json:"type"`

	// Tier is the priority tier (P0-P4).
	Tier Tier `json:"tier"`

	// Location is the source position (file:line:col).
	Location string `json:"location"`

	// Description is a human-readable explanation.
	Description string `json:"description"`

	// Target is the affected entity (field name, variable name,
	// channel name, return type, etc.).
	Target string `json:"target"`

	// Classification is the contractual classification of this
	// side effect. Nil when classification has not been performed.
	Classification *Classification `json:"classification,omitempty"`
}

// FunctionTarget identifies the function under analysis.
type FunctionTarget struct {
	// Package is the full import path.
	Package string `json:"package"`

	// Function is the function or method name.
	Function string `json:"function"`

	// Receiver is the receiver type for methods (e.g., "*Store"),
	// empty for package-level functions.
	Receiver string `json:"receiver,omitempty"`

	// Signature is the full function signature string.
	Signature string `json:"signature"`

	// Location is the source position of the function declaration.
	Location string `json:"location"`
}

// QualifiedName returns the fully qualified function name including
// receiver if present. E.g., "(*Store).Save" or "ParseConfig".
func (ft FunctionTarget) QualifiedName() string {
	if ft.Receiver != "" {
		return fmt.Sprintf("(%s).%s", ft.Receiver, ft.Function)
	}
	return ft.Function
}

// Metadata holds analysis run metadata.
type Metadata struct {
	GazeVersion string        `json:"gaze_version"`
	GoVersion   string        `json:"go_version"`
	Timestamp   time.Time     `json:"-"`
	Duration    time.Duration `json:"-"`
	Warnings    []string      `json:"warnings"`
}

// MarshalJSON customizes JSON encoding to use duration_ms and
// ISO 8601 timestamp.
func (m Metadata) MarshalJSON() ([]byte, error) {
	type Alias Metadata
	ts := ""
	if !m.Timestamp.IsZero() {
		ts = m.Timestamp.UTC().Format(time.RFC3339)
	}
	return json.Marshal(&struct {
		Alias
		DurationMS int64  `json:"duration_ms"`
		Timestamp  string `json:"timestamp,omitempty"`
	}{
		Alias:      Alias(m),
		DurationMS: m.Duration.Milliseconds(),
		Timestamp:  ts,
	})
}

// AnalysisResult is the complete output for one function.
type AnalysisResult struct {
	// Target identifies the analyzed function.
	Target FunctionTarget `json:"target"`

	// SideEffects is the list of detected side effects.
	SideEffects []SideEffect `json:"side_effects"`

	// Metadata contains run information.
	Metadata Metadata `json:"metadata"`
}

// AssertionType enumerates the kinds of test assertions Gaze can detect.
type AssertionType string

// Assertion type constants.
const (
	AssertionEquality   AssertionType = "equality"
	AssertionErrorCheck AssertionType = "error_check"
	AssertionNilCheck   AssertionType = "nil_check"
	AssertionDiffCheck  AssertionType = "diff_check"
	AssertionCustom     AssertionType = "custom"
)

// UnmappedReasonType enumerates the reasons why an assertion could not
// be linked to a detected side effect.
type UnmappedReasonType string

// Unmapped reason constants.
const (
	// UnmappedReasonHelperParam indicates the assertion is inside a
	// helper function body (depth > 0). The helper's parameter objects
	// cannot be traced back to the test's variable assignments without
	// param-substitution support.
	UnmappedReasonHelperParam UnmappedReasonType = "helper_param"

	// UnmappedReasonInlineCall indicates the target function was called
	// inline without assigning its return value (e.g., "if f() != x").
	// Gaze only traces return values that appear on the LHS of an
	// assignment statement.
	UnmappedReasonInlineCall UnmappedReasonType = "inline_call"

	// UnmappedReasonNoEffectMatch indicates the assertion is in the
	// test body and return values were traced, but no identifier in
	// the assertion expression matched a traced side effect object.
	// Typically a cross-target assertion or an unsupported pattern.
	UnmappedReasonNoEffectMatch UnmappedReasonType = "no_effect_match"
)

// AssertionMapping links a test assertion to the side effect it verifies.
type AssertionMapping struct {
	// AssertionLocation is the source position (file:line) of the assertion.
	AssertionLocation string `json:"assertion_location"`

	// AssertionType is the kind of assertion (equality, error_check, etc.).
	AssertionType AssertionType `json:"assertion_type"`

	// SideEffectID is the stable ID of the mapped side effect.
	SideEffectID string `json:"side_effect_id"`

	// Confidence is the mapping confidence (0-100).
	Confidence int `json:"confidence"`

	// UnmappedReason explains why this assertion could not be linked to
	// a side effect. Only populated for unmapped assertions (Confidence 0,
	// SideEffectID empty). Omitted from JSON for mapped assertions.
	UnmappedReason UnmappedReasonType `json:"unmapped_reason,omitempty"`
}

// ContractCoverage is the primary test quality metric: the ratio of
// contractual side effects that the test asserts on.
type ContractCoverage struct {
	// Percentage is the coverage ratio (0-100).
	Percentage float64 `json:"percentage"`

	// CoveredCount is the number of contractual effects asserted on.
	CoveredCount int `json:"covered_count"`

	// TotalContractual is the total number of contractual effects.
	TotalContractual int `json:"total_contractual"`

	// Gaps lists contractual effects that are NOT asserted on.
	Gaps []SideEffect `json:"gaps"`

	// GapHints contains a Go code snippet for each gap, suggesting how
	// to write the missing assertion. Parallel to Gaps: len(GapHints)
	// always equals len(Gaps). Omitted from JSON when there are no gaps.
	GapHints []string `json:"gap_hints,omitempty"`

	// DiscardedReturns lists contractual return/error effects whose
	// values were explicitly discarded (e.g., _ = target()),
	// making them definitively unasserted.
	DiscardedReturns []SideEffect `json:"discarded_returns"`

	// DiscardedReturnHints contains a Go code snippet for each
	// discarded return, suggesting how to assert on it. Parallel to
	// DiscardedReturns: len(DiscardedReturnHints) == len(DiscardedReturns).
	// Omitted from JSON when there are no discarded returns.
	DiscardedReturnHints []string `json:"discarded_return_hints,omitempty"`
}

// OverSpecificationScore measures how many incidental side effects
// the test asserts on, indicating refactoring fragility.
type OverSpecificationScore struct {
	// Count is the number of incidental side effects asserted on.
	Count int `json:"count"`

	// Ratio is incidental assertions / total assertions (0.0-1.0).
	Ratio float64 `json:"ratio"`

	// IncidentalAssertions lists mappings to incidental effects.
	IncidentalAssertions []AssertionMapping `json:"incidental_assertions"`

	// Suggestions provides actionable advice per incidental assertion.
	Suggestions []string `json:"suggestions"`
}

// QualityReport is the complete test quality output for one
// test-target pair.
type QualityReport struct {
	// TestFunction is the name of the test function.
	TestFunction string `json:"test_function"`

	// TestLocation is the source position of the test function.
	TestLocation string `json:"test_location"`

	// TargetFunction identifies the function under test.
	TargetFunction FunctionTarget `json:"target_function"`

	// ContractCoverage is the primary coverage metric.
	ContractCoverage ContractCoverage `json:"contract_coverage"`

	// OverSpecification is the fragility metric.
	OverSpecification OverSpecificationScore `json:"over_specification"`

	// AmbiguousEffects lists side effects excluded from metrics
	// due to ambiguous classification.
	AmbiguousEffects []SideEffect `json:"ambiguous_effects"`

	// UnmappedAssertions lists assertions that could not be linked
	// to any detected side effect.
	UnmappedAssertions []AssertionMapping `json:"unmapped_assertions"`

	// AssertionDetectionConfidence indicates the fraction of test
	// assertions that were successfully pattern-matched (0-100).
	AssertionDetectionConfidence int `json:"assertion_detection_confidence"`

	// Metadata contains run information.
	Metadata Metadata `json:"metadata"`
}

// PackageSummary holds aggregate quality metrics for a package.
type PackageSummary struct {
	// TotalTests is the number of test functions analyzed.
	TotalTests int `json:"total_tests"`

	// AverageContractCoverage is the mean coverage across tests.
	AverageContractCoverage float64 `json:"average_contract_coverage"`

	// TotalOverSpecifications is the sum of incidental assertion
	// counts across all tests.
	TotalOverSpecifications int `json:"total_over_specifications"`

	// WorstCoverageTests lists the bottom 5 tests by coverage.
	WorstCoverageTests []QualityReport `json:"worst_coverage_tests"`

	// AssertionDetectionConfidence is the aggregate detection
	// confidence across all tests.
	AssertionDetectionConfidence int `json:"assertion_detection_confidence"`
}

// GenerateID produces a stable, deterministic ID for a side effect
// based on its context. The ID is a sha256 hash truncated to 8 hex
// characters, prefixed with "se-".
func GenerateID(pkg, function, effectType, location string) string {
	input := fmt.Sprintf("%s:%s:%s:%s", pkg, function, effectType, location)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("se-%x", hash[:4])
}
