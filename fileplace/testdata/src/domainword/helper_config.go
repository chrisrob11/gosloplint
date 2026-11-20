package domainword

// HelperConfig is named with helper as a domain word, so helper_config.go
// is named for it.
type HelperConfig struct{}

// Ours reports nothing.
func (HelperConfig) Ours() bool { return false }
