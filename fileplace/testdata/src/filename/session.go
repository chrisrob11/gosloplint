package conn // want `filename: session.go is named for nothing it declares`

// session.go holds lookup and state, neither named session.
type state int

func lookup() state { return 0 }
