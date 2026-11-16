// Package custom is analyzed with -suffixes=ref,Prefix.
package custom

func ref(tableRef string) error {
	if tableRef == "" { // want `guard on empty tableRef exits silently`
		return nil
	}
	return nil
}

func prefix(pathPrefix string) error {
	if pathPrefix == "" { // want `guard on empty pathPrefix exits silently`
		return nil
	}
	return nil
}

// Not reported: the flag replaces the default list rather than adding to it.
func id(userID string) error {
	if userID == "" {
		return nil
	}
	return nil
}
