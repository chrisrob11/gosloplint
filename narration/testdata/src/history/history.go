// Package history holds comments that narrate a change; each is reported.
package history

// Load this now uses a map. // want `history: "this now" narrates how the code changed`
func Load() {}

// Save reads the row and writes it back. This function was changed to // want `"This function was changed to"`
// take a context.
func Save() {}

// Index keeps the keys sorted. The code was refactored for speed. // want `"was refactored"`
func Index() {}

// Scan walks the tree; it used to return a slice. // want `"it used to return"`
func Scan() {}

// Get now uses the cache before the store. // want `"now uses"`
func Get() {}

// Put stores the value. We replaced the mutex with a channel. // want `"We replaced"`
func Put() {}

// Del removes the key. Previously, it panicked on a missing key. // want `"Previously,"`
func Del() {}

// Previously the lookup went through the store. // want `"Previously the"`
func Lookup() {}

// Len counts the rows, simpler after the refactor. // want `"after the refactor" narrates`
func Len() {}

// Cap is the row limit, per review. // want `"per review"`
func Cap() {}

// Grow doubles the buffer, as requested in code review. // want `"requested in code review"`
func Grow() {}

// Trim drops trailing blanks; this addresses review feedback. // want `"addresses review"`
func Trim() {}

// Split cuts at the separator. This fixes the bug where an empty // want `"fixes the bug where"`
// input panicked.
func Split() {}

// Join was renamed from Concat. // want `"renamed from"`
func Join() {}

// Pad was moved here from the strings helper. // want `"moved here from"`
func Pad() {}

// Fold, formerly called Lower, folds case. // want `"formerly called"`
func Fold() {}

// Limit is 10; the bound was added in #123. // want `"added in #123"`
func Limit() {}

// Quote escapes the string (#456) changed to use strconv. // want `"#456\) changed"`
func Quote() {}

// Hash mixes the bytes; the old implementation was slower. // want `"old implementation"`
func Hash() {}

// Retry backs off. The sleep was added to fix a flaky test. // want `"added to fix"`
func Retry() {}

// Close no longer blocks: this no longer waits for readers. // want `"this no longer"`
func Close() {}

func body() {
	x := 1
	// we used to compute x twice here // want `"we used to"`
	_ = x
	/* the helper was switched to the fast path */ // want `"the helper was switched to"`
}

// Wrap is the first line of a block whose second line narrates:
// the handler was replaced by a closure. // want `"the handler was replaced by"`
func Wrap() {}

// Pick returns the first match, kept from review feedback. // want `"review feedback"`
func Pick() {}

// Parse reads "a, b" pairs; this function now uses a scanner. // want `"this function now"`
func Parse() {}

// Halve takes a "dangling quote and was renamed from Cut. // want `renamed from`
func Halve() {}
