// Package shadow declares its own any, which is a struct.
package shadow

type any struct{}

// F returns the package's any, a struct.
func F() any { return any{} }
