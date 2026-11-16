// Package slop is the end-to-end example for the golangci-lint plugin: each
// declaration is something one gosloplint analyzer reports.
package slop

import (
	"context"
	"errors"
)

// Store keeps its context in a field instead of taking one per call.
type Store struct {
	ctx context.Context
}

// LegacyClient uses a word the .golangci.yml bans through vocab.words.
type LegacyClient struct{}

// Load starts a fresh context instead of threading the caller's.
func (s *Store) Load(id string) error {
	return query(context.Background(), id)
}

// DeleteSessions reports success when the userID it needs was dropped.
func DeleteSessions(userID string) error {
	if userID == "" {
		return nil
	}
	return nil
}

// Drop is reported only because .golangci.yml adds "ref" to the suffixes.
func Drop(tableRef string) error {
	if tableRef == "" {
		return nil
	}
	return nil
}

func query(ctx context.Context, id string) error {
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if id == "" {
		return errors.New("query: empty id")
	}
	return nil
}

// Cache is the only implementation of Getter, so NewCache should return it.
type Getter interface{ Get(key string) string }

type memCache struct{}

func (memCache) Get(string) string { return "" }

// NewCache hides its one implementation behind an interface.
func NewCache() Getter { return memCache{} }

// describe narrates what the next line says.
func describe(n int) string {
	// Here we check whether n is zero.
	// If n is zero we return the word zero.
	// Otherwise we fall through to the general case.
	// The general case returns the word many.
	if n == 0 {
		return "zero"
	}
	return "many"
}

type node struct{ kids []*node }

// size is reported only because .golangci.yml sets steer.recursion-packages.
func size(n *node) int {
	total := 1
	for _, k := range n.kids {
		total += size(k)
	}
	return total
}
