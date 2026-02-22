package classify

import (
	"go/ast"
	"go/types"
	"unicode"
	"unicode/utf8"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// maxVisibilityWeight is the maximum total weight for API surface
// visibility signals.
const maxVisibilityWeight = 20

// visibilityDimensions defines the weight per visibility dimension.
const (
	exportedFunctionWeight = 8
	exportedReturnWeight   = 6
	exportedReceiverWeight = 6
)

// AnalyzeVisibilitySignal checks if the side effect is observable
// through the exported API surface. The score is graduated: exported
// function, exported return types, and exported receiver types each
// contribute independently up to maxVisibilityWeight total.
func AnalyzeVisibilitySignal(
	funcDecl *ast.FuncDecl,
	funcObj types.Object,
	effectType taxonomy.SideEffectType,
) taxonomy.Signal {
	if funcDecl == nil || funcObj == nil {
		return taxonomy.Signal{}
	}

	weight := 0
	var reasons []string

	// Dimension 1: Is the function itself exported?
	if funcObj.Exported() {
		weight += exportedFunctionWeight
		reasons = append(reasons, "function is exported")
	}

	// Dimension 2: Are return types exported/visible?
	if funcDecl.Type.Results != nil {
		for _, result := range funcDecl.Type.Results.List {
			if isExportedType(result.Type) {
				weight += exportedReturnWeight
				reasons = append(reasons, "return type is exported")
				break // Count once per dimension
			}
		}
	}

	// Dimension 3: Is the receiver type exported?
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		recvType := funcDecl.Recv.List[0].Type
		if isExportedType(recvType) {
			weight += exportedReceiverWeight
			reasons = append(reasons, "receiver type is exported")
		}
	}

	// Clamp to max.
	if weight > maxVisibilityWeight {
		weight = maxVisibilityWeight
	}

	if weight == 0 {
		return taxonomy.Signal{}
	}

	reasoning := "API surface visibility:"
	for _, r := range reasons {
		reasoning += " " + r + ";"
	}

	return taxonomy.Signal{
		Source:    "visibility",
		Weight:    weight,
		Reasoning: reasoning,
	}
}

// isExportedType checks if an AST type expression refers to an
// exported type.
func isExportedType(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.IsExported()
	case *ast.StarExpr:
		return isExportedType(t.X)
	case *ast.SelectorExpr:
		return isExported(t.Sel.Name)
	case *ast.ArrayType:
		return isExportedType(t.Elt)
	case *ast.MapType:
		return isExportedType(t.Key) || isExportedType(t.Value)
	default:
		return false
	}
}

// isExported checks if a name starts with an uppercase letter.
// Uses utf8.DecodeRuneInString for Unicode correctness.
func isExported(name string) bool {
	if name == "" {
		return false
	}
	r, _ := utf8.DecodeRuneInString(name)
	return unicode.IsUpper(r)
}
