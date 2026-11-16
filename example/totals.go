package slop

// Totals sums readings by week and by month, by one loop written twice.
type Totals struct{ cap int }

// Weekly clamps and sums days.
func (t Totals) Weekly(days []int) int {
	sum, peak := 0, 0
	for _, d := range days {
		if d > t.cap {
			d = t.cap
		}
		sum += d
		if d > peak {
			peak = d
		}
	}
	sum -= peak / 2
	if peak == 0 {
		return 0
	}
	avg := sum / peak
	if avg > t.cap {
		return t.cap
	}
	return avg
}

// Monthly is Weekly again under other names: dupl reports it.
func (t Totals) Monthly(weeks []int) int {
	total, top := 0, 0
	for _, w := range weeks {
		if w > t.cap {
			w = t.cap
		}
		total += w
		if w > top {
			top = w
		}
	}
	total -= top / 2
	if top == 0 {
		return 0
	}
	mean := total / top
	if mean > t.cap {
		return t.cap
	}
	return mean
}
