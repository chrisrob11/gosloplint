// Package header recognises a file's header comments: a license or
// copyright notice is boilerplate, not prose about the code, so no check
// reads it as one.
package header

import (
	"go/ast"
	"strings"
)

// Is reports whether cg is a header: a block above f's package clause that
// is not its doc (a license header, a build constraint's note), or any
// block naming a copyright or an SPDX license.
func Is(f *ast.File, cg *ast.CommentGroup) bool {
	if cg != f.Doc && cg.End() < f.Package {
		return true
	}
	for _, c := range cg.List {
		low := strings.ToLower(c.Text)
		if strings.Contains(low, "copyright") || strings.Contains(low, "spdx-license-identifier") {
			return true
		}
	}
	return false
}
