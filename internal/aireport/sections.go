package aireport

import (
	"fmt"
	"strings"
)

// Section names one renderable part of a gaze report. The set of
// sections selects which analysis steps run and which report
// sections render (--sections flag).
type Section string

const (
	// SectionCRAP selects the CRAP/GazeCRAP analysis and section.
	SectionCRAP Section = "crap"
	// SectionQuality selects the test quality analysis and section.
	SectionQuality Section = "quality"
	// SectionClassification selects the side-effect classification
	// analysis and section.
	SectionClassification Section = "classification"
	// SectionHealth selects the overall health assessment, which
	// requires the CRAP, quality, and docscan steps.
	SectionHealth Section = "health"
)

// canonicalSections lists every section in canonical render order.
// Render order never depends on the order sections were requested.
var canonicalSections = [...]Section{
	SectionCRAP,
	SectionQuality,
	SectionClassification,
	SectionHealth,
}

// validSectionsHint is the shared tail of every --sections parse
// error, naming the closed set of accepted values.
const validSectionsHint = "valid sections are crap, quality, classification, health"

// SectionSet is an immutable set of report sections. The zero value
// is the empty set; use ParseSections or AllSections to build one.
type SectionSet struct {
	bits uint8
}

// sectionBit returns the bit for s and whether s is a known section.
func sectionBit(s Section) (uint8, bool) {
	for i, c := range canonicalSections {
		if c == s {
			return 1 << uint(i), true
		}
	}
	return 0, false
}

// AllSections returns the set containing every section — the default
// when --sections is not provided.
func AllSections() SectionSet {
	return SectionSet{bits: 1<<uint(len(canonicalSections)) - 1}
}

// ParseSections builds a SectionSet from raw --sections values. Each
// element may itself be a comma-separated list. Duplicates collapse;
// an empty list, an empty element, or an unknown name is an error.
func ParseSections(raw []string) (SectionSet, error) {
	if len(raw) == 0 {
		return SectionSet{}, fmt.Errorf("invalid --sections value \"\": %s", validSectionsHint)
	}
	var set SectionSet
	for _, item := range raw {
		for _, name := range strings.Split(item, ",") {
			name = strings.TrimSpace(name)
			bit, ok := sectionBit(Section(name))
			if !ok {
				return SectionSet{}, fmt.Errorf("invalid --sections value %q: %s", name, validSectionsHint)
			}
			set.bits |= bit
		}
	}
	return set, nil
}

// Has reports whether s contains section sec.
func (s SectionSet) Has(sec Section) bool {
	bit, ok := sectionBit(sec)
	return ok && s.bits&bit != 0
}

// List returns the sections in s in canonical render order.
func (s SectionSet) List() []Section {
	bits := s.bits
	out := make([]Section, 0, len(canonicalSections))

	for i, c := range canonicalSections {
		if bits&(1<<uint(i)) != 0 {
			out = append(out, c)
		}
	}

	return out
}

// String returns the canonical comma-joined form of s (for example
// "crap,quality"), as used in error messages.
func (s SectionSet) String() string {
	bits := s.bits
	parts := make([]string, 0, len(canonicalSections))

	for i, c := range canonicalSections {
		if bits&(1<<uint(i)) != 0 {
			parts = append(parts, string(c))
		}
	}

	return strings.Join(parts, ",")
}

// StepSet records which analysis pipeline steps must run.
type StepSet struct {
	CRAP     bool
	Quality  bool
	Classify bool
	Docscan  bool
}

// Steps maps the selected sections to the analysis steps they need:
// crap → CRAP, quality → Quality, classification → Classify, and
// health → CRAP + Quality + Docscan.
func (s SectionSet) Steps() StepSet {
	bits := s.bits
	crap, _ := sectionBit(SectionCRAP)
	quality, _ := sectionBit(SectionQuality)
	classification, _ := sectionBit(SectionClassification)
	health, _ := sectionBit(SectionHealth)

	return StepSet{
		CRAP:     bits&crap != 0 || bits&health != 0,
		Quality:  bits&quality != 0 || bits&health != 0,
		Classify: bits&classification != 0,
		Docscan:  bits&health != 0,
	}
}

// CheckThresholds rejects threshold flags whose analysis step the
// selected sections do not run: --max-crapload and
// --max-gaze-crapload need the CRAP step, --min-contract-coverage
// needs the quality step.
func CheckThresholds(s SectionSet, cfg ThresholdConfig) error {
	steps := s.Steps()
	checks := []struct {
		set  bool
		runs bool
		flag string
		step string
	}{
		{cfg.MaxCrapload != nil, steps.CRAP, "max-crapload", "crap"},
		{cfg.MaxGazeCrapload != nil, steps.CRAP, "max-gaze-crapload", "crap"},
		{cfg.MinContractCoverage != nil, steps.Quality, "min-contract-coverage", "quality"},
	}
	for _, c := range checks {
		if c.set && !c.runs {
			return fmt.Errorf("--%s requires the %s analysis, which --sections=%s does not run", c.flag, c.step, s)
		}
	}
	return nil
}
