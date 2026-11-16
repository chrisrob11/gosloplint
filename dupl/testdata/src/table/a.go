package fixture

type row struct {
	a, b, c, d int
	s          string
}

func one() []row {
	return []row{
		{a: 1, b: 1, c: 1, d: 1, s: "one1"},
		{a: 2, b: 2, c: 2, d: 2, s: "one2"},
		{a: 3, b: 3, c: 3, d: 3, s: "one3"},
		{a: 4, b: 4, c: 4, d: 4, s: "one4"},
		{a: 5, b: 5, c: 5, d: 5, s: "one5"},
		{a: 6, b: 6, c: 6, d: 6, s: "one6"},
		{a: 7, b: 7, c: 7, d: 7, s: "one7"},
		{a: 8, b: 8, c: 8, d: 8, s: "one8"},
		{a: 9, b: 9, c: 9, d: 9, s: "one9"},
		{a: 10, b: 10, c: 10, d: 10, s: "one10"},
		{a: 11, b: 11, c: 11, d: 11, s: "one11"},
		{a: 12, b: 12, c: 12, d: 12, s: "one12"},
	}
}

func two() []row {
	return []row{
		{a: 1, b: 1, c: 1, d: 1, s: "two1"},
		{a: 2, b: 2, c: 2, d: 2, s: "two2"},
		{a: 3, b: 3, c: 3, d: 3, s: "two3"},
		{a: 4, b: 4, c: 4, d: 4, s: "two4"},
		{a: 5, b: 5, c: 5, d: 5, s: "two5"},
		{a: 6, b: 6, c: 6, d: 6, s: "two6"},
		{a: 7, b: 7, c: 7, d: 7, s: "two7"},
		{a: 8, b: 8, c: 8, d: 8, s: "two8"},
		{a: 9, b: 9, c: 9, d: 9, s: "two9"},
		{a: 10, b: 10, c: 10, d: 10, s: "two10"},
		{a: 11, b: 11, c: 11, d: 11, s: "two11"},
		{a: 12, b: 12, c: 12, d: 12, s: "two12"},
	}
}
