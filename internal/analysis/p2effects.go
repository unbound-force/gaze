package analysis

import (
	"fmt"
	"go/ast"
	"go/token"
	"go/types"

	"github.com/jflowers/gaze/internal/taxonomy"
)

// p2SelectorEffects maps import path → function name →
// SideEffectType for P2 effects detectable via selector calls
// (pkg.Func). Keys are full import paths (not short names) so
// that import aliases are handled correctly via resolveImportPath.
var p2SelectorEffects = map[string]map[string]taxonomy.SideEffectType{
	"os": {
		// FileSystemWrite
		"WriteFile": taxonomy.FileSystemWrite,
		"Create":    taxonomy.FileSystemWrite,
		"OpenFile":  taxonomy.FileSystemWrite,
		"Mkdir":     taxonomy.FileSystemWrite,
		"MkdirAll":  taxonomy.FileSystemWrite,
		"Rename":    taxonomy.FileSystemWrite,

		// FileSystemDelete
		"Remove":    taxonomy.FileSystemDelete,
		"RemoveAll": taxonomy.FileSystemDelete,

		// FileSystemMeta
		"Chmod":    taxonomy.FileSystemMeta,
		"Chown":    taxonomy.FileSystemMeta,
		"Chtimes":  taxonomy.FileSystemMeta,
		"Lchown":   taxonomy.FileSystemMeta,
		"Link":     taxonomy.FileSystemMeta,
		"Symlink":  taxonomy.FileSystemMeta,
		"Truncate": taxonomy.FileSystemMeta,
	},
	"log": {
		// LogWrite
		"Print":   taxonomy.LogWrite,
		"Printf":  taxonomy.LogWrite,
		"Println": taxonomy.LogWrite,
		"Fatal":   taxonomy.LogWrite,
		"Fatalf":  taxonomy.LogWrite,
		"Fatalln": taxonomy.LogWrite,
		"Panic":   taxonomy.LogWrite,
		"Panicf":  taxonomy.LogWrite,
		"Panicln": taxonomy.LogWrite,
	},
	"log/slog": {
		// LogWrite (log/slog)
		"Debug": taxonomy.LogWrite,
		"Info":  taxonomy.LogWrite,
		"Warn":  taxonomy.LogWrite,
		"Error": taxonomy.LogWrite,
	},
	"context": {
		// ContextCancellation
		"WithCancel":   taxonomy.ContextCancellation,
		"WithTimeout":  taxonomy.ContextCancellation,
		"WithDeadline": taxonomy.ContextCancellation,
	},
}

// AnalyzeP2Effects detects P2-tier side effects in a function body
// using AST inspection. This covers:
//   - GoroutineSpawn: go statements
//   - Panic: calls to builtin panic()
//   - FileSystemWrite: os.WriteFile, os.Create, os.OpenFile, os.Mkdir, etc.
//   - FileSystemDelete: os.Remove, os.RemoveAll
//   - FileSystemMeta: os.Chmod, os.Chown, os.Symlink, etc.
//   - LogWrite: log.Print*, log.Fatal*, slog.Info, etc.
//   - ContextCancellation: context.WithCancel, WithTimeout, WithDeadline
//   - CallbackInvocation: calling function-typed parameters
//   - DatabaseWrite: db.Exec, db.ExecContext on *sql.DB/*sql.Tx/*sql.Stmt
//   - DatabaseTransaction: db.Begin, db.BeginTx on *sql.DB
func AnalyzeP2Effects(
	fset *token.FileSet,
	info *types.Info,
	fd *ast.FuncDecl,
	pkg string,
	funcName string,
) []taxonomy.SideEffect {
	if fd.Body == nil {
		return nil
	}

	var effects []taxonomy.SideEffect
	seen := make(map[string]bool)

	// Build set of function-typed parameter names for callback detection.
	funcParams := collectFuncParams(fd, info)

	ast.Inspect(fd.Body, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.GoStmt:
			// GoroutineSpawn: any go statement.
			key := fmt.Sprintf("goroutine:%d", fset.Position(node.Pos()).Line)
			if !seen[key] {
				seen[key] = true
				loc := fset.Position(node.Pos()).String()
				effects = append(effects, taxonomy.SideEffect{
					ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.GoroutineSpawn), key),
					Type:        taxonomy.GoroutineSpawn,
					Tier:        taxonomy.TierP2,
					Location:    loc,
					Description: "spawns a goroutine",
				})
			}

		case *ast.CallExpr:
			// Panic: builtin panic() call.
			if isPanicCall(node, info) {
				key := fmt.Sprintf("panic:%d", fset.Position(node.Pos()).Line)
				if !seen[key] {
					seen[key] = true
					loc := fset.Position(node.Pos()).String()
					effects = append(effects, taxonomy.SideEffect{
						ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.Panic), key),
						Type:        taxonomy.Panic,
						Tier:        taxonomy.TierP2,
						Location:    loc,
						Description: "calls panic()",
					})
				}
			}

			// Selector-based detection (pkg.Func pattern).
			if sel, ok := node.Fun.(*ast.SelectorExpr); ok {
				if ident, ok := sel.X.(*ast.Ident); ok {
					// Resolve the identifier to its actual import
					// path via types.Info, falling back to the AST
					// name if type info is unavailable. This
					// correctly handles import aliases (e.g.,
					// import myos "os") and avoids false positives
					// from user packages with the same short name.
					pkgName := resolveImportPath(ident, info)
					if funcs, ok := p2SelectorEffects[pkgName]; ok {
						if effectType, ok := funcs[sel.Sel.Name]; ok {
							key := fmt.Sprintf("%s:%s.%s:%d",
								effectType, ident.Name, sel.Sel.Name,
								fset.Position(node.Pos()).Line)
							if !seen[key] {
								seen[key] = true
								loc := fset.Position(node.Pos()).String()
								effects = append(effects, taxonomy.SideEffect{
									ID:          taxonomy.GenerateID(pkg, funcName, string(effectType), key),
									Type:        effectType,
									Tier:        taxonomy.TierP2,
									Location:    loc,
									Description: fmt.Sprintf("calls %s.%s", ident.Name, sel.Sel.Name),
									Target:      fmt.Sprintf("%s.%s", ident.Name, sel.Sel.Name),
								})
							}
						}
					}
				}

				// Database detection: Exec/ExecContext/Begin/BeginTx on *sql.DB/Tx/Stmt.
				if isDatabaseMethod(sel, info) {
					effectType := databaseMethodEffect(sel.Sel.Name)
					if effectType != "" {
						key := fmt.Sprintf("%s:%s:%d",
							effectType, sel.Sel.Name,
							fset.Position(node.Pos()).Line)
						if !seen[key] {
							seen[key] = true
							loc := fset.Position(node.Pos()).String()
							effects = append(effects, taxonomy.SideEffect{
								ID:          taxonomy.GenerateID(pkg, funcName, string(effectType), key),
								Type:        effectType,
								Tier:        taxonomy.TierP2,
								Location:    loc,
								Description: fmt.Sprintf("calls %s on database type", sel.Sel.Name),
								Target:      sel.Sel.Name,
							})
						}
					}
				}
			}

			// Callback invocation: calling a function-typed parameter.
			if ident, ok := node.Fun.(*ast.Ident); ok {
				if funcParams[ident.Name] {
					key := fmt.Sprintf("callback:%s:%d",
						ident.Name, fset.Position(node.Pos()).Line)
					if !seen[key] {
						seen[key] = true
						loc := fset.Position(node.Pos()).String()
						effects = append(effects, taxonomy.SideEffect{
							ID:          taxonomy.GenerateID(pkg, funcName, string(taxonomy.CallbackInvocation), key),
							Type:        taxonomy.CallbackInvocation,
							Tier:        taxonomy.TierP2,
							Location:    loc,
							Description: fmt.Sprintf("invokes callback parameter '%s'", ident.Name),
							Target:      ident.Name,
						})
					}
				}
			}
		}
		return true
	})

	return effects
}

// resolveImportPath resolves an AST identifier to its actual import
// path using type information. For example, if the source has
// `import myos "os"`, then `myos.WriteFile(...)` will resolve to
// "os". Falls back to the AST identifier name if type info is
// unavailable or the identifier is not a package name.
func resolveImportPath(ident *ast.Ident, info *types.Info) string {
	if info != nil {
		if obj := info.Uses[ident]; obj != nil {
			if pkgName, ok := obj.(*types.PkgName); ok {
				return pkgName.Imported().Path()
			}
		}
	}
	return ident.Name
}

// isPanicCall checks if a call expression is a call to the
// builtin panic() function using type resolution to avoid false
// positives from user-defined functions named "panic".
func isPanicCall(call *ast.CallExpr, info *types.Info) bool {
	ident, ok := call.Fun.(*ast.Ident)
	if !ok {
		return false
	}
	if ident.Name != "panic" {
		return false
	}
	// Verify this is the builtin panic, not a user-defined function.
	if info != nil {
		if obj := info.Uses[ident]; obj != nil {
			_, isBuiltin := obj.(*types.Builtin)
			return isBuiltin
		}
	}
	// Fallback: accept name match when type info is unavailable.
	return true
}

// collectFuncParams returns a set of parameter names that have
// function types (used for CallbackInvocation detection).
func collectFuncParams(fd *ast.FuncDecl, info *types.Info) map[string]bool {
	params := make(map[string]bool)
	if fd.Type.Params == nil || info == nil {
		return params
	}
	for _, field := range fd.Type.Params.List {
		for _, name := range field.Names {
			// Check if the parameter type is a function type.
			if obj := info.Defs[name]; obj != nil {
				if _, ok := obj.Type().Underlying().(*types.Signature); ok {
					params[name.Name] = true
				}
			}
		}
	}
	return params
}

// isDatabaseMethod checks if a selector expression's receiver is a
// database/sql type (*sql.DB, *sql.Tx, *sql.Stmt).
func isDatabaseMethod(sel *ast.SelectorExpr, info *types.Info) bool {
	if info == nil {
		return false
	}
	tv, ok := info.Types[sel.X]
	if !ok {
		return false
	}
	typeStr := tv.Type.String()
	return typeStr == "*database/sql.DB" ||
		typeStr == "*database/sql.Tx" ||
		typeStr == "*database/sql.Stmt"
}

// databaseMethodEffect returns the P2 SideEffectType for a database
// method name, or empty string if it's not a write/transaction method.
func databaseMethodEffect(methodName string) taxonomy.SideEffectType {
	switch methodName {
	case "Exec", "ExecContext":
		return taxonomy.DatabaseWrite
	case "Begin", "BeginTx":
		return taxonomy.DatabaseTransaction
	default:
		return ""
	}
}
