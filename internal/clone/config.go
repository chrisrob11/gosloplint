package clone

import (
	"cmp"
	"flag"
	"fmt"
	"path"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/chrisrob11/gosloplint/internal/listflag"
	"github.com/chrisrob11/gosloplint/internal/pkgpattern"
)

// The defaults a zero Config field takes. DefaultMinNodes is about fifteen
// lines of ordinary Go, at three to four nodes a line.
const (
	DefaultMinNodes      = 60
	DefaultMinStatements = 1
)

// Config is what dupl.Settings holds, shared with `gosloplint dupl`. The
// zero value is the defaults.
type Config struct {
	// MinNodes is the size a run must reach, in syntax nodes. Zero means
	// DefaultMinNodes.
	MinNodes int
	// MinStatements is the length a run must reach, in statements. Zero
	// means DefaultMinStatements.
	MinStatements int
	// Tests includes _test.go files.
	Tests bool
	// Exempt are patterns of packages and files not read: an import path
	// pattern ("..." matches any string) or a file base name glob.
	Exempt []string
}

// Validate reports a negative minimum and a malformed exemption.
func (c Config) Validate() error {
	switch {
	case c.MinNodes < 0:
		return fmt.Errorf("min-nodes: %d is negative", c.MinNodes)
	case c.MinStatements < 0:
		return fmt.Errorf("min-statements: %d is negative", c.MinStatements)
	}
	for i, p := range c.Exempt {
		if strings.TrimSpace(p) == "" {
			return fmt.Errorf("exempt[%d]: empty pattern", i)
		}
		if _, err := path.Match(p, ""); err != nil {
			return fmt.Errorf("exempt[%d]: %q: %w", i, p, err)
		}
	}
	return nil
}

// Includes reports whether the file named filename, of the package at
// pkgPath, is read. A pattern matching a package matches its external test
// package too.
func (c Config) Includes(pkgPath, filename string) bool {
	name := filepath.Base(filename)
	if !c.Tests && strings.HasSuffix(name, "_test.go") {
		return false
	}
	pkgPath = strings.TrimSuffix(pkgPath, "_test")
	for _, p := range c.Exempt {
		if ok, _ := path.Match(p, name); ok || pkgpattern.Match(p, pkgPath) {
			return false
		}
	}
	return true
}

// BindFlags registers c's fields as flags of fs.
func BindFlags(fs *flag.FlagSet, c *Config) {
	fs.IntVar(&c.MinNodes, "min-nodes", 0, "size a copied run must reach, in syntax nodes (0: "+strconv.Itoa(DefaultMinNodes)+")")
	fs.IntVar(&c.MinStatements, "min-statements", 0, "length a copied run must reach, in statements (0: "+strconv.Itoa(DefaultMinStatements)+")")
	fs.BoolVar(&c.Tests, "tests", false, "read _test.go files too")
	fs.Var(listflag.List{Items: &c.Exempt}, "exempt", `comma-separated patterns not read: an import path pattern ("..." matches anything) or a file name glob`)
}

func (c Config) minNodes() int { return cmp.Or(c.MinNodes, DefaultMinNodes) }

func (c Config) minStatements() int { return cmp.Or(c.MinStatements, DefaultMinStatements) }
