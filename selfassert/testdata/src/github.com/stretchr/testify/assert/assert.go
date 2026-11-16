// Package assert is a stub of testify's assert, enough to type-check the
// fixtures.
package assert

type TestingT interface{ Errorf(string, ...any) }

func Equal(t TestingT, expected, actual any, msgAndArgs ...any) bool        { return true }
func Equalf(t TestingT, expected, actual any, msg string, args ...any) bool { return true }
func EqualValues(t TestingT, expected, actual any, msgAndArgs ...any) bool  { return true }
func Same(t TestingT, expected, actual any, msgAndArgs ...any) bool         { return true }
func Exactly(t TestingT, expected, actual any, msgAndArgs ...any) bool      { return true }
func NotEqual(t TestingT, expected, actual any, msgAndArgs ...any) bool     { return true }
func NotSame(t TestingT, expected, actual any, msgAndArgs ...any) bool      { return true }
func True(t TestingT, value bool, msgAndArgs ...any) bool                   { return true }
func NotEmpty(t TestingT, object any, msgAndArgs ...any) bool               { return true }
func Contains(t TestingT, s, contains any, msgAndArgs ...any) bool          { return true }

type Assertions struct{ t TestingT }

func New(t TestingT) *Assertions { return &Assertions{t} }

func (a *Assertions) Equal(expected, actual any, msgAndArgs ...any) bool    { return true }
func (a *Assertions) NotEqual(expected, actual any, msgAndArgs ...any) bool { return true }
