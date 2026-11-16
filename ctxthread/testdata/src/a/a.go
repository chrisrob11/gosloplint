package a

import c "context" // an aliased import is still package context

var bg = c.Background() // want `context.Background creates a context from nothing`

type s struct{ ctx c.Context } // want `context.Context stored in a struct`

type embeds struct {
	c.Context // want `context.Context stored in a struct`
}

type two struct {
	a, b c.Context // want `context.Context stored in a struct` `context.Context stored in a struct`
}

type v struct{}

func (v) Value() int { return 0 }

type key struct{}

func use() {
	_ = bg // want `bg holds a context made from nothing`
	_ = bg // want `bg holds a context made from nothing`
}

func g(ctx c.Context, x v) {
	_ = ctx.Value(key{})                 // want `ctx.Value reads from the context as a value bag`
	_ = c.WithValue(ctx, key{}, 1)       // want `context.WithValue uses the context as a value bag`
	_ = c.TODO()                         // want `context.TODO creates a context from nothing`
	_ = x.Value()                        // a Value method that is not a context's
	_ = embeds{ctx}.Value(key{})         // want `ctx.Value reads from the context as a value bag`
	h := func(ctx v) { _ = ctx.Value() } // a parameter named ctx that is not a context
	_ = h
}

// context.Background() in a comment is not a call.

func shadow() {
	bg := 1 // a local named bg is a different variable
	_ = bg
}

// Threading the caller's context is the shape that is never reported.
func threaded(ctx c.Context) error {
	ctx, cancel := c.WithCancel(ctx)
	defer cancel()
	return ctx.Err()
}

var _ = s{}
var _ = two{}
var _ = use
var _ = g
var _ = shadow
