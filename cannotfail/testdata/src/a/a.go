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
