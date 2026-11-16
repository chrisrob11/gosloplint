package a

import "context"

// Not reported: a test is where a context starts.
var _ = context.Background()

type fixture struct{ ctx context.Context }
