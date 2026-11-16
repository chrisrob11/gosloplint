package fixture

type pair struct{ x, y int }

func f(a, b any) int { return 0 }

// base, lits and nodes differ only in a literal's kind or a node's kind.
func base(s pair, v []int, k int) int {
	n := 0
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	n += f(s.x, 1)
	return n
}

func lits(s pair, v []int, k int) int {
	n := 0
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	n += f(s.x, "1")
	return n
}

func nodes(s pair, v []int, k int) int {
	n := 0
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	n += f(v[k], 1)
	return n
}
