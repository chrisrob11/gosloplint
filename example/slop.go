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
