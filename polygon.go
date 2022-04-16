package triangle

import (
	"image"
	"math/rand"
	"time"
)

// pointRate defines the default point rate.
// Changing this value will modify the triangles sizes.
const pointRate = 0.875

// GetEdgePoints retrieves the triangle points after the Sobel threshold has been applied.
func GetEdgePoints(img *image.NRGBA, threshold, maxPoints int) []Point {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	width, height := img.Bounds().Dx(), img.Bounds().Dy()

	var (
		sum, total     uint8
		x, y, sx, sy   int
		row, col, step int
		points         []Point
		dpoints        []Point
	)

	for y = 0; y < height; y++ {
		for x = 0; x < width; x++ {
			sum, total = 0, 0

			for row = -1; row <= 1; row++ {
				sy = y + row
				step = sy * width
				if sy >= 0 && sy < height {
					for col = -1; col <= 1; col++ {
						sx = x + col
						if sx >= 0 && sx < width {
							sum += img.Pix[(sx+step)<<2]
							total++
						}
					}
				}
			}
			if total > 0 {
				sum /= total
			}
			if sum > uint8(threshold) {
				points = append(points, Point{X: float64(x), Y: float64(y)})
			}
		}
	}
	ilen := len(points)
	limit := int(float64(ilen) * pointRate)

	if limit > maxPoints {
		limit = maxPoints
	}

	for i := 0; i < limit && i < ilen; i++ {
		j := int(float64(ilen) * r.Float64())
		dpoints = append(dpoints, points[j])
	}
	return dpoints
}
