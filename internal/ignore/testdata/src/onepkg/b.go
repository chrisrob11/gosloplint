package onepkg // want `one: onepkg`

// A directive covers lines of its own file only: a.go has one on lines 7
// and 8.



var oneOtherFile int // want `one: oneOtherFile`
