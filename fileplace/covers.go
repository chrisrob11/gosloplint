package fileplace

import (
	"strings"

	"github.com/chrisrob11/gosloplint/internal/camel"
)

// platforms are the GOOS and GOARCH values a file name may end in, and
// unix and other, the names a file built for a group of platforms takes
// beside them (bindcap_other.go builds where bindcap_linux.go does not).
var platforms = map[string]bool{
	"unix": true, "other": true,
	"aix": true, "android": true, "darwin": true, "dragonfly": true, "freebsd": true, "hurd": true,
	"illumos": true, "ios": true, "js": true, "linux": true, "netbsd": true, "openbsd": true,
	"plan9": true, "solaris": true, "wasip1": true, "windows": true, "zos": true,
	"386": true, "amd64": true, "arm": true, "arm64": true, "loong64": true, "mips": true,
	"mipsle": true, "mips64": true, "mips64le": true, "ppc64": true, "ppc64le": true,
	"riscv64": true, "s390x": true, "wasm": true,
}

// dropPlatform drops up to two trailing parts of platforms: foo_linux_amd64
// is foo.
func dropPlatform(s string) string {
	for range 2 {
		i := strings.LastIndexByte(s, '_')
		if i <= 0 || !platforms[s[i+1:]] {
			break
		}
		s = s[:i]
	}
	return s
}

// norm is a name as a file name spells it: lower case, no underscores.
func norm(s string) string { return strings.ToLower(strings.ReplaceAll(s, "_", "")) }

// variants are the names a file stem may be read as: the whole stem, and
// each trailing run of its parts, so expr_call.go is read as call too (a
// family prefix, then the name). Each is norm'd.
func variants(stem string) []string {
	parts := strings.Split(stem, "_")
	out := make([]string, 0, len(parts))
	for k := range parts {
		out = append(out, norm(strings.Join(parts[k:], "")))
	}
	return out
}

// covers reports whether s, a norm'd stem, spells a run of name's camelCase
// words, or all of them when all is set. A piece of s spells a word whole,
// or abbreviates it (three or more of its letters in order, from its first:
// ctx for context), and the last piece may be the word's plural. A word of
// one or two letters past the first may be skipped: conflictrange.go holds
// ConflictOnRange, and role.go RoleID.
func covers(s, name string, all bool) bool {
	words := camel.Words(name)
	for i, w := range words {
		words[i] = strings.ToLower(w)
	}
	starts := len(words)
	if all {
		starts = min(1, len(words))
	}
	type state struct{ pos, j int }
	for j0 := range starts {
		todo := []state{{0, j0}}
		seen := map[state]bool{}
		for len(todo) > 0 {
			st := todo[len(todo)-1]
			todo = todo[:len(todo)-1]
			if st.pos == len(s) && st.j > j0 && (!all || st.j == len(words)) {
				return true
			}
			if st.j == len(words) || seen[st] {
				continue
			}
			seen[st] = true
			if st.j > j0 && len(words[st.j]) <= 2 {
				todo = append(todo, state{st.pos, st.j + 1})
			}
			for k := st.pos + 1; k <= len(s); k++ {
				if spellsWord(s[st.pos:k], words[st.j], k == len(s)) {
					todo = append(todo, state{k, st.j + 1})
				}
			}
		}
	}
	return false
}

// abbreviations are short forms that share no letters in order with the
// word they stand for, or too few: a file may use either for the other.
var abbreviations = map[string]string{"txn": "transaction", "tx": "transaction", "err": "error", "ctx": "context"}

// spellsWord reports whether piece is w, an abbreviation of it, or, when
// last, its plural.
func spellsWord(piece, w string, last bool) bool {
	for _, x := range []string{w, abbreviations[w]} {
		if x != "" && (piece == x || last && plural(piece, x)) {
			return true
		}
	}
	if abbreviations[piece] == w {
		return true
	}
	if one, ok := strings.CutSuffix(piece, "s"); ok && last && abbreviates(one, w) {
		return true
	}
	return abbreviates(piece, w)
}

// plural reports whether p is w's plural: ws, wes, or w's y as ies.
func plural(p, w string) bool {
	return p == w+"s" || p == w+"es" || strings.HasSuffix(w, "y") && p == w[:len(w)-1]+"ies"
}

// abbreviates reports whether piece is three or more of w's letters in
// order, from its first: ctx for context.
func abbreviates(piece, w string) bool {
	if len(piece) < 3 || len(piece) >= len(w) || piece[0] != w[0] {
		return false
	}
	i := 0
	for j := 0; j < len(w) && i < len(piece); j++ {
		if w[j] == piece[i] {
			i++
		}
	}
	return i == len(piece)
}
