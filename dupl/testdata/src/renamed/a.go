package fixture

type input struct{ x, y, z int }

func check(int) error { return nil }

type calc struct{}

func (c *calc) first(in input) (int, error) {
	a := in.x + in.y*2
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
	return a + b + in.x, nil
}

func second(p input) (int, error) {
	total := p.y + p.z*7 // want `lines 26-38 duplicate lines 10-22 of a.go \(calc.first\)`
	if total > 99 {
		return total, nil
	}
	rest := p.x + total
	for k := 3; k < rest; k++ {
		total += k
	}
	if failure := check(total); failure != nil {
		return 7, failure
	}
	rest += total * 11
	return total + rest + p.x, nil
}
