// Package rec is a recursion package in the tests.
package rec

import "strings"

type node struct{ kids []*node }

func walk(n *node) int { // want `unguarded-recursion: walk recurses with no depth bound`
	s := 1
	for _, k := range n.kids {
		s += walk(k)
	}
	return s
}

func walkBounded(n *node, depth int) int {
	if depth > 100 {
		return 0
	}
	s := 1
	for _, k := range n.kids {
		s += walkBounded(k, depth+1)
	}
	return s
}

func even(n int) bool { // want `unguarded-recursion: even,odd recurse`
	if n == 0 {
		return true
	}
	return odd(n - 1)
}

func odd(n int) bool {
	if n == 0 {
		return false
	}
	return even(n - 1)
}

func (n *node) size() int { // want `unguarded-recursion: node.size recurses`
	t := 1
	for _, k := range n.kids {
		t += k.size()
	}
	return t
}

func (n *node) flat() int { return len(n.kids) }

// sizer is called through, so leaf.count reaches itself via the interface.
type sizer interface{ count() int }

type leaf struct{ next sizer }

func (l leaf) count() int { // want `unguarded-recursion: leaf.count recurses`
	if l.next == nil {
		return 1
	}
	return 1 + l.next.count()
}

// other implements sizer too, so leaf.count's call reaches other.count, but
// other.count calls nothing back: it is not recursive.
type other struct{}

func (other) count() int { return strings.Count("aa", "a") }

func closure(n *node) int { // want `unguarded-recursion: closure recurses`
	f := func() int { return closure(n.kids[0]) }
	return f()
}

func generic[T any](xs []T) int { // want `unguarded-recursion: generic recurses`
	if len(xs) == 0 {
		return 0
	}
	return 1 + generic(xs[1:])
}

// callsOnly calls walk but is not in its cycle.
func callsOnly(n *node) int { return walk(n) }
