package typefile

type target interface{ run() }

// remote is a target with methods, in target.go.
type remote struct{} // want `typefile: type remote has 1 method\(s\) but is declared in target.go: move it and its methods into remote.go`

func (remote) run() {}
