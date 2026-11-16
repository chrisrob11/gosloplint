package shape

// session.go is named for a concern, not a type, so shape's methods may
// live here, beside a helper type named for neither.
type sessionKey int

func (s shape[T]) begin() sessionKey { return 0 }
