// Package analysis provides the core side effect detection engine
// for Gaze. It uses AST and SSA analysis to detect observable side
// effects in Go functions.
package analysis

import (
	"go/ast"
	"go/token"
	"go/types"
	"runtime"
	"time"

	"github.com/jflowers/gaze/internal/loader"
	"github.com/jflowers/gaze/internal/taxonomy"
	"golang.org/x/tools/go/packages"
)

// Options configures the analysis behavior.
type Options struct {
	// IncludeUnexported includes unexported functions in package-
	// level analysis.
	IncludeUnexported bool

	// FunctionFilter limits analysis to a specific function name.
	// Empty string means analyze all functions.
	FunctionFilter string
}

// Analyze performs side effect analysis on all functions in the
// loaded package, returning an AnalysisResult per function.
func Analyze(pkg *packages.Package, opts Options) ([]taxonomy.AnalysisResult, error) {
	start := time.Now()

	fset := pkg.Fset
	var results []taxonomy.AnalysisResult

	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name == nil || fd.Body == nil {
				continue
			}

			// Apply filters.
			if opts.FunctionFilter != "" && fd.Name.Name != opts.FunctionFilter {
				continue
			}
			if !opts.IncludeUnexported && !fd.Name.IsExported() {
				continue
			}

			result := analyzeFunction(fset, pkg, file, fd)
			results = append(results, result)
		}

		// Analyze sentinel errors at file level.
		if opts.FunctionFilter == "" {
			sentinels := AnalyzeSentinels(fset, file, pkg.PkgPath)
			if len(sentinels) > 0 {
				// Attach sentinels to a synthetic package-level result.
				results = append(results, taxonomy.AnalysisResult{
					Target: taxonomy.FunctionTarget{
						Package:  pkg.PkgPath,
						Function: "<package>",
						Location: fset.Position(file.Pos()).String(),
					},
					SideEffects: sentinels,
					Metadata:    buildMetadata(start),
				})
			}
		}
	}

	// Update metadata timing for all results.
	for i := range results {
		results[i].Metadata = buildMetadata(start)
	}

	return results, nil
}

// AnalyzeFunction performs side effect analysis on a single function.
func AnalyzeFunction(
	pkg *packages.Package,
	fd *ast.FuncDecl,
) taxonomy.AnalysisResult {
	start := time.Now()
	fset := pkg.Fset

	// Find the file containing this function.
	var file *ast.File
	for _, f := range pkg.Syntax {
		for _, d := range f.Decls {
			if d == fd {
				file = f
				break
			}
		}
		if file != nil {
			break
		}
	}

	result := analyzeFunction(fset, pkg, file, fd)
	result.Metadata = buildMetadata(start)
	return result
}

// analyzeFunction runs all analyzers on a single function declaration.
func analyzeFunction(
	fset *token.FileSet,
	pkg *packages.Package,
	file *ast.File,
	fd *ast.FuncDecl,
) taxonomy.AnalysisResult {
	funcName := fd.Name.Name
	pkgPath := pkg.PkgPath

	target := taxonomy.FunctionTarget{
		Package:   pkgPath,
		Function:  funcName,
		Receiver:  receiverName(fd),
		Signature: funcSignature(fset, fd),
		Location:  fset.Position(fd.Pos()).String(),
	}

	var effects []taxonomy.SideEffect

	// 1. Return value analysis (AST-based).
	returnEffects := AnalyzeReturns(fset, pkg.TypesInfo, fd, pkgPath, funcName)
	effects = append(effects, returnEffects...)

	// 2. Mutation analysis (SSA-based).
	obj := pkg.TypesInfo.Defs[fd.Name]
	if obj != nil {
		if fnObj, ok := obj.(*types.Func); ok {
			mutationEffects := AnalyzeMutations(fset, pkg, fd, fnObj, pkgPath, funcName)
			effects = append(effects, mutationEffects...)
		}
	}

	// 3. P1-tier effects (AST-based).
	p1Effects := AnalyzeP1Effects(fset, pkg.TypesInfo, fd, pkgPath, funcName)
	effects = append(effects, p1Effects...)

	// 4. P2-tier effects (AST-based).
	p2Effects := AnalyzeP2Effects(fset, pkg.TypesInfo, fd, pkgPath, funcName)
	effects = append(effects, p2Effects...)

	return taxonomy.AnalysisResult{
		Target:      target,
		SideEffects: effects,
	}
}

// buildMetadata creates analysis metadata with current timing.
func buildMetadata(start time.Time) taxonomy.Metadata {
	return taxonomy.Metadata{
		GazeVersion: "0.1.0",
		GoVersion:   runtime.Version(),
		Duration:    time.Since(start),
		Warnings:    nil,
	}
}

// FindFuncDecl finds a FuncDecl by name in a package.
// Returns nil if not found.
func FindFuncDecl(pkg *packages.Package, name string) *ast.FuncDecl {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name == nil {
				continue
			}
			if fd.Name.Name == name {
				return fd
			}
		}
	}
	return nil
}

// FindMethodDecl finds a method declaration by receiver type and
// method name. Returns nil if not found.
func FindMethodDecl(pkg *packages.Package, recvType, methodName string) *ast.FuncDecl {
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			fd, ok := decl.(*ast.FuncDecl)
			if !ok || fd.Name == nil || fd.Recv == nil {
				continue
			}
			if fd.Name.Name != methodName {
				continue
			}
			recv := receiverName(fd)
			if recv == recvType {
				return fd
			}
		}
	}
	return nil
}

// LoadAndAnalyze is a convenience function that loads a package and
// runs analysis with the given options.
func LoadAndAnalyze(pattern string, opts Options) ([]taxonomy.AnalysisResult, error) {
	result, err := loader.Load(pattern)
	if err != nil {
		return nil, err
	}
	return Analyze(result.Pkg, opts)
}
