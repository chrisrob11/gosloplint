// Package other has a T that is not testing.T.
package other

type T struct{}

func (*T) Fatal(...any) {}
