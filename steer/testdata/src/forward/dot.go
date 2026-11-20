package forward

import . "unicode/utf8"

// A dot import's function is another package's.
func runeLen(r rune) int { return RuneLen(r) }
