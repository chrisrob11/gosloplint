package shape

// more is fake's, a test type, in another test file: allowed.
func (fake) more() {}

// only is a test's method on shape.go's type.
func (shape[T]) only() {} // want `receiver: method shape.only is outside shape.go, the file that declares shape: a test adds a method to shape.go's type: make only a function in the test`
