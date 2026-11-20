// Package forwardapi declares interfaces forward's methods implement.
package forwardapi

type Parts struct{ N int }

// Resumer fixes Resume's signature for every operator.
type Resumer interface{ Resume(p *Parts) error }

// Merger is generic: a method matches it by name and arity.
type Merger[S any] interface{ Merge(into *S, from S) error }

// Gauge is embedded in Meter, whose method set holds Inc.
type Gauge interface{ Inc(n int) }

type Meter interface {
	Gauge
	Dec(n int)
}
