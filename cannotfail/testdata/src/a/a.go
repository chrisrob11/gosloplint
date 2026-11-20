package a

import (
	"errors"
	"testing"
)

func query() []string { return nil }

func load(string) (int, error) { return 0, errors.New("x") }

type Config struct {
	Name string
	Port int
	Tags []string
}

func (c *Config) Normalize() { c.Port = 80 }

// checkOK is a helper outside the test files that fails the test.
func checkOK(tb testing.TB, ok bool) {
	if !ok {
		tb.Error("not ok")
	}
}

// MustSave fails by panicking, which this fixture's body leaves out: a
// Must function outside the test files counts as able to fail by its name.
func MustSave(string) {}

// Mustang is not a Must function.
func Mustang() []string { return nil }

// Getenv is not os.Getenv.
func Getenv(string) string { return "" }
