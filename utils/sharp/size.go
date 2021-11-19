package sharp_utils

import "math"

// Size represents the image width and height values
type Size struct {
	Width  uint `json:"width"`
	Height uint `json:"height"`
}

func NewSize(width, height uint) Size {
	return Size{width, height}
}

func NewSizeFromFloat(width, height float64) Size {
	return Size{uint(width), uint(height)}
}

func NewSizeFromInt(width, height int) Size {
	return Size{uint(width), uint(height)}
}

func NewZeroSize() Size {
	return Size{}
}

func (s Size) IsZero() bool {
	return s.Height == 0 || s.Width == 0
}

// Calculates scaling factors using old and new image dimensions.
func calcSizeFactors(width, height uint, oldWidth, oldHeight float64) (scaleX, scaleY float64) {
	if width == 0 {
		if height == 0 {
			scaleX = 1.0
			scaleY = 1.0
		} else {
			scaleY = oldHeight / float64(height)
			scaleX = scaleY
		}
	} else {
		scaleX = oldWidth / float64(width)
		if height == 0 {
			scaleY = scaleX
		} else {
			scaleY = oldHeight / float64(height)
		}
	}
	return
}

// ResizeWithAspectRatio 等比缩放新的长宽,
// 参考targetSize 按照原Size的长宽比进行缩放
func (s Size) ResizeWithAspectRatio(targetSize Size) Size {
	if s.IsZero() {
		return NewZeroSize()
	}

	scaleX, scaleY := calcSizeFactors(targetSize.Width, targetSize.Height, float64(s.Width), float64(s.Height))

	scale := math.Max(scaleX, scaleY)

	return NewSizeFromFloat(float64(s.Width) / scale, float64(s.Height) / scale)
}

func (s Size) Reverse() Size {
	return NewSize(s.Height, s.Width)
}