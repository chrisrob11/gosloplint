package shape

// shape declares the type.
type shape[T any] struct{ v T }

func (s shape[T]) area() int { return 0 }

// pair is a second type with its method in its own file.
type pair [2]int

func (p pair) sum() int { return p[0] + p[1] }
