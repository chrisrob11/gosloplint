package fixture

func second(in input) (int, error) {
	a := in.x - in.y*2
	if a > 10 {
		return a, nil
	}
	b := in.z - a
	for i := 0; i < b; i++ {
		a += i
	}
	if err := check(a); err != nil {
		return 0, err
	}
	b += a * 3
	return a - b, nil
}
