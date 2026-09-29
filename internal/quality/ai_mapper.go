package quality

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/token"
	"io"
	"strings"

	"golang.org/x/tools/go/ssa"

	"github.com/unbound-force/gaze/v2/internal/taxonomy"
)

// AIMapperContext contains the context needed for AI-assisted
// assertion mapping. It is passed to AIMapperFunc when all
// mechanical passes (direct identity, indirect root, helper
// bridge, inline call) fail to map an assertion to a side effect.
type AIMapperContext struct {
	// AssertionSource is the assertion expression as readable
	// Go source code (e.g., "got != want" or "assert.Equal(t, got, want)").
	AssertionSource string

	// AssertionKind is the pattern type that was detected
	// (e.g., stdlib_comparison, testify_equal).
	AssertionKind AssertionKind

	// TestFuncSource is the full test function source code,
	// providing context for the AI to understand variable
	// assignments, setup calls, and the assertion's role.
	TestFuncSource string

	// TargetFunc is the qualified name of the function under test
	// (e.g., "(*Counter).Increment" or "pkg.Foo").
	TargetFunc string

	// SideEffects is the list of side effects produced by the
	// target function. The AI should determine which (if any)
	// the assertion verifies and return its ID.
	SideEffects []taxonomy.SideEffect
}

// AIMapperFunc evaluates whether an unmapped assertion verifies a
// side effect of the target function. It is called only for
// assertions that all mechanical mapping passes failed to map.
//
// Returns the matched side effect ID if the AI determines the
// assertion verifies one of the provided effects, or empty string
// if no match. Returns an error if the AI evaluation itself fails
// (network error, model unavailable, etc.) — the assertion will
// remain unmapped and a warning will be logged.
type AIMapperFunc func(ctx AIMapperContext) (effectID string, err error)

// tryAIMapping constructs an AIMapperContext from the assertion site,
// target function, and effects, then calls the AI mapper. Returns
// a mapping at confidence 50 if the AI finds a match, or nil.
func tryAIMapping(
	site AssertionSite,
	targetFunc *ssa.Function,
	effects []taxonomy.SideEffect,
	fset *token.FileSet,
	aiMapperFn AIMapperFunc,
	stderr io.Writer,
) *taxonomy.AssertionMapping {

	assertionSrc := extractExprSource(site.Expr, fset)
	testFuncSrc := extractFuncSource(site.FuncDecl, fset)

	targetName := ""
	if targetFunc != nil {
		targetName = qualifiedSSAName(targetFunc)
	}

	// Copy effects to prevent the callback from mutating the
	// caller's slice (AIMapperFunc is user-supplied).
	effectsCopy := make([]taxonomy.SideEffect, len(effects))
	copy(effectsCopy, effects)

	ctx := AIMapperContext{
		AssertionSource: assertionSrc,
		AssertionKind:   site.Kind,
		TestFuncSource:  testFuncSrc,
		TargetFunc:      targetName,
		SideEffects:     effectsCopy,
	}

	effectID, err := aiMapperFn(ctx)
	if err != nil {
		if stderr != nil {
			_, _ = fmt.Fprintf(stderr, "warning: AI mapper failed for assertion at %s: %v\n",
				site.Location, err)
		}
		return nil
	}
	if effectID == "" {
		return nil
	}

	// Validate that the returned effect ID exists.
	for _, e := range effects {
		if e.ID == effectID {
			return &taxonomy.AssertionMapping{
				AssertionLocation: site.Location,
				AssertionType:     mapKindToType(site.Kind),
				SideEffectID:      effectID,
				Confidence:        50, // AI-evaluated match
			}
		}
	}

	return nil
}

// BuildAIMapperPrompt constructs a structured prompt from an
// AIMapperContext for sending to an AI model. The prompt asks
// the AI to determine which (if any) side effect the assertion
// verifies, and to respond with ONLY the effect ID or "NONE".
//
// This function is exported so callers can build their own
// AIMapperFunc implementations using any AI backend.
func BuildAIMapperPrompt(ctx AIMapperContext) string {
	var b strings.Builder

	b.WriteString("Given this Go test assertion:\n")
	b.WriteString("  ")
	b.WriteString(ctx.AssertionSource)
	b.WriteString("\n\n")

	b.WriteString("In this test function:\n")
	b.WriteString(ctx.TestFuncSource)
	b.WriteString("\n\n")

	b.WriteString("The target function ")
	b.WriteString(ctx.TargetFunc)
	b.WriteString(" produces these side effects:\n")
	for i, se := range ctx.SideEffects {
		fmt.Fprintf(&b, "  %d. [%s] %s (ID: %s)\n",
			i+1, se.Type, se.Description, se.ID)
	}
	b.WriteString("\n")

	b.WriteString("Does this assertion verify any of these side effects?\n")
	b.WriteString("Consider semantic relationships — for example, calling a getter\n")
	b.WriteString("after a setter to verify the mutation, or checking a return value\n")
	b.WriteString("through an intermediate variable or helper function.\n\n")
	b.WriteString("Respond with ONLY the effect ID if there is a match, or NONE if\n")
	b.WriteString("the assertion does not verify any of the listed side effects.\n")

	return b.String()
}

// ParseAIMapperResponse extracts a side effect ID from an AI model's
// response. It trims whitespace and checks for the "NONE" sentinel.
// Returns the effect ID or empty string if no match.
func ParseAIMapperResponse(response string, validIDs map[string]bool) string {
	trimmed := strings.TrimSpace(response)

	// Check for explicit no-match.
	if strings.EqualFold(trimmed, "NONE") {
		return ""
	}

	// Check if the response is a valid effect ID.
	if validIDs[trimmed] {
		return trimmed
	}

	// The AI may include the ID embedded in a sentence.
	// Search for valid IDs in the response. If multiple are found,
	// return empty (ambiguous — cannot deterministically pick one).
	var found string
	for id := range validIDs {
		if strings.Contains(trimmed, id) {
			if found != "" {
				return "" // ambiguous: multiple IDs found
			}
			found = id
		}
	}

	return found
}

// extractExprSource renders an AST expression as readable Go source.
// Returns a best-effort string — falls back to a position string on error.
func extractExprSource(expr ast.Expr, fset *token.FileSet) string {
	if expr == nil {
		return "<expr>"
	}
	if fset == nil {
		fset = token.NewFileSet()
	}
	var buf strings.Builder
	if err := format.Node(&buf, fset, expr); err != nil {
		return fmt.Sprintf("<expr at %s>", fset.Position(expr.Pos()))
	}
	return buf.String()
}

// extractFuncSource renders a function declaration as readable Go source.
// Returns a best-effort string — falls back to the function name on error.
func extractFuncSource(decl *ast.FuncDecl, fset *token.FileSet) string {
	if decl == nil {
		return "<func>"
	}
	if fset == nil {
		fset = token.NewFileSet()
	}
	var buf strings.Builder
	if err := format.Node(&buf, fset, decl); err != nil {
		return fmt.Sprintf("<func %s>", decl.Name.Name)
	}
	return buf.String()
}
