package fixture

func second(in input) (int, error) {
	a := in.x + in.y*2 // want `lines 4-21 duplicate lines 8-25 of a.go \(first\)`
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
	for j := b; j > 0; j-- {
		a -= j
	}
	if a < 0 {
		a = -a
	}
	return a + b, nil
}
