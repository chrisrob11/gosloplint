// Package cli stands in for a third-party command-line library whose API
// takes slices of its own interface.
package cli

// Flag is a command-line flag; the package has several kinds.
type Flag interface{ Names() []string }

// StringFlag is one Flag.
type StringFlag struct{ Name string }

// Names implements Flag.
func (f *StringFlag) Names() []string { return []string{f.Name} }

// BoolFlag is another.
type BoolFlag struct{ Name string }

// Names implements Flag.
func (f *BoolFlag) Names() []string { return []string{f.Name} }

// Command takes its flags as a []Flag.
type Command struct {
	Name  string
	Flags []Flag
}

// Register takes flags as a parameter.
func Register(name string, flags []Flag) {}

// Add takes flags one at a time.
func Add(flags ...Flag) {}
