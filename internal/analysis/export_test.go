package analysis

import (
	"go/ast"

	"golang.org/x/tools/go/packages"
)

// FindFuncDecl is exported for testing. See findFuncDecl.
func FindFuncDecl(pkg *packages.Package, name string) *ast.FuncDecl {
	return findFuncDecl(pkg, name)
}

// FindMethodDecl is exported for testing. See findMethodDecl.
func FindMethodDecl(pkg *packages.Package, recvType, methodName string) *ast.FuncDecl {
	return findMethodDecl(pkg, recvType, methodName)
}
