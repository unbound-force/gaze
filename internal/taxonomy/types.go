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
	Duration    time.Duration `json:"-"`
	Warnings    []string      `json:"warnings"`
}

// MarshalJSON customizes JSON encoding to use duration_ms.
func (m Metadata) MarshalJSON() ([]byte, error) {
	type Alias Metadata
	return json.Marshal(&struct {
		Alias
		DurationMS int64 `json:"duration_ms"`
	}{
		Alias:      Alias(m),
		DurationMS: m.Duration.Milliseconds(),
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

// GenerateID produces a stable, deterministic ID for a side effect
// based on its context. The ID is a sha256 hash truncated to 8 hex
// characters, prefixed with "se-".
func GenerateID(pkg, function, effectType, location string) string {
	input := fmt.Sprintf("%s:%s:%s:%s", pkg, function, effectType, location)
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("se-%x", hash[:4])
}
