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
