package aireport

import (
	"reflect"
	"testing"
)

func intPtr040(v int) *int { return &v }

func TestParseSections_Valid(t *testing.T) {
	tests := []struct {
		name string
		raw  []string
		want []Section
	}{
		{"single", []string{"crap"}, []Section{SectionCRAP}},
		{"multi elements", []string{"quality", "crap"}, []Section{SectionCRAP, SectionQuality}},
		{"comma list", []string{"health,classification"}, []Section{SectionClassification, SectionHealth}},
		{"whitespace trimmed", []string{" crap , quality "}, []Section{SectionCRAP, SectionQuality}},
		{"duplicates collapse", []string{"crap,crap", "crap"}, []Section{SectionCRAP}},
		{"reordered to canonical", []string{"health,classification,quality,crap"}, []Section{SectionCRAP, SectionQuality, SectionClassification, SectionHealth}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSections(tt.raw)
			if err != nil {
				t.Fatalf("ParseSections(%q) error: %v", tt.raw, err)
			}
			if !reflect.DeepEqual(got.List(), tt.want) {
				t.Errorf("List() = %v, want %v", got.List(), tt.want)
			}
		})
	}
}

func TestParseSections_Errors(t *testing.T) {
	tests := []struct {
		name    string
		raw     []string
		wantErr string
	}{
		{"empty list", nil, `invalid --sections value "": valid sections are crap, quality, classification, health`},
		{"empty string", []string{""}, `invalid --sections value "": valid sections are crap, quality, classification, health`},
		{"empty element", []string{"crap,,quality"}, `invalid --sections value "": valid sections are crap, quality, classification, health`},
		{"unknown", []string{"crap,bogus"}, `invalid --sections value "bogus": valid sections are crap, quality, classification, health`},
		{"case sensitive", []string{"CRAP"}, `invalid --sections value "CRAP": valid sections are crap, quality, classification, health`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseSections(tt.raw)
			if err == nil {
				t.Fatalf("ParseSections(%q) = %v, want error", tt.raw, got.List())
			}
			if err.Error() != tt.wantErr {
				t.Errorf("error = %q, want %q", err.Error(), tt.wantErr)
			}
			if len(got.List()) != 0 {
				t.Errorf("error result should be empty set, got %v", got.List())
			}
		})
	}
}

func TestAllSections(t *testing.T) {
	all := AllSections()
	want := []Section{SectionCRAP, SectionQuality, SectionClassification, SectionHealth}
	if !reflect.DeepEqual(all.List(), want) {
		t.Errorf("AllSections().List() = %v, want %v", all.List(), want)
	}
	if all.String() != "crap,quality,classification,health" {
		t.Errorf("AllSections().String() = %q", all.String())
	}
	if all.Has(Section("bogus")) {
		t.Error("Has(bogus) = true, want false")
	}
}

func TestSectionSet_ZeroValue(t *testing.T) {
	var s SectionSet
	if len(s.List()) != 0 || s.String() != "" {
		t.Errorf("zero SectionSet List=%v String=%q, want empty", s.List(), s.String())
	}
	if s.Has(SectionCRAP) {
		t.Error("zero SectionSet Has(crap) = true")
	}
	if s.Steps() != (StepSet{}) {
		t.Errorf("zero SectionSet Steps = %+v, want none", s.Steps())
	}
}

func TestSectionSet_Steps(t *testing.T) {
	tests := []struct {
		raw  string
		want StepSet
	}{
		{"crap", StepSet{CRAP: true}},
		{"quality", StepSet{Quality: true}},
		{"classification", StepSet{Classify: true}},
		{"health", StepSet{CRAP: true, Quality: true, Docscan: true}},
		{"crap,quality,classification,health", StepSet{CRAP: true, Quality: true, Classify: true, Docscan: true}},
	}
	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			s, err := ParseSections([]string{tt.raw})
			if err != nil {
				t.Fatalf("ParseSections: %v", err)
			}
			if got := s.Steps(); got != tt.want {
				t.Errorf("Steps() = %+v, want %+v", got, tt.want)
			}
		})
	}
	if got := AllSections().Steps(); got != (StepSet{CRAP: true, Quality: true, Classify: true, Docscan: true}) {
		t.Errorf("AllSections().Steps() = %+v", got)
	}
}

func TestSectionSet_String(t *testing.T) {
	s, err := ParseSections([]string{"quality,crap"})
	if err != nil {
		t.Fatalf("ParseSections: %v", err)
	}
	if s.String() != "crap,quality" {
		t.Errorf("String() = %q, want %q", s.String(), "crap,quality")
	}
}

func TestCheckThresholds(t *testing.T) {
	tests := []struct {
		name     string
		sections string
		cfg      ThresholdConfig
		wantErr  string
	}{
		{"no thresholds", "classification", ThresholdConfig{}, ""},
		{"crapload with crap", "crap", ThresholdConfig{MaxCrapload: intPtr040(0)}, ""},
		{"crapload with health", "health", ThresholdConfig{MaxCrapload: intPtr040(5)}, ""},
		{"crapload without crap", "quality", ThresholdConfig{MaxCrapload: intPtr040(5)},
			"--max-crapload requires the crap analysis, which --sections=quality does not run"},
		{"gaze crapload with crap", "crap", ThresholdConfig{MaxGazeCrapload: intPtr040(1)}, ""},
		{"gaze crapload without crap", "quality,classification", ThresholdConfig{MaxGazeCrapload: intPtr040(1)},
			"--max-gaze-crapload requires the crap analysis, which --sections=quality,classification does not run"},
		{"coverage with quality", "quality", ThresholdConfig{MinContractCoverage: intPtr040(8)}, ""},
		{"coverage with health", "health", ThresholdConfig{MinContractCoverage: intPtr040(8)}, ""},
		{"coverage without quality", "crap", ThresholdConfig{MinContractCoverage: intPtr040(8)},
			"--min-contract-coverage requires the quality analysis, which --sections=crap does not run"},
		{"all with all", "crap,quality,classification,health", ThresholdConfig{
			MaxCrapload: intPtr040(38), MaxGazeCrapload: intPtr040(6), MinContractCoverage: intPtr040(8)}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s, err := ParseSections([]string{tt.sections})
			if err != nil {
				t.Fatalf("ParseSections: %v", err)
			}
			err = CheckThresholds(s, tt.cfg)
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("CheckThresholds error = %v, want nil", err)
				}
				return
			}
			if err == nil || err.Error() != tt.wantErr {
				t.Errorf("CheckThresholds error = %v, want %q", err, tt.wantErr)
			}
		})
	}
}
