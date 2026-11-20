//go:build !darwin && !linux

package forward

func platform() string { return "other" }
