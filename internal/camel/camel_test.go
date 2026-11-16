package camel

import (
	"slices"
	"testing"
)

func TestWords(t *testing.T) {
	cases := map[string][]string{
		"identity":      {"identity"},
		"valid":         {"valid"},
		"userID":        {"user", "ID"},
		"userIDs":       {"user", "IDs"},
		"IDs":           {"IDs"},
		"userIds":       {"user", "Ids"},
		"HTTPServer":    {"HTTP", "Server"},
		"order_id":      {"order", "id"},
		"_leading":      {"leading"},
		"req.sessionID": {"session", "ID"},
		"x":             {"x"},
		"":              nil,
	}
	for in, want := range cases {
		if got := Words(in); !slices.Equal(got, want) {
			t.Errorf("Words(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestHasWord(t *testing.T) {
	cases := []struct {
		name, word string
		want       bool
	}{
		{"userAge", "age", true},
		{"AGE", "age", true},
		{"image", "age", false},
		{"manager", "age", false},
		{"pageSize", "age", false},
		{"LegacyClientID", "legacy client", true},
		{"LegacyClientID", "LegacyClient", true},
		{"legacyServerClient", "legacy client", false},
		{"legacy", "legacy client", false},
		{"x", "", false},
	}
	for _, tc := range cases {
		if got := HasWord(tc.name, tc.word); got != tc.want {
			t.Errorf("HasWord(%q, %q) = %v, want %v", tc.name, tc.word, got, tc.want)
		}
	}
}
