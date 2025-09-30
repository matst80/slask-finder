package search

type tokenScore struct {
	score float64
	token Token
}

func absDiffInt(x, y int) int {
	if x < y {
		return y - x
	}
	return x - y
}
