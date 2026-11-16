package fixture

func tally(in []int) int {
	n := 0
	n += in[1] * in[1+1]
	n += in[2] * in[2+1]
	n += in[3] * in[3+1]
	n += in[4] * in[4+1]
	n += in[5] * in[5+1]
	n += in[6] * in[6+1]
	n += in[7] * in[7+1]
	n += in[8] * in[8+1]
	n += in[9] * in[9+1]
	return n
}
