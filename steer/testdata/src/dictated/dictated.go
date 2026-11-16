// Package dictated holds methods whose signatures an interface fixes.
package dictated

import (
	"container/heap"
	"database/sql/driver"

	"example.com/hooks"
)

// queue implements heap.Interface, whose Pop returns any: heap fixes it, so
// nothing here is reported.
type queue []int

func (q queue) Len() int           { return len(q) }
func (q queue) Less(i, j int) bool { return q[i] < q[j] }
func (q queue) Swap(i, j int)      { q[i], q[j] = q[j], q[i] }
func (q *queue) Push(x any)        { *q = append(*q, x.(int)) }
func (q *queue) Pop() any {
	old := *q
	x := old[len(old)-1]
	*q = old[:len(old)-1]
	return x
}

var _ heap.Interface = (*queue)(nil)

// money implements driver.Valuer, whose result is driver.Value, an alias of
// any.
type money int

func (m money) Value() (driver.Value, error) { return int64(m), nil }

// Source is this package's own contract: it is reported, and so is each
// implementation, which says the interface fixes it.
type Source interface {
	Open() Reader // want `returns-interface: Source.Open`
}

// Reader is a package interface.
type Reader interface{ Read() int }

type file struct{}

func (file) Read() int { return 0 }

type disk struct{}

func (disk) Open() Reader { return file{} } // want `returns-interface: disk.Open returns Reader, an interface type: .*; the interface Source fixes this signature, so change it there first` `returns-one-concrete: disk.Open .*; the interface Source fixes`

// loose has a Pop method returning any, but implements no interface with
// one, so it is reported.
type loose struct{}

func (loose) Pop() any { return nil } // want `returns-any: loose.Pop`

// run, and the literals below, take their type from hooks.Hook and
// hooks.Register: hooks fixes the signature, so none is reported.
func run() any { return nil }

func register() {
	_ = hooks.Hook{Run: run}
	var h hooks.Hook
	h.Run = func() any { return nil }
	hooks.Register(func() any { return 1 })
	_ = h
}

// own is called, never passed, so its signature is its own.
func own() any { return nil } // want `returns-any: own`

var _ = own()

// The literal is called where it stands, so its type is its own.
var _ = func() any { return nil }() // want `returns-any: _.func`
