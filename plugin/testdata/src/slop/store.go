package slop

// cache is reported by fileplace only when typefile is enabled: store.go is
// not named for it.
type cache struct{}

func (cache) get() {}
