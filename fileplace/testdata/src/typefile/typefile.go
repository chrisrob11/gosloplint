package typefile

type T int

func (T) String() string { return "" }

// Other lives in the package's own file, where its types may.
type Other int

func (Other) M() {}
