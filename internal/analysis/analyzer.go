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

	"golang.org/x/tools/go/packages"
	"golang.org/x/tools/go/ssa"

	"github.com/unbound-force/gaze/internal/loader"
	"github.com/unbound-force/gaze/internal/taxonomy"
)

// Options configures the analysis behavior.
type Options struct {
	// IncludeUnexported includes unexported functions in package-
	// level analysis.
	IncludeUnexported bool

	// FunctionFilter limits analysis to a specific function name.
	// Empty string means analyze all functions.
	FunctionFilter string

	// Version is the Gaze version string to embed in metadata.
	// If empty, defaults to "dev".
	Version string
}

// Analyze performs side effect analysis on all functions in the
// loaded package. Returns a slice of AnalysisResult (one per
// function) and any error encountered during analysis.
func Analyze(pkg *packages.Package, opts Options) ([]taxonomy.AnalysisResult, error) {
	start := time.Now()

	fset := pkg.Fset

	// Build SSA once for the entire package to avoid redundant
	// reconstruction per function.
	ssaPkg := BuildSSA(pkg)

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

			result := analyzeFunction(fset, pkg, ssaPkg, fd)
			results = append(results, result)
		}

		// Analyze sentinel errors at file level.
		if opts.FunctionFilter == "" {
			sentinels := AnalyzeSentinels(fset, file, pkg.PkgPath)
			if len(sentinels) > 0 {
				// Attach sentinels to a synthetic package-level
				// result. The function name "<package>" indicates
				// these are package-level declarations, not
				// associated with any specific function. Each
				// sentinel's Target field identifies the specific
				// variable (e.g., "ErrNotFound") and the Location
				// field points to its declaration site.
				fileName := fset.Position(file.Pos()).Filename
				results = append(results, taxonomy.AnalysisResult{
					Target: taxonomy.FunctionTarget{
						Package:  pkg.PkgPath,
						Function: "<package>",
						Location: fileName,
					},
					SideEffects: sentinels,
					Metadata:    buildMetadata(start, opts.Version),
				})
			}
		}
	}

	// Update metadata timing for all results.
	for i := range results {
		results[i].Metadata = buildMetadata(start, opts.Version)
	}

	return results, nil
}

// AnalyzeFunction performs side effect analysis on a single function.
// For analyzing multiple functions in the same package, prefer
// Analyze() which builds SSA once, or use AnalyzeFunctionWithSSA
// with a pre-built SSA package.
func AnalyzeFunction(
	pkg *packages.Package,
	fd *ast.FuncDecl,
) taxonomy.AnalysisResult {
	return AnalyzeFunctionWithSSA(pkg, fd, nil)
}

// AnalyzeFunctionWithSSA performs side effect analysis on a single
// function using a pre-built SSA package. Returns an AnalysisResult
// containing the function target metadata and detected side effects.
// If ssaPkg is nil, SSA is built on-demand. Pre-building SSA via
// BuildSSA and reusing it across multiple calls avoids redundant SSA
// construction.
func AnalyzeFunctionWithSSA(
	pkg *packages.Package,
	fd *ast.FuncDecl,
	ssaPkg *ssa.Package,
) taxonomy.AnalysisResult {
	start := time.Now()
	fset := pkg.Fset

	if ssaPkg == nil {
		ssaPkg = BuildSSA(pkg)
	}

	result := analyzeFunction(fset, pkg, ssaPkg, fd)
	result.Metadata = buildMetadata(start, "")
	return result
}

// analyzeFunction runs all analyzers on a single function declaration.
func analyzeFunction(
	fset *token.FileSet,
	pkg *packages.Package,
	ssaPkg *ssa.Package,
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
			mutationEffects := AnalyzeMutations(fset, ssaPkg, fd, fnObj, pkgPath, funcName)
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
func buildMetadata(start time.Time, version string) taxonomy.Metadata {
	if version == "" {
		version = "dev"
	}
	return taxonomy.Metadata{
		GazeVersion: version,
		GoVersion:   runtime.Version(),
		Timestamp:   start,
		Duration:    time.Since(start),
		Warnings:    nil,
	}
}

// findFuncDecl finds a FuncDecl by name in a package.
// Returns nil if not found.
func findFuncDecl(pkg *packages.Package, name string) *ast.FuncDecl {
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

// findMethodDecl finds a method declaration by receiver type and
// method name. Returns nil if not found.
func findMethodDecl(pkg *packages.Package, recvType, methodName string) *ast.FuncDecl {
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
