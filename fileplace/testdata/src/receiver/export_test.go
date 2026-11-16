package shape

// Area exports area to the external tests, as export_test.go is for.
func (s shape[T]) Area() int { return s.area() }
