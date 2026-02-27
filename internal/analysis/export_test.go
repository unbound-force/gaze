package analysis

import (
	"go/ast"
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
