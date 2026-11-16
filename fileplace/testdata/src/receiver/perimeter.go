package shape

// perimeter.go is perimeter's file, so shape's edge does not belong here.
type perimeter int

func (s *shape[T]) edge() perimeter { return 0 } // want `receiver: method shape.edge is outside shape.go, the file that declares shape \(its files hold \d+ lines, at most 1500, so no concern files\): move it into shape.go`
