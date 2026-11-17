// Package onepkg is reported by fake for its name: the directive at the end
// of this doc covers the package clause, the line below it.
//
//gosloplint:ignore fake:one the package name is the fixture's
package onepkg

//gosloplint:ignore fake:one the line below is covered
var oneAbove int

var oneTrailing int //gosloplint:ignore fake the directive's own line is covered

//gosloplint:ignore fake two lines above covers nothing // want `ignore: //gosloplint:ignore fake suppresses no fake finding: delete it`

var oneFar int // want `one: oneFar`

//gosloplint:ignore	fake	a tab separates the reason too
var oneTabbed int

//gosloplint:ignore fake:two names another check // want `ignore: //gosloplint:ignore fake:two suppresses no fake finding`
var oneOther int // want `one: oneOther`

// A directive alone on its line in a body covers the statement below.
func helper() {
	//gosloplint:ignore fake:one the statement below
	oneLocal := 0
	_ = oneLocal
}

// oneFunc's doc directive covers its whole body.
//
//gosloplint:ignore fake the whole declaration
func oneFunc() {
	var oneInside, twoInside int
	_, _ = oneInside, twoInside
}

// A grouped declaration: the group's doc covers the group, a spec's doc
// covers the spec.
//
//gosloplint:ignore fake:two every two in the group
var (
	twoGrouped int
	//gosloplint:ignore fake:one this spec only
	oneSpec = struct {
		oneInner int
	}{}
	oneLoose int // want `one: oneLoose`
)

// want +1 `ignore: //gosloplint:ignore fake needs a reason after it`
//gosloplint:ignore fake
var oneNoReason int // want `one: oneNoReason`

//gosloplint:ignore nosuch whatever // want `ignore: //gosloplint:ignore names unknown analyzer "nosuch" \(known: fake, other, plain\)`

//gosloplint:ignore fake:three no such check // want `ignore: //gosloplint:ignore fake:three: fake has no check "three" \(checks: one, two\)`
var oneBadCheck int // want `one: oneBadCheck`

// want +1 `ignore: //gosloplint:ignore names no analyzer`
//gosloplint:ignore

//gosloplint:ignore other other is known but does not run here, so this is not stale

//gosloplint:ignored is another directive, not this one
var oneNotADirective int // want `one: oneNotADirective`

//gosloplint:ignore plain plain's directive is plain's to judge, not fake's

// A spec's doc covers the whole spec.
type (
	//gosloplint:ignore fake:one this spec only
	holder struct {
		oneField int
	}
	oneType int // want `one: oneType`
)
