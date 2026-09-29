package classify_test

import (
	"testing"

	"github.com/unbound-force/gaze/v2/internal/classify"
	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

func TestCountLabels_Mixed(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			SideEffects: []taxonomy.SideEffect{
				{Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
				{Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
				{Classification: &taxonomy.Classification{Label: taxonomy.Ambiguous}},
			},
		},
		{
			SideEffects: []taxonomy.SideEffect{
				{Classification: &taxonomy.Classification{Label: taxonomy.Contractual}},
				{Classification: &taxonomy.Classification{Label: taxonomy.Incidental}},
			},
		},
	}

	contractual, ambiguous, incidental := classify.CountLabels(results)

	if contractual != 3 {
		t.Errorf("expected 3 contractual, got %d", contractual)
	}
	if ambiguous != 1 {
		t.Errorf("expected 1 ambiguous, got %d", ambiguous)
	}
	if incidental != 1 {
		t.Errorf("expected 1 incidental, got %d", incidental)
	}
}

func TestCountLabels_NoClassification(t *testing.T) {
	results := []taxonomy.AnalysisResult{
		{
			SideEffects: []taxonomy.SideEffect{
				{Classification: nil},
				{Classification: nil},
			},
		},
	}

	contractual, ambiguous, incidental := classify.CountLabels(results)

	if contractual != 0 {
		t.Errorf("expected 0 contractual, got %d", contractual)
	}
	if ambiguous != 0 {
		t.Errorf("expected 0 ambiguous, got %d", ambiguous)
	}
	if incidental != 0 {
		t.Errorf("expected 0 incidental, got %d", incidental)
	}
}

func TestCountLabels_Empty(t *testing.T) {
	contractual, ambiguous, incidental := classify.CountLabels(nil)

	if contractual != 0 || ambiguous != 0 || incidental != 0 {
		t.Errorf("expected all zeros for nil input, got %d/%d/%d", contractual, ambiguous, incidental)
	}
}
