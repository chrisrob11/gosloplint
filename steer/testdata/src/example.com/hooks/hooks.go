// Package hooks fixes the signatures of the callbacks it takes.
package hooks

// Hook takes a callback.
type Hook struct{ Run func() any } // want `returns-any: Hook.Run`

// Register takes a callback too.
func Register(func() any) {} // want `returns-any: Register.param0`
