// Package assert is a stub of testify's assert, enough to type-check the
// fixtures.
package assert

type TestingT interface{ Errorf(string, ...any) }

func Equal(t TestingT, expected, actual any, msgAndArgs ...any) bool       { return true }
func EqualValues(t TestingT, expected, actual any, msgAndArgs ...any) bool { return true }
func NotEqual(t TestingT, expected, actual any, msgAndArgs ...any) bool    { return true }
func True(t TestingT, value bool, msgAndArgs ...any) bool                  { return true }
func False(t TestingT, value bool, msgAndArgs ...any) bool                 { return true }
func Nil(t TestingT, object any, msgAndArgs ...any) bool                   { return true }
func NoError(t TestingT, err error, msgAndArgs ...any) bool                { return true }
func NotNil(t TestingT, object any, msgAndArgs ...any) bool                { return true }
func NotEmpty(t TestingT, object any, msgAndArgs ...any) bool              { return true }
func Len(t TestingT, object any, length int, msgAndArgs ...any) bool       { return true }

type Assertions struct{ t TestingT }

func New(t TestingT) *Assertions { return &Assertions{t} }

func (a *Assertions) True(value bool, msgAndArgs ...any) bool   { return true }
func (a *Assertions) NoError(err error, msgAndArgs ...any) bool { return true }
