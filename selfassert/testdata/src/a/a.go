package a

func Database(a, b string) string { return a + b }

func columnarFamily(root string, s string) map[string]string {
	return map[string]string{"cq": root + s}
}

func encode(s string) string       { return s }
func decode(s string) string       { return s }
func recordPrefix(s string) string { return "R/" + s }
func normalize(string) string      { return "x" }
func fillA(map[string]int)         {}
func fillB(map[string]int)         {}
func fill(*string)                 {}
func newID() string                { return "" }

// Not reported: a self-comparison outside a test file is a different defect.
func same(x int) bool { return x == x }

var _ = same
