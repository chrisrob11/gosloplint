// Package api returns containers of another module's interface, some of
// which that module's API dictates.
package api

import "example.com/cli"

// Shape is this package's own interface.
type Shape interface{ Area() int }

type square struct{}

func (square) Area() int { return 1 }

type circle struct{}

func (circle) Area() int { return 2 }

// Mixed holds two concrete flag types: no struct replaces the []cli.Flag.
func Mixed(def string) []cli.Flag {
	return []cli.Flag{&cli.StringFlag{Name: def}, &cli.BoolFlag{Name: "dry-run"}, stringFlag("x")}
}

func stringFlag(name string) *cli.StringFlag { return &cli.StringFlag{Name: name} }

// Appended builds its mix by append and by index.
func Appended() []cli.Flag {
	out := []cli.Flag{}
	out = append(out, &cli.StringFlag{})
	out = append(out, &cli.BoolFlag{})
	return out
}

// Indexed builds a map of two kinds by index assignment.
func Indexed() map[string]cli.Flag {
	m := map[string]cli.Flag{}
	m["a"] = &cli.StringFlag{}
	m["b"] = &cli.BoolFlag{}
	return m
}

// ToField has one concrete type, but its call is passed straight to
// cli.Command.Flags.
func ToField() []cli.Flag { return []cli.Flag{&cli.StringFlag{}} }

// ToAssigned is passed straight to the field by assignment.
func ToAssigned() []cli.Flag { return []cli.Flag{&cli.StringFlag{}} }

// ToParam is passed straight to cli.Register's parameter.
func ToParam() []cli.Flag { return []cli.Flag{&cli.StringFlag{}} }

// ToSpread is spread into cli.Add's variadic parameter.
func ToSpread() []cli.Flag { return []cli.Flag{&cli.StringFlag{}} }

func use() {
	_ = cli.Command{Name: "x", Flags: ToField()}
	var c cli.Command
	c.Flags = ToAssigned()
	cli.Register("x", ToParam())
	cli.Add(ToSpread()...)
	_ = append(ToLocal(), nil)
	local(ToLocal())
}

func local([]cli.Flag) {}

// ToLocal holds one concrete type and reaches the API only through append
// or this package: it can return []*cli.StringFlag.
func ToLocal() []cli.Flag { return []cli.Flag{&cli.StringFlag{}} } // want `returns-interface: ToLocal returns \[\]cli.Flag, which holds an interface`

// Single returns a lone cli.Flag, not a container: it is reported however
// many types it may hold.
func Single(b bool) cli.Flag { // want `returns-interface: Single returns cli.Flag, an interface type`
	if b {
		return &cli.StringFlag{}
	}
	return &cli.BoolFlag{}
}

// SingleOne returns one concrete type behind a third-party interface.
func SingleOne() cli.Flag { return &cli.StringFlag{} } // want `returns-interface: SingleOne` `returns-one-concrete: SingleOne .* return \*cli.StringFlag\.`

// Nils puts nil and one type: one concrete type.
func Nils() []cli.Flag { return []cli.Flag{nil, &cli.StringFlag{}, &cli.StringFlag{}} } // want `returns-interface: Nils`

// Local mixes two types of this package's own interface: the package can
// change it, so it is reported.
func Local() []Shape { return []Shape{square{}, circle{}} } // want `returns-interface: Local`

// Closure mixes its types inside a func literal, which is not counted.
func Closure() []cli.Flag { // want `returns-interface: Closure`
	f := func() []cli.Flag { return []cli.Flag{&cli.StringFlag{}, &cli.BoolFlag{}} }
	_ = f
	return []cli.Flag{&cli.StringFlag{}}
}

// Grid nests the interface a level down: the outer element is a slice.
func Grid() [][]cli.Flag { // want `returns-interface: Grid`
	return [][]cli.Flag{{&cli.StringFlag{}, &cli.BoolFlag{}}}
}
