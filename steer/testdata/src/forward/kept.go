package forward

import _ "unsafe"

// Generic callee, inferred: reported, its instantiation is the signature.
func IsReservedID(id ID) bool { return isReserved(id) } // want `alias: IsReservedID only calls isReserved`

type ID int64

func isReserved[T ~int64](id T) bool { return id < 100 }

// Deprecated: a paragraph of its own keeps the old name for its callers.
func OldName(n int) int { return newName(n) }

func newName(n int) int { return n }

// Mentioning that something is Deprecated: mid-sentence is no deprecation.
func Stale(n int) int { return stale(n) } // want `twin-wrapper: Stale only calls stale`

func stale(n int) int { return n }

// The callee is named by a linkname directive.
func Linked(n int) int { return linked(n) }

//go:linkname linked runtime.forwardLinked
func linked(n int) int { return n }

// The wrapper is.
//
//go:linkname Exposed
func Exposed(n int) int { return exposed(n) }

func exposed(n int) int { return n }

//go:noinline
func Pinned(n int) int { return pinned(n) }

func pinned(n int) int { return n }

func Frame(n int) int { return frame(n) }

//go:noinline
func frame(n int) int { return n }

// A //go:nosplit directive is no reason.
//
//go:nosplit
func Split(n int) int { return split(n) } // want `twin-wrapper: Split only calls split`

func split(n int) int { return n }

// The callee implemented in assembly has no body.
func Sqrt(x float64) float64 { return sqrt(x) }

func sqrt(x float64) float64

// A recursive callee, and one used as a value.
func Walk(n int) int { return walk(n) }

func walk(n int) int {
	if n == 0 {
		return 0
	}
	return walk(n - 1)
}

func Fold(n int) int { return fold(n) }

func fold(n int) int { return n }

var folder = fold

// A callee calling others is not recursive.
func Chain(n int) int { return chain(n) } // want `twin-wrapper: Chain only calls chain`

func chain(n int) int { return newName(n) + 1 }

// A callee per platform: os_darwin.go, os_linux.go and os_other.go.
func Platform() string { return platform() }

// A callee in a file with a //go:build line and no other variant.
func Tagged() int { return tagged() }

// Used as a value only by a test: still reported.
func Probe(n int) int { return probe(n) } // want `twin-wrapper: Probe only calls probe`

func probe(n int) int { return n }
