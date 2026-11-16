package split

type session int

func (*Conn) namespace() session { return 0 } // want `receiver: method Conn.namespace is outside conn.go`
