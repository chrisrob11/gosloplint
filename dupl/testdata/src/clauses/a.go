package fixture

type limits struct{ low, high, step int }

func parse(key string, args []int, l *limits) int {
	switch key {
	case "low":
		if len(args) < 2 {
			return -1
		}
		v := args[0]*10 + args[1]
		if v < 0 || v > 100 {
			return -2
		}
		v -= args[1] % 3
		l.step = l.step*2 + v
		for v%l.step != 0 {
			v++
		}
		l.low = v
	case "high":
		if len(args) < 2 { // want `lines 22-47 duplicate lines 8-19 of a.go \(parse\)`
			return -1
		}
		v := args[0]*10 + args[1]
		if v < 0 || v > 100 {
			return -2
		}
		v -= args[1] % 3
		l.step = l.step*2 + v
		for v%l.step != 0 {
			v++
		}
		l.high = v
	case "step":
		if len(args) < 2 {
			return -1
		}
		v := args[0]*10 + args[1]
		if v < 0 || v > 100 {
			return -2
		}
		v -= args[1] % 3
		l.step = l.step*2 + v
		for v%l.step != 0 {
			v++
		}
		l.step = v
	}
	return 0
}
