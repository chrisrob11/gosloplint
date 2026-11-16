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
