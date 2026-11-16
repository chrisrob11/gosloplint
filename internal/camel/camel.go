// Package camel splits Go identifiers into their camelCase and snake_case
// words, so a check can match a word without matching every identifier that
// merely contains its letters: "valid" ends in "id" but is one word.
package camel

import "strings"

// Words splits name into words. A package or receiver qualifier ("req.") is
// dropped first. Acronym runs stay whole ("HTTPServer" is HTTP, Server), and a
// plural acronym keeps its s ("userIDs" is user, IDs).
func Words(name string) []string {
	if i := strings.LastIndex(name, "."); i >= 0 {
		name = name[i+1:]
	}
	var words []string
	runes := []rune(name)
	start := 0
	for i := 1; i < len(runes); i++ {
		prev, cur := runes[i-1], runes[i]
		boundary := false
		switch {
		case isLower(prev) && isUpper(cur):
			boundary = true
		case isUpper(prev) && isUpper(cur) && i+1 < len(runes) && isLower(runes[i+1]):
			boundary = !pluralAcronymTail(runes, i)
		case cur == '_':
			boundary = true
		}
		if boundary {
			if w := strings.Trim(string(runes[start:i]), "_"); w != "" {
				words = append(words, w)
			}
			start = i
		}
	}
	if w := strings.Trim(string(runes[start:]), "_"); w != "" {
		words = append(words, w)
	}
	return words
}

// HasWord reports whether the words of word appear consecutively among the
// words of name, compared case-insensitively. "identity" does not have the
// word "entity"; "LegacyClientID" has the words "legacy client".
func HasWord(name, word string) bool {
	want := Words(strings.ReplaceAll(word, " ", "_"))
	if len(want) == 0 {
		return false
	}
	got := Words(name)
	for i := 0; i+len(want) <= len(got); i++ {
		matched := true
		for j := range want {
			if !strings.EqualFold(got[i+j], want[j]) {
				matched = false
				break
			}
		}
		if matched {
			return true
		}
	}
	return false
}

// pluralAcronymTail reports whether runes[i] is the last letter of an acronym
// followed by a plural s that ends the name or a word: the D in "IDs".
func pluralAcronymTail(runes []rune, i int) bool {
	if i+1 >= len(runes) || runes[i+1] != 's' {
		return false
	}
	return i+2 == len(runes) || isUpper(runes[i+2])
}

func isLower(r rune) bool { return r >= 'a' && r <= 'z' }
func isUpper(r rune) bool { return r >= 'A' && r <= 'Z' }
