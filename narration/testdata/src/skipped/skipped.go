// Copyright 2026 The Authors. This header was changed to MIT; previously, it
// was BSD.
// NOTE: header one.
// NOTE: header two.

//go:build !never

// Package skipped holds narration where none is read. Copyright notice:
// this was changed to the MIT license after the refactor.
package skipped

//gosloplint:ignore narration:history this now uses a reason that narrates
//nolint:all // this function now uses the cache, but nolint is a directive
func Directive() {}

//nolint // the helper was moved here from util
func Bare() {}

var Lit = "this now uses a string, per review: NOTE: one, NOTE: two"
