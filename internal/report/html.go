package report

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"strings"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

//go:embed analyze.html.tmpl
var analyzeTemplateFS embed.FS

// htmlReportData is the top-level view model passed to the HTML template.
// All fields are pre-computed, deterministic, and safe for contextual
// escaping by html/template.
type htmlReportData struct {
	Version           string
	Functions         []htmlFunctionData
	TotalEffects      int
	HasClassification bool
	HasDetail         bool
}

// htmlFunctionData is the per-function view model.
type htmlFunctionData struct {
	Name        string
	Location    string
	Signature   string
	EffectCount int
	Effects     []htmlEffectData
}

// htmlEffectData is the per-side-effect view model. All string fields
// are plain text; html/template applies contextual escaping during
// execution.
type htmlEffectData struct {
	Tier           string
	TierClass      string
	Type           string
	Description    string
	Location       string
	Classification string
	Detail         string
}

// WriteHTML writes analysis results as a self-contained HTML document
// to w. The version string is embedded in the document metadata; if
// empty, it defaults to "dev".
//
// The function converts taxonomy values into a deterministic
// presentation view model before template execution. SideEffect.Detail
// metadata is JSON-serialized with sorted map keys for stable output.
// All source-derived values pass through html/template contextual
// escaping; no values are cast to trusted template content types.
//
// Errors from template parsing or execution are returned with
// operation-specific context wrapping.
func WriteHTML(w io.Writer, results []taxonomy.AnalysisResult, version string) error {
	if version == "" {
		version = "dev"
	}

	data, err := buildHTMLData(results, version)
	if err != nil {
		return fmt.Errorf("preparing HTML report data: %w", err)
	}

	tmpl, err := template.ParseFS(analyzeTemplateFS, "analyze.html.tmpl")
	if err != nil {
		return fmt.Errorf("parsing HTML report template: %w", err)
	}

	if err := tmpl.Execute(w, data); err != nil {
		return fmt.Errorf("executing HTML report template: %w", err)
	}

	return nil
}

// buildHTMLData converts analysis results into the deterministic
// presentation view model consumed by the HTML template.
func buildHTMLData(results []taxonomy.AnalysisResult, version string) (*htmlReportData, error) {
	data := &htmlReportData{
		Version: version,
	}

	totalEffects := 0
	hasClassification := false
	hasDetail := false

	for _, r := range results {
		fn := htmlFunctionData{
			Name:        r.Target.QualifiedName(),
			Location:    r.Target.Location,
			Signature:   r.Target.Signature,
			EffectCount: len(r.SideEffects),
		}

		for _, se := range r.SideEffects {
			effect := htmlEffectData{
				Tier:        string(se.Tier),
				TierClass:   tierCSSClass(se.Tier),
				Type:        string(se.Type),
				Description: se.Description,
				Location:    se.Location,
			}

			if se.Classification != nil {
				hasClassification = true
				effect.Classification = formatClassification(se.Classification)
			}

			if len(se.Detail) > 0 {
				hasDetail = true
				serialized, err := serializeDetail(se.Detail)
				if err != nil {
					return nil, fmt.Errorf("serializing detail for effect %s: %w", se.ID, err)
				}
				effect.Detail = serialized
			}

			fn.Effects = append(fn.Effects, effect)
		}

		totalEffects += len(r.SideEffects)
		data.Functions = append(data.Functions, fn)
	}

	data.TotalEffects = totalEffects
	data.HasClassification = hasClassification
	data.HasDetail = hasDetail

	return data, nil
}

// tierCSSClass returns the lowercase CSS class suffix for a tier
// value (e.g., TierP0 -> "p0"). Unknown values are lowercased
// unchanged and produce a corresponding "tier-<value>" class.
func tierCSSClass(tier taxonomy.Tier) string {
	return strings.ToLower(string(tier))
}

// formatClassification produces a human-readable string from a
// Classification value (e.g., "contractual (85%)").
func formatClassification(c *taxonomy.Classification) string {
	return fmt.Sprintf("%s (%d%%)", c.Label, c.Confidence)
}

// serializeDetail deterministically JSON-encodes a Detail map.
// Go's encoding/json sorts string map keys, ensuring stable output
// across calls. Returns an error if the map contains values that
// cannot be JSON-serialized.
func serializeDetail(detail map[string]any) (string, error) {
	b, err := json.Marshal(detail)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
