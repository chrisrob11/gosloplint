package tiny // want `tiny: twice.go is 3 lines and declares no type: fold it into the file of the type its code serves, or its only caller's file`

func twice(x int) int { return 2 * x }
