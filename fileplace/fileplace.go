// Package fileplace defines an Analyzer that reports code in the wrong file:
// where a method, type or test lives, and file names and sizes that say
// nothing about what the file holds.
//
// Rules:
//
//	receiver  a method outside its receiver type's file; once the type's
//	          files pass LongLines, <type>_<concern>.go is allowed too
//	testname  a _test.go file named for no source file of its directory
//	typefile  a type with methods declared in a file not named for it
//	filename  a non-test file named for nothing it declares
//	grabbag   a file named util, helpers, common, misc, types, base, impl,
//	          manager and the like (it is then not also filename or testname)
//	tiny      a non-test file under TinyLines that declares no type
//	long      a non-test file over LongLines
//
// A file is named for a declaration when its name, lower-cased with its
// underscores and any GOOS/GOARCH suffix dropped, spells the declaration's
// name or a run of its camelCase words, abbreviated or plural:
// volatility.go holds VolatilityOf, refs.go ReferencedColumns, txn.go
// Transaction. A file named for its package or directory is named, and so
// is errors.go holding errors. Generated files and doc.go are exempt from
// every rule.
//
// Each rule reads one package pass: its files, and, for testname, the names
// of the non-test .go files in its directory (read from the directory, so
// an external foo_test package, whose pass holds only its test files, is
// checked against them too, and so are files another build excludes). A
// whole-file finding is reported on the file's package clause.
package fileplace

import (
	"fmt"
	"strconv"
	"strings"

	"golang.org/x/tools/go/analysis"

	"github.com/chrisrob11/gosloplint/internal/listflag"
)

// Analyzer reports misplaced code. Its settings come from flags; New builds
// one with explicit settings instead.
var Analyzer = newAnalyzer(&Settings{}, true)

// New returns a fileplace Analyzer configured by s rather than by flags. It
// panics if s does not pass Validate.
func New(s Settings) *analysis.Analyzer {
	if err := s.Validate(); err != nil {
		panic("fileplace.New: " + err.Error())
	}
	return newAnalyzer(&s, false)
}

func newAnalyzer(s *Settings, withFlags bool) *analysis.Analyzer {
	a := &analysis.Analyzer{
		Name: "fileplace",
		Doc:  "fileplace: report code in the wrong file\n\nRules: receiver, testname, typefile, filename, grabbag, tiny (under 40 lines by default) and long (over 1500). -disable turns rules off; -exempt exempts packages or files from one.",
		URL:  "https://github.com/chrisrob11/gosloplint#fileplace",
	}
	a.Run = func(pass *analysis.Pass) (any, error) {
		if err := s.Validate(); err != nil {
			return nil, err
		}
		newPkg(pass, *s).run()
		return nil, nil
	}
	if withFlags {
		a.Flags.Var(listflag.List{Items: &s.Disable}, "disable", "comma-separated rules not to run: "+strings.Join(Rules, ", "))
		a.Flags.IntVar(&s.TinyLines, "tiny-lines", 0, "size below which a file declaring no type is tiny (0: "+strconv.Itoa(DefaultTinyLines)+")")
		a.Flags.IntVar(&s.LongLines, "long-lines", 0, "size above which a file is long and a type may split by concern (0: "+strconv.Itoa(DefaultLongLines)+")")
		a.Flags.Var(listflag.List{Items: &s.GrabBag}, "grab-bag", "comma-separated grab-bag file name parts, replacing the default: "+strings.Join(DefaultGrabBag, ","))
		a.Flags.Var(exemptFlag{&s.Exempt}, "exempt", `comma-separated rule=pattern exemptions; a pattern is an import path pattern ("..." matches anything) or a file name glob`)
	}
	return a
}

// exemptFlag is the -exempt flag: comma-separated rule=pattern pairs.
type exemptFlag struct{ m *map[string][]string }

func (e exemptFlag) String() string {
	if e.m == nil {
		return ""
	}
	var out []string
	for rule, ps := range *e.m {
		for _, p := range ps {
			out = append(out, rule+"="+p)
		}
	}
	return strings.Join(out, ",")
}

func (e exemptFlag) Set(v string) error {
	*e.m = nil
	for _, pair := range strings.Split(v, ",") {
		if pair = strings.TrimSpace(pair); pair == "" {
			continue
		}
		rule, p, ok := strings.Cut(pair, "=")
		if !ok {
			return fmt.Errorf("%q is not rule=pattern", pair)
		}
		if *e.m == nil {
			*e.m = map[string][]string{}
		}
		(*e.m)[rule] = append((*e.m)[rule], p)
	}
	return nil
}
