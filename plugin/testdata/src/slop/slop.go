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
