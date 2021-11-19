package sharp_utils

type Point struct {
	X uint
	Y uint
}

func NewPoint(x, y uint) Point {
	return Point{x, y}
}