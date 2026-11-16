// Package main is where a context starts, so nothing in it is reported.
package main

import "context"

var root = context.Background()

func main() { _ = root }
