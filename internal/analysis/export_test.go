package analysis

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"
)

// FindFuncDecl is exported for testing. See findFuncDecl.
func FindFuncDecl(pkg *packages.Package, name string) *ast.FuncDecl {
	return findFuncDecl(pkg, name)
}

// FindMethodDecl is exported for testing. See findMethodDecl.
func FindMethodDecl(pkg *packages.Package, recvType, methodName string) *ast.FuncDecl {
	return findMethodDecl(pkg, recvType, methodName)
}

// FindSSAFunction is exported for testing. See findSSAFunction.
func FindSSAFunction(ssaPkg *ssa.Package, fnObj *types.Func, fd *ast.FuncDecl) *ssa.Function {
	return findSSAFunction(ssaPkg, fnObj, fd)
}

// BaseTypeName is exported for testing. See baseTypeName.
func BaseTypeName(expr ast.Expr) string {
	return baseTypeName(expr)
}

// ExprRootIdent is exported for testing. See exprRootIdent.
func ExprRootIdent(expr ast.Expr) *ast.Ident {
	return exprRootIdent(expr)
}

// IsPointerArgStore is exported for testing. See isPointerArgStore.
func IsPointerArgStore(store *ssa.Store, ptrParams map[string]*ssa.Parameter) (string, bool) {
	return isPointerArgStore(store, ptrParams)
}

// MatchesWriteSignature is exported for testing. See matchesWriteSignature.
func MatchesWriteSignature(sig *types.Signature) bool {
	return matchesWriteSignature(sig)
}

// HandleReceiverAssignStmt is exported for testing. See handleReceiverAssignStmt.
func HandleReceiverAssignStmt(node *ast.AssignStmt, receiverName string) (bool, token.Pos) {
	return handleReceiverAssignStmt(node, receiverName)
}

// HandleReceiverIncDecStmt is exported for testing. See handleReceiverIncDecStmt.
func HandleReceiverIncDecStmt(node *ast.IncDecStmt, receiverName string) (bool, token.Pos) {
	return handleReceiverIncDecStmt(node, receiverName)
}

// HandleReceiverCallExpr is exported for testing. See handleReceiverCallExpr.
func HandleReceiverCallExpr(node *ast.CallExpr, receiverName string) (bool, token.Pos) {
	return handleReceiverCallExpr(node, receiverName)
}
