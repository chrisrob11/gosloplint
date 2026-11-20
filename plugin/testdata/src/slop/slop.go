// Package slop holds one finding per configurable setting. It carries no want
// comments: the plugin tests count the findings each configuration produces.
package slop

// LegacyClient is reported by vocab only when "legacy client" is banned.
type LegacyClient struct{}

// byUser is reported by emptyid under the default suffixes.
func byUser(userID string) error {
	if userID == "" {
		return nil
	}
	return nil
}

// byTable is reported by emptyid only when "ref" is a suffix.
func byTable(tableRef string) error {
	if tableRef == "" {
		return nil
	}
	return nil
}

// walk is reported by steer only when recursion-packages names slop.
func walk(n int) int {
	if n == 0 {
		return 0
	}
	return walk(n - 1)
}

// nine is reported by comment under the default func limit of 8 lines.
// two
// three
// four
// five
// six
// seven
// eight
// nine
func nine() {}

// second is reported by dupl as a copy of first under the default minimum
// size.
func first(xs []int) (int, int) {
	lo, hi := xs[0], xs[0]
	for _, x := range xs {
		if x < lo {
			lo = x
		}
		if x > hi {
			hi = x
		}
	}
	hi -= lo / 2
	span := hi - lo
	if span > 100 {
		return lo, 100
	}
	if span < 0 {
		return hi, -span
	}
	return lo, span
}

func second(ys []int) (int, int) {
	a, b := ys[1], ys[1]
	for _, y := range ys {
		if y < a {
			a = y
		}
		if y > b {
			b = y
		}
	}
	b -= a / 2
	width := b - a
	if width > 50 {
		return a, 50
	}
	if width < 0 {
		return b, -width
	}
	return a, width
}

// cached is reported by narration unless history is disabled: it now uses a
// map.
func cached() {}
