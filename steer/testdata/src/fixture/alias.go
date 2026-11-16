package fixture

import nd "example.com/other"

func Aliased() nd.Node { return nil } // want `returns-interface: Aliased returns nd.Node`
