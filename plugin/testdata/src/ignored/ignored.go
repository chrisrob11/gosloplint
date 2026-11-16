// Package ignored suppresses one emptyid finding with a directive, and
// carries one directive that suppresses nothing.
package ignored

//gosloplint:ignore emptyid an empty ID means every user here
func byUser(userID string) error {
	if userID == "" {
		return nil
	}
	return nil
}

//gosloplint:ignore emptyid stale: nothing below is reported
func clean() {}
