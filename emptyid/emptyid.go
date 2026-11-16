// Package emptyid defines an Analyzer that reports a guard on an empty
// identifier whose body exits silently: return nil, return nil, nil, a bare
// return, or continue.
//
// An empty ID means two things that look the same at the guard: "absent, so
// nothing to do", and "a caller dropped it". A guard that exits silently
// turns the second into the first, so a mis-addressed operation reports
// success and the empty value surfaces, if ever, far from where it was lost.
// The fix is not to delete the guard but to make it loud: return an error,
// panic, or log at warn or error before leaving.
//
// The guard is read for what it means, not how it is spelled:
//
//	if userID == "" { return nil }
//	if !ok || orderID == "" { continue }  // riding along with another test
//	if len(sessionKey) == 0 { return }    // a slice or a string
//	switch userID { case "": return nil } // the switch spelling
//	if m["userID"] == "" { return nil }   // a map entry, named by its key
//
// What counts as an identifier is decided by the last camelCase or
// snake_case word of its name, matched case-insensitively against the
// -suffixes flag (default id,ids,key,keys,name). Matching whole words keeps
// "valid" and "grid" out. Files ending in _test.go are skipped.
package emptyid

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/camel"
)

// Analyzer reports silent guards on empty identifiers. Its suffixes come from
// the -suffixes flag; New builds one with explicit settings instead.
var Analyzer = newAnalyzer(defaultSuffixes(), true)

// DefaultSuffixes is the default value of the -suffixes flag.
const DefaultSuffixes = "id,ids,key,keys,name"

// Settings configures an Analyzer built by New.
type Settings struct {
	// Suffixes are the final name words that mark an identifier, matched
	// case-insensitively. Empty means DefaultSuffixes.
	Suffixes []string
}

// New returns an emptyid Analyzer configured by s rather than by flags.
func New(s Settings) *analysis.Analyzer {
	list := defaultSuffixes()
	if len(s.Suffixes) > 0 {
		list = suffixList{}
		for _, w := range s.Suffixes {
			list.add(w)
		}
	}
	return newAnalyzer(list, false)
}

func newAnalyzer(list suffixList, withFlag bool) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "emptyid",
		Doc:  "emptyid: report a guard on an empty ID whose body exits silently\n\nReports `if id == \"\" { return nil }` and its other spellings (len(x) == 0, a case \"\": clause, a guard joined with && or ||), when the body only returns nil, returns, or continues. What counts as an ID is set by -suffixes.",
		URL:  "https://github.com/chrisrob11/gosloplint#emptyid",
	}
	a.Run = func(pass *analysis.Pass) (any, error) {
		(&checker{pass: pass, suffixes: list}).run()
		return nil, nil
	}
	if withFlag {
		a.Flags.Var(&list, "suffixes",
			"comma-separated final name words that mark an identifier, matched case-insensitively")
	}
	return a
}

func defaultSuffixes() suffixList {
	list := suffixList{}
	_ = list.Set(DefaultSuffixes)
	return list
}

// suffixList is a comma-separated flag value.
type suffixList map[string]bool

func (s *suffixList) String() string {
	var out []string
	for w := range *s {
		out = append(out, w)
	}
	return strings.Join(out, ",")
}

func (s *suffixList) Set(v string) error {
	clear(*s)
	for _, w := range strings.Split(v, ",") {
		s.add(w)
	}
	return nil
}

func (s *suffixList) add(w string) {
	if w = strings.ToLower(strings.TrimSpace(w)); w != "" {
		(*s)[w] = true
	}
}

// checker holds one pass and the suffixes it matches.
type checker struct {
	pass     *analysis.Pass
	suffixes suffixList
}

func (c *checker) run() {
	pass := c.pass
	for _, file := range pass.Files {
		if strings.HasSuffix(pass.Fset.File(file.Pos()).Name(), "_test.go") {
			continue
		}
		ast.Inspect(file, func(n ast.Node) bool {
			switch st := n.(type) {
			case *ast.IfStmt:
				c.checkIf(st)
			case *ast.SwitchStmt:
				c.checkSwitch(st)
			}
			return true
		})
	}
}

func (c *checker) checkIf(st *ast.IfStmt) {
	// An else branch makes the guard a fork, not a silent skip.
	if st.Else != nil {
		return
	}
	name, ok := c.emptyGuard(st.Cond)
	if !ok {
		return
	}
	if exit, silent := silentExit(c.pass, st.Body.List); silent {
		report(c.pass, st.Pos(), name, exit)
	}
}

// checkSwitch reports the switch spelling of the same guard: a case "":
// clause on a switch whose tag is an identifier.
func (c *checker) checkSwitch(sw *ast.SwitchStmt) {
	if sw.Tag == nil {
		return
	}
	name, ok := c.subject(sw.Tag)
	if !ok {
		return
	}
	for _, cl := range sw.Body.List {
		cc, ok := cl.(*ast.CaseClause)
		if !ok || len(cc.List) != 1 || !isEmptyString(c.pass, cc.List[0]) {
			continue
		}
		if exit, silent := silentExit(c.pass, cc.Body); silent {
			report(c.pass, cc.Pos(), name, exit)
		}
	}
}

func report(pass *analysis.Pass, pos token.Pos, name, exit string) {
	pass.Reportf(pos, "guard on empty %s exits silently (%s): return an error or log, so a dropped ID is not reported as success", name, exit)
}

// emptyGuard reports whether cond tests an identifier for emptiness, either
// alone or as one operand of && or ||, and returns the identifier's name.
func (c *checker) emptyGuard(cond ast.Expr) (string, bool) {
	switch e := cond.(type) {
	case *ast.ParenExpr:
		return c.emptyGuard(e.X)
	case *ast.BinaryExpr:
		switch e.Op {
		case token.LAND, token.LOR:
			if name, ok := c.emptyGuard(e.X); ok {
				return name, true
			}
			return c.emptyGuard(e.Y)
		case token.EQL:
			return c.equalityGuard(e)
		default:
			return "", false
		}
	}
	return "", false
}

// equalityGuard reads x == "" and len(x) == 0, in either operand order.
func (c *checker) equalityGuard(bin *ast.BinaryExpr) (string, bool) {
	pass := c.pass
	for _, pair := range [2][2]ast.Expr{{bin.X, bin.Y}, {bin.Y, bin.X}} {
		subj, other := pair[0], pair[1]
		switch {
		case isEmptyString(pass, other):
			if name, ok := c.subject(subj); ok {
				return name, true
			}
		case isZero(pass, other):
			if arg, ok := lenArg(pass, subj); ok {
				if name, ok := c.subject(arg); ok {
					return name, true
				}
			}
		}
	}
	return "", false
}

// subject names e and reports whether that name looks like an identifier.
func (c *checker) subject(e ast.Expr) (string, bool) {
	name, ok := exprName(e)
	if !ok || !c.looksLikeIdentifier(name) {
		return "", false
	}
	return name, true
}

// lenArg returns x from a call to the builtin len(x).
func lenArg(pass *analysis.Pass, e ast.Expr) (ast.Expr, bool) {
	call, ok := e.(*ast.CallExpr)
	if !ok || len(call.Args) != 1 {
		return nil, false
	}
	id, ok := call.Fun.(*ast.Ident)
	if !ok {
		return nil, false
	}
	b, ok := pass.TypesInfo.Uses[id].(*types.Builtin)
	if !ok || b.Name() != "len" {
		return nil, false
	}
	return call.Args[0], true
}

// isEmptyString reports whether e is a constant empty string, as a literal
// or a named constant.
func isEmptyString(pass *analysis.Pass, e ast.Expr) bool {
	v := pass.TypesInfo.Types[e].Value
	return v != nil && v.Kind() == constant.String && constant.StringVal(v) == ""
}

// isZero reports whether e is the constant integer 0.
func isZero(pass *analysis.Pass, e ast.Expr) bool {
	v := pass.TypesInfo.Types[e].Value
	return v != nil && v.Kind() == constant.Int && constant.Sign(v) == 0
}

// exprName renders x, x.y, f().y and m["k"] as a name. A map entry with a
// string-literal key is named by the key, which says what was guarded; any
// other index falls back to the container.
func exprName(e ast.Expr) (string, bool) {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name, true
	case *ast.SelectorExpr:
		if base, ok := exprName(v.X); ok {
			return base + "." + v.Sel.Name, true
		}
		return v.Sel.Name, true
	case *ast.CallExpr:
		return exprName(v.Fun)
	case *ast.IndexExpr:
		if lit, ok := v.Index.(*ast.BasicLit); ok && lit.Kind == token.STRING && len(lit.Value) > 2 {
			return lit.Value[1 : len(lit.Value)-1], true
		}
		return exprName(v.X)
	case *ast.ParenExpr:
		return exprName(v.X)
	}
	return "", false
}

// looksLikeIdentifier reports whether the final word of name is one of the
// configured suffixes.
func (c *checker) looksLikeIdentifier(name string) bool {
	words := camel.Words(name)
	return len(words) > 0 && c.suffixes[strings.ToLower(words[len(words)-1])]
}

// silentExit reports whether body does nothing but leave. Statements before
// the exit that could signal (an error, a panic, a warning) make it loud.
func silentExit(pass *analysis.Pass, body []ast.Stmt) (string, bool) {
	if len(body) == 0 {
		return "", false
	}
	for _, st := range body[:len(body)-1] {
		if maySignal(st) {
			return "", false
		}
	}
	switch t := body[len(body)-1].(type) {
	case *ast.ReturnStmt:
		if len(t.Results) == 0 {
			return "return", true
		}
		for _, r := range t.Results {
			if !pass.TypesInfo.Types[r].IsNil() {
				return "", false
			}
		}
		return "return nil", true
	case *ast.BranchStmt:
		if t.Tok == token.CONTINUE {
			return "continue", true
		}
	}
	return "", false
}

// maySignal reports whether a statement calls something whose name says it
// surfaces a problem: panic, an error constructor, a warning or error log.
func maySignal(st ast.Stmt) bool {
	signal := false
	ast.Inspect(st, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok || signal {
			return !signal
		}
		name, ok := exprName(call.Fun)
		if !ok {
			return true
		}
		l := strings.ToLower(name)
		signal = strings.Contains(l, "panic") || strings.Contains(l, "error") ||
			strings.Contains(l, "warn") || strings.Contains(l, "wrap") ||
			strings.Contains(l, "new") && strings.Contains(l, "err")
		return !signal
	})
	return signal
}
