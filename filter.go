package triangulator

import (
	"image"
	"image/draw"
)

type Filter interface {
	Draw(dst draw.Image, src image.Image)
	// Bounds calculates the appropriate bounds of an image after applying the filter.
	Bounds(srcBounds image.Rectangle) (dstBounds image.Rectangle)
}

// GIFT implements a list of filters that can be applied to an image at once.
type Triangulator struct {
	Filters []Filter
}

// New creates a new instance of the filter toolkit and initializes it with the given list of filters.
func New(filters ...Filter) *Triangulator {
	return &Triangulator{
		Filters: filters,
	}
}

// Draw applies all the added filters to the src image and outputs the result to the dst image.
func (t *Triangulator) Draw(dst draw.Image, src image.Image) {
	first, last := 0, len(t.Filters)-1
	var tmpIn image.Image
	var tmpOut draw.Image

	for i, f := range t.Filters {
		if i == first {
			tmpIn = src
		} else {
			tmpIn = tmpOut
		}

		if i == last {
			tmpOut = dst
		} else {
			tmpOut = createTempImage(f.Bounds(tmpIn.Bounds()))
		}

		f.Draw(tmpOut, tmpIn)
	}
}

// create default temp image
func createTempImage(r image.Rectangle) draw.Image {
	return image.NewNRGBA64(r) // use 16 bits per channel images internally
}