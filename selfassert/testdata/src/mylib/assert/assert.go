// Package assert is a local package that shares testify's name but is not
// testify: its Equal is not an assertion selfassert knows.
package assert

func Equal(_ any, x, y any) bool { return x == y }
