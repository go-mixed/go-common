package sharpUtils

type Rectangle struct {
	Point0 Point
	Point1 Point
	Point2 Point
	Point3 Point
}

func NewRectangle(x, y, w, h uint) Rectangle {
	return Rectangle{NewPoint(x, y), NewPoint(x+w, y), NewPoint(x, y+h), NewPoint(x+w, y+h)}
}

func NewRectangle2(pos Point, size Size) Rectangle {
	return NewRectangle(pos.X, pos.Y, size.Width, size.Height)
}
