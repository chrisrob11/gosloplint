package clone

import "fmt"

// Clone is two runs of statements with the same shape. A is the earlier
// occurrence, by file name and then line; B, the later, is where a clone is
// reported.
type Clone struct {
	A, B Span
}

// key names the clone by its two files and declarations, never by line, so
// the runs one pair of declarations share fold into one clone.
func (c Clone) key() string {
	return c.A.File + "\x00" + c.A.Decl + "\x00" + c.B.File + "\x00" + c.B.Decl
}

// Message is the finding reported at c.B. file is how to name c.A's file:
// its base name within one package, a relative path across packages.
func (c Clone) Message(file string) string {
	return fmt.Sprintf("lines %d-%d duplicate lines %d-%d of %s (%s): move the shared statements into one function both call",
		c.B.Start, c.B.End, c.A.Start, c.A.End, file, c.A.Decl)
}
