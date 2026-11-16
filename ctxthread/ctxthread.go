// Package ctxthread defines an Analyzer that reports where code does not
// thread its context.Context from the caller.
//
// A context carries the caller's deadline, cancellation and trace. Code that
// makes its own loses all three without a visible sign, and code that hides
// one in a struct or a global, or uses it to pass values, makes the data flow
// invisible at the call site.
//
// Reported, in non-test files of non-main packages, skipping generated files:
//   - a call to context.Background or context.TODO;
//   - each use of a package-level variable initialised by one of them;
//   - a struct field of type context.Context, embedded or named;
//   - a call to context.WithValue;
//   - a call to the Value method of a context.Context.
//
// Package main and tests are where a context legitimately starts, so they
// are skipped.
package ctxthread

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/types/typeutil"
)

// Analyzer reports contexts created from nothing, stored, or used as a value bag.
var Analyzer = &analysis.Analyzer{
	Name: "ctxthread",
	Doc:  "ctxthread: report a context created from nothing, stored in a struct or global, or used as a value bag\n\nReports context.Background and context.TODO, uses of a package-level variable holding one, struct fields of type context.Context, context.WithValue, and ctx.Value, in non-test, non-generated files outside package main.",
	URL:  "https://github.com/chrisrob11/gosloplint#ctxthread",
	Run:  run,
}

func run(pass *analysis.Pass) (any, error) {
	if pass.Pkg.Name() == "main" {
		return nil, nil
	}
	var files []*ast.File
	for _, f := range pass.Files {
		name := pass.Fset.File(f.Pos()).Name()
		if !strings.HasSuffix(name, "_test.go") && !ast.IsGenerated(f) {
			files = append(files, f)
		}
	}
	standins := standinVars(pass, files)
	for _, f := range files {
		ast.Inspect(f, func(n ast.Node) bool {
			switch x := n.(type) {
			case *ast.StructType:
				checkStruct(pass, x)
			case *ast.CallExpr:
				checkCall(pass, x)
			case *ast.Ident:
				if v, ok := pass.TypesInfo.Uses[x].(*types.Var); ok && standins[v] {
					pass.Reportf(x.Pos(), "%s holds a context made from nothing: thread the caller's ctx instead", x.Name)
				}
			}
			return true
		})
	}
	return nil, nil
}

// standinVars returns the package-level variables initialised by
// context.Background or context.TODO.
func standinVars(pass *analysis.Pass, files []*ast.File) map[*types.Var]bool {
	out := map[*types.Var]bool{}
	for _, f := range files {
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, spec := range gd.Specs {
				vs := spec.(*ast.ValueSpec)
				for i, val := range vs.Values {
					call, ok := val.(*ast.CallExpr)
					if !ok || i >= len(vs.Names) || !isContextFunc(pass, call, "Background", "TODO") {
						continue
					}
					if v, ok := pass.TypesInfo.Defs[vs.Names[i]].(*types.Var); ok {
						out[v] = true
					}
				}
			}
		}
	}
	return out
}

func checkStruct(pass *analysis.Pass, st *ast.StructType) {
	for _, field := range st.Fields.List {
		if !isContextType(pass.TypesInfo.TypeOf(field.Type)) {
			continue
		}
		const msg = "context.Context stored in a struct: pass ctx as the first parameter of each call instead"
		if len(field.Names) == 0 {
			pass.Reportf(field.Pos(), msg)
		}
		for _, name := range field.Names {
			pass.Reportf(name.Pos(), msg)
		}
	}
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr) {
	switch {
	case isContextFunc(pass, call, "Background", "TODO"):
		fn := typeutil.Callee(pass.TypesInfo, call)
		pass.Reportf(call.Pos(), "context.%s creates a context from nothing: take ctx from the caller", fn.Name())
	case isContextFunc(pass, call, "WithValue"):
		pass.Reportf(call.Pos(), "context.WithValue uses the context as a value bag: pass the value as a parameter")
	case isContextValue(pass, call):
		pass.Reportf(call.Pos(), "ctx.Value reads from the context as a value bag: pass the value as a parameter")
	}
}

// isContextFunc reports whether call invokes one of the named package-level
// functions of package context.
func isContextFunc(pass *analysis.Pass, call *ast.CallExpr, names ...string) bool {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	if !ok || fn.Pkg() == nil || fn.Pkg().Path() != "context" || fn.Signature().Recv() != nil {
		return false
	}
	for _, n := range names {
		if fn.Name() == n {
			return true
		}
	}
	return false
}

// isContextValue reports whether call is the Value method of context.Context,
// on a context or on a type that embeds one. It is the only exported Value
// method in package context.
func isContextValue(pass *analysis.Pass, call *ast.CallExpr) bool {
	fn, ok := typeutil.Callee(pass.TypesInfo, call).(*types.Func)
	return ok && fn.Pkg() != nil && fn.Pkg().Path() == "context" && fn.Name() == "Value" &&
		fn.Signature().Recv() != nil
}

// isContextType reports whether t is context.Context.
func isContextType(t types.Type) bool {
	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return false
	}
	obj := named.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "context" && obj.Name() == "Context"
}
