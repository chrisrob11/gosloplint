package forward

// An export_test.go hands an unexported function to the external tests
// under its exported name: not a twin.
func Resolve(schema, name string) (int, error) { return resolveUncached(schema, name) }

func ResolveUncachedForTest(schema, name string) (int, error) { return resolveUncached(schema, name) }

func JoinT(sep string, parts ...string) string { return join(sep, parts...) }

// A test helper renaming for readability.
func reserved(id int) bool { return IsReserved(id) }

func (c *Conn) Ready() bool { return c.ready() }

var probes = []func(int) int{probe, Probe}
