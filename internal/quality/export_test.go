package quality

import (
	"go/ast"
	"go/types"
)

// ResolveExprRoot exports resolveExprRoot for testing.
var ResolveExprRoot = resolveExprRoot

// ResolveExprRootFunc is the exported type signature for resolveExprRoot.
type ResolveExprRootFunc = func(expr ast.Expr, info *types.Info) *ast.Ident
