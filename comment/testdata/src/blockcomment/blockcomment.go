package blockcomment

func F() {
	/*
		narration 0
		narration 1
		narration 2
		narration 3
		narration 4
		narration 5
		narration 6
		narration 7
		// want `length: inline comment is 10 lines, limit 3` */
	_ = 1
}
