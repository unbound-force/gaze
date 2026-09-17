package adapter

import (
	"testing"

	"github.com/unbound-force/gaze/internal/taxonomy"
)

// ---------------------------------------------------------------------------
// findSideEffectID tests (table-driven)
// ---------------------------------------------------------------------------

func TestFindSideEffectID(t *testing.T) {
	tests := []struct {
		name    string
		effects []taxonomy.SideEffect
		typ     string
		wantID  string
	}{
		{
			name: "matching type found",
			effects: []taxonomy.SideEffect{
				{ID: "se-001", Type: "ReturnValue"},
				{ID: "se-002", Type: "ErrorReturn"},
			},
			typ:    "ErrorReturn",
			wantID: "se-002",
		},
		{
			name: "no match returns empty string",
			effects: []taxonomy.SideEffect{
				{ID: "se-001", Type: "ReturnValue"},
			},
			typ:    "MapMutation",
			wantID: "",
		},
		{
			name:    "empty effects slice",
			effects: nil,
			typ:     "ReturnValue",
			wantID:  "",
		},
		{
			name: "multiple effects with same type returns first match",
			effects: []taxonomy.SideEffect{
				{ID: "se-first", Type: "ReturnValue"},
				{ID: "se-second", Type: "ReturnValue"},
			},
			typ:    "ReturnValue",
			wantID: "se-first",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := findSideEffectID(tt.effects, tt.typ)
			if got != tt.wantID {
				t.Errorf("findSideEffectID() = %q, want %q", got, tt.wantID)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// confidenceRange tests (table-driven)
// ---------------------------------------------------------------------------

func TestConfidenceRange(t *testing.T) {
	tests := []struct {
		name    string
		effects []taxonomy.SideEffect
		wantMin int
		wantMax int
		wantOK  bool
	}{
		{
			name:    "empty slice",
			effects: nil,
			wantMin: 0, wantMax: 0, wantOK: false,
		},
		{
			name: "all nil classification",
			effects: []taxonomy.SideEffect{
				{Classification: nil},
				{Classification: nil},
			},
			wantMin: 0, wantMax: 0, wantOK: false,
		},
		{
			name: "all classified",
			effects: []taxonomy.SideEffect{
				{Classification: &taxonomy.Classification{Confidence: 78}},
				{Classification: &taxonomy.Classification{Confidence: 85}},
				{Classification: &taxonomy.Classification{Confidence: 79}},
			},
			wantMin: 78, wantMax: 85, wantOK: true,
		},
		{
			name: "mixed nil and classified",
			effects: []taxonomy.SideEffect{
				{Classification: nil},
				{Classification: &taxonomy.Classification{Confidence: 60}},
				{Classification: nil},
				{Classification: &taxonomy.Classification{Confidence: 72}},
			},
			wantMin: 60, wantMax: 72, wantOK: true,
		},
		{
			name: "single effect",
			effects: []taxonomy.SideEffect{
				{Classification: &taxonomy.Classification{Confidence: 50}},
			},
			wantMin: 50, wantMax: 50, wantOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minC, maxC, found := confidenceRange(tt.effects)
			if found != tt.wantOK {
				t.Errorf("found = %v, want %v", found, tt.wantOK)
			}
			if minC != tt.wantMin {
				t.Errorf("minConf = %d, want %d", minC, tt.wantMin)
			}
			if maxC != tt.wantMax {
				t.Errorf("maxConf = %d, want %d", maxC, tt.wantMax)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// deriveCoverageReason tests (table-driven)
// ---------------------------------------------------------------------------

func TestDeriveCoverageReason(t *testing.T) {
	tests := []struct {
		name       string
		effects    []taxonomy.SideEffect
		cc         taxonomy.ContractCoverage
		wantReason string
		wantMin    int
		wantMax    int
	}{
		{
			name:       "no effects",
			effects:    nil,
			cc:         taxonomy.ContractCoverage{},
			wantReason: "no_effects_detected",
			wantMin:    0, wantMax: 0,
		},
		{
			name: "all nil classification",
			effects: []taxonomy.SideEffect{
				{Classification: nil},
				{Classification: nil},
			},
			cc:         taxonomy.ContractCoverage{TotalContractual: 0},
			wantReason: "all_effects_unclassified",
			wantMin:    0, wantMax: 0,
		},
		{
			name: "all ambiguous (classified but below threshold)",
			effects: []taxonomy.SideEffect{
				{Classification: &taxonomy.Classification{Confidence: 78}},
				{Classification: &taxonomy.Classification{Confidence: 79}},
			},
			cc:         taxonomy.ContractCoverage{TotalContractual: 0},
			wantReason: "all_effects_ambiguous",
			wantMin:    78, wantMax: 79,
		},
		{
			name: "with contractual effects",
			effects: []taxonomy.SideEffect{
				{Classification: &taxonomy.Classification{Confidence: 90}},
			},
			cc:         taxonomy.ContractCoverage{TotalContractual: 1},
			wantReason: "",
			wantMin:    0, wantMax: 0,
		},
		{
			name: "contractual > 0 with all nil classification",
			effects: []taxonomy.SideEffect{
				{Classification: nil},
				{Classification: nil},
			},
			cc:         taxonomy.ContractCoverage{TotalContractual: 1},
			wantReason: "",
			wantMin:    0, wantMax: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reason, minC, maxC := deriveCoverageReason(tt.effects, tt.cc)
			if reason != tt.wantReason {
				t.Errorf("reason = %q, want %q", reason, tt.wantReason)
			}
			if minC != tt.wantMin {
				t.Errorf("minConf = %d, want %d", minC, tt.wantMin)
			}
			if maxC != tt.wantMax {
				t.Errorf("maxConf = %d, want %d", maxC, tt.wantMax)
			}
		})
	}
}
