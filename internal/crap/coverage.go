package crap

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/tools/cover"
)

// FuncCoverage holds the coverage percentage for a single function.
type FuncCoverage struct {
	// File is the absolute filesystem path.
	File string `json:"file"`

	// FuncName is the function name (e.g., "Save" or "(*Store).Save").
	FuncName string `json:"func_name"`

	// StartLine is the function declaration start line.
	StartLine int `json:"start_line"`

	// EndLine is the function body end line.
	EndLine int `json:"end_line"`

	// CoveredStmts is the number of statements covered by tests.
	CoveredStmts int64 `json:"covered_stmts"`

	// TotalStmts is the total number of statements in the function.
	TotalStmts int64 `json:"total_stmts"`

	// Percentage is the coverage percentage (0-100).
	Percentage float64 `json:"percentage"`
}

// ParseCoverProfile reads a Go coverage profile and computes
// per-function coverage percentages.
//
// The moduleDir is the root directory of the Go module, used to
// resolve import paths to filesystem paths. If empty, the current
// directory is used.
func ParseCoverProfile(profilePath string, moduleDir string) ([]FuncCoverage, error) {
	profiles, err := cover.ParseProfiles(profilePath)
	if err != nil {
		return nil, err
	}

	if moduleDir == "" {
		moduleDir, _ = os.Getwd()
	}

	var results []FuncCoverage

	for _, profile := range profiles {
		// Resolve the import path to a filesystem path.
		filePath := resolveFilePath(profile.FileName, moduleDir)
		if filePath == "" {
			continue
		}

		// Find all functions in this source file.
		funcs, err := findFunctions(filePath)
		if err != nil {
			continue
		}

		// Compute coverage per function.
		for _, fn := range funcs {
			covered, total := funcCoverage(fn, profile)
			pct := 0.0
			if total > 0 {
				pct = 100.0 * float64(covered) / float64(total)
			}
			results = append(results, FuncCoverage{
				File:         filePath,
				FuncName:     fn.name,
				StartLine:    fn.startLine,
				EndLine:      fn.endLine,
				CoveredStmts: covered,
				TotalStmts:   total,
				Percentage:   pct,
			})
		}
	}

	return results, nil
}

// funcExtent describes a function's source position.
type funcExtent struct {
	name      string
	startLine int
	startCol  int
	endLine   int
	endCol    int
}

// findFunctions parses a Go source file and returns the extent of
// each function declaration.
func findFunctions(filePath string) ([]funcExtent, error) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, filePath, nil, 0)
	if err != nil {
		return nil, err
	}

	var funcs []funcExtent
	ast.Inspect(f, func(n ast.Node) bool {
		fn, ok := n.(*ast.FuncDecl)
		if !ok || fn.Body == nil {
			return true
		}
		start := fset.Position(fn.Pos())
		end := fset.Position(fn.End())

		name := fn.Name.Name
		if fn.Recv != nil && fn.Recv.NumFields() > 0 {
			recvType := recvTypeString(fn.Recv.List[0].Type)
			name = "(" + recvType + ")." + fn.Name.Name
		}

		funcs = append(funcs, funcExtent{
			name:      name,
			startLine: start.Line,
			startCol:  start.Column,
			endLine:   end.Line,
			endCol:    end.Column,
		})
		return true
	})
	return funcs, nil
}

// recvTypeString extracts the receiver type as a string.
func recvTypeString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.StarExpr:
		return "*" + recvTypeString(t.X)
	case *ast.Ident:
		return t.Name
	case *ast.IndexExpr:
		return recvTypeString(t.X) + "[" + recvTypeString(t.Index) + "]"
	default:
		return "?"
	}
}

// funcCoverage computes the covered and total statement counts for a
// function within a coverage profile.
func funcCoverage(fn funcExtent, profile *cover.Profile) (covered, total int64) {
	for _, b := range profile.Blocks {
		// Block entirely after the function — stop.
		if b.StartLine > fn.endLine {
			break
		}
		if b.StartLine == fn.endLine && b.StartCol >= fn.endCol {
			break
		}
		// Block entirely before the function — skip.
		if b.EndLine < fn.startLine {
			continue
		}
		if b.EndLine == fn.startLine && b.EndCol <= fn.startCol {
			continue
		}
		// Block overlaps the function.
		total += int64(b.NumStmt)
		if b.Count > 0 {
			covered += int64(b.NumStmt)
		}
	}
	return
}

// resolveFilePath maps a coverage profile filename (import path
// relative, e.g., "github.com/user/pkg/file.go") to an absolute
// filesystem path.
func resolveFilePath(profileName string, moduleDir string) string {
	// Try direct filesystem path first (some profiles use absolute paths).
	if filepath.IsAbs(profileName) {
		if _, err := os.Stat(profileName); err == nil {
			return profileName
		}
	}

	// The profile uses import-path-relative filenames like
	// "github.com/jflowers/gaze/internal/analysis/returns.go".
	// We need to map this to the actual file on disk.
	//
	// Strategy: strip the module prefix and join with moduleDir.
	// Read go.mod to find the module path.
	modulePath := readModulePath(moduleDir)
	if modulePath == "" {
		return ""
	}

	if strings.HasPrefix(profileName, modulePath) {
		rel := strings.TrimPrefix(profileName, modulePath)
		rel = strings.TrimPrefix(rel, "/")
		absPath := filepath.Join(moduleDir, rel)
		if _, err := os.Stat(absPath); err == nil {
			return absPath
		}
	}

	return ""
}

// readModulePath reads the module path from go.mod in the given directory.
func readModulePath(dir string) string {
	data, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module"))
		}
	}
	return ""
}
