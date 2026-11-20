package forward

import (
	"encoding/binary"
	"strconv"

	"forwardapi"
)

type Conn struct {
	w    *Conn
	open bool
}

// Fail is reported though it has a doc comment: a doc does not justify a
// second name.
func (c *Conn) Fail(err error) error { return c.fail(err) } // want `twin-wrapper: Conn.Fail only calls Conn.fail with its own parameters: export the one function instead`

func (c *Conn) fail(err error) error {
	c.open = false
	return err
}

func (c *Conn) Close() { c.close() } // want `twin-wrapper: Conn.Close only calls Conn.close`

func (c *Conn) close() { c.open = false }

// ResolveUncached keeps its camel case after the first letter.
func ResolveUncached(schema, name string) (int, error) { return resolveUncached(schema, name) } // want `twin-wrapper: ResolveUncached only calls resolveUncached`

func resolveUncached(schema, name string) (int, error) { return len(schema + name), nil }

func Join(sep string, parts ...string) string { return join(sep, parts...) } // want `twin-wrapper: Join only calls join`

func join(sep string, parts ...string) string { return sep + parts[0] }

func IsReservedRelation(id int) bool { return IsReserved(id) } // want `alias: IsReservedRelation only calls IsReserved, of the same signature, with its own parameters: call the original; a second name`

func IsReserved(id int) bool { return id < 100 }

type Kind int

type Type struct{ k Kind }

func ty(k Kind) Type { return TypeOf(k) } // want `alias: ty only calls TypeOf`

func TypeOf(k Kind) Type { return Type{k: k} }

// A twin pair of different case is an alias when the wrapper is
// unexported: there is nothing to export.
func lower(n int) int { return Lower(n) } // want `alias: lower only calls Lower`

func Lower(n int) int { return n }

type Server struct{ slots map[string]int }

func (s *Server) releaseConn(id string) { s.releaseSlot(id) } // want `alias: Server.releaseConn only calls Server.releaseSlot`

func (s *Server) releaseSlot(id string) { delete(s.slots, id) }

func Pair[T any](a, b T) []T { return pair(a, b) } // want `twin-wrapper: Pair only calls pair`

func pair[U any](a, b U) []U { return []U{a, b} }

type Set[T comparable] struct{ m map[T]bool }

func (s *Set[T]) Add(v T) { s.add(v) } // want `twin-wrapper: Set.Add only calls Set.add`

func (s *Set[T]) add(v T) { s.m[v] = true }

// Near misses: none is reported.

// Different signature: the result type differs.
func Wide(n int) error { return wide(n) }

type wideErr struct{}

func (*wideErr) Error() string { return "" }

func wide(n int) *wideErr { return nil }

// A parameter type differs, so the call converts implicitly.
func Name(s fmtStringer) string { return name(s) }

type fmtStringer interface{ String() string }

type named struct{}

func (named) String() string { return "" }

func name(s any) string { return "" }

// Reordered arguments.
func Sub(a, b int) int { return sub(b, a) }

func sub(a, b int) int { return a - b }

// A conversion of an argument.
func Kinded(k Kind) Type { return TypeOf(Kind(int(k))) }

// A constant in place of a parameter.
func Base(s string, b int) (int64, error) { return base(s, 10) }

func base(s string, b int) (int64, error) { return strconv.ParseInt(s, b, 64) }

// A missing argument and an extra one.
func Short(a, b int) int { return short(a) }

func short(a int) int { return a }

func Long(a int) int { return long(a, a) }

func long(a, b int) int { return a + b }

// A variadic parameter not spread, and a slice spread into a variadic one.
func Pack(xs ...any) int { return pack(xs) }

func pack(xs ...any) int { return len(xs) }

func Spread(xs []int) int { return spread(xs...) }

func spread(xs ...int) int { return len(xs) }

// Two results passed as two arguments.
func Both(a, b int) int { return both(pairOf(a, b)) }

func both(a, b int) int { return a + b }

func pairOf(a, b int) (int, int) { return a, b }

// A method expression names the receiver as an argument.
func (c Conn) Check(err error) error { return Conn.check(c, err) }

func (Conn) check(err error) error { return err }

// The call, then another statement.
func Log(s string) {
	log(s)
	log(s)
}

func log(s string) {}

// Two statements.
func Two(n int) int {
	n++
	return two(n)
}

func two(n int) int { return n }

// A call into another package is not an alias.
func appendVarint(b []byte, v int64) []byte { return binary.AppendVarint(b, v) }

// A method calling a function, or a field's method, or another value's.
func (c *Conn) Open(err error) error { return open(err) }

func open(err error) error { return err }

func (c *Conn) Flush(err error) error { return c.w.flush(err) }

func (c *Conn) flush(err error) error { return err }

func (c *Conn) Reset(other *Conn, err error) error { return other.reset(err) }

func (c *Conn) reset(err error) error { return err }

// A function calls a method value's, through no receiver of its own.
func Ready(c *Conn) bool { return c.ready() }

func (c *Conn) ready() bool { return c.open }

// A method promoted from an embedded field.
type Outer struct{ *Conn }

func (o Outer) Shut() { o.Close() }

// A self call: the callee is recursive.
func Loop(n int) int { return Loop(n) }

// Explicit instantiation.
func Inst(a, b int) []int { return pair[int](a, b) }

// A twin through a func-typed variable is no function.
var hook = func(n int) int { return n }

func Hook(n int) int { return hook(n) }

// Used as values: the named function may be needed for its identity.
func Visit(n int) int { return visit(n) }

func visit(n int) int { return n }

func Each(n int) int { return each(n) }

func each(n int) int { return n }

func (c *Conn) Done() { c.done() }

func (c *Conn) done() {}

type table struct{ fn func(int) int }

var visitors = []func(int) int{Visit}

func register(c *Conn) {
	_ = table{fn: Each}
	f := c.Done
	f()
}

// Methods an interface fixes, here or in an import.
type Filter struct{ in int }

func (f *Filter) Resume(p *forwardapi.Parts) error { return f.resume(p) }

func (f *Filter) resume(p *forwardapi.Parts) error { return nil }

type Project struct{}

func (p *Project) Resume(q *forwardapi.Parts) error { return p.resumedInput(q) }

func (p *Project) resumedInput(q *forwardapi.Parts) error { return nil }

type AggState struct{ n int }

type CountStar struct{}

func (c CountStar) Merge(into *AggState, from AggState) error { return c.merge(into, from) }

func (CountStar) merge(into *AggState, from AggState) error { return nil }

type gauge struct{}

func (g gauge) Inc(n int) { g.inc(n) }

func (gauge) inc(int) {}

// An interface of this package fixes Lookup.
type looker interface{ Lookup(k string) int }

type cache struct{}

func (c cache) Lookup(k string) int { return c.lookup(k) }

func (cache) lookup(string) int { return 0 }

// Of the name but not the signature: still reported.
type Meter struct{}

func (m Meter) Resume(p *forwardapi.Parts, n int) error { return m.resume(p, n) } // want `twin-wrapper: Meter.Resume only calls Meter.resume`

func (Meter) resume(*forwardapi.Parts, int) error { return nil }
