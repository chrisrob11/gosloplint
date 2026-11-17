// Package require is a stub of testify's require, enough to type-check the
// fixtures.
package require

type TestingT interface {
	Errorf(string, ...any)
	FailNow()
}

func Equal(t TestingT, expected, actual any, msgAndArgs ...any) {}
func NoError(t TestingT, err error, msgAndArgs ...any)          {}
func Len(t TestingT, object any, length int, msgAndArgs ...any) {}
func NotEmpty(t TestingT, object any, msgAndArgs ...any)        {}
