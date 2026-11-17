// Package check fails a test through a handler registered elsewhere, so its
// functions take no testing value.
package check

func That(ok bool) {}

type Suite struct{}

func (Suite) Holds(ok bool) {}
