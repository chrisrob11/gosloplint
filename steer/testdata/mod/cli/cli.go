// Package cli is another module, with an interface its API takes slices of.
package cli

// Flag is a command-line flag.
type Flag interface{ Names() []string }

// StringFlag is one Flag.
type StringFlag struct{}

// Names implements Flag.
func (*StringFlag) Names() []string { return nil }

// BoolFlag is another.
type BoolFlag struct{}

// Names implements Flag.
func (*BoolFlag) Names() []string { return nil }
