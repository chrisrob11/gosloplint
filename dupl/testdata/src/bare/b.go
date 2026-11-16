package fixture

//gosloplint:dupl-ok
func second(in input) (int, error) {
	a := in.x + in.y*2 // want `lines 5-17 duplicate lines 8-20 of a.go \(first\)`
	if a > 10 {
		return a, nil
	}
	b := in.z + a
	for i := 0; i < b; i++ {
		a += i
	}
	if err := check(a); err != nil {
		return 0, err
	}
	b += a * 3
	return a + b, nil
}
