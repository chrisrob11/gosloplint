// Package a is analyzed with -words=age,legacy client.
package a

type UserAge struct { // want `declares UserAge, which contains the banned word "age"`
	MaxAge int // want `declares MaxAge, which contains the banned word "age"`
}

type LegacyClientID string // want `declares LegacyClientID, which contains the banned word "legacy client"`

// AGE_LIMIT is matched case-insensitively.
const AGE_LIMIT = 120 // want `declares AGE_LIMIT`

// image, manager and pageSize contain the letters "age" but not the word, and
// legacy alone is not "legacy client". None is reported.
func f(image, manager string, pageSize int, legacy bool) UserAge { // want `uses UserAge`
	u := UserAge{MaxAge: AGE_LIMIT} // want `uses UserAge` `uses MaxAge` `uses AGE_LIMIT`
	_, _, _, _ = image, manager, pageSize, legacy
	return u
}
