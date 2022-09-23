package triangle

import (
	"image"
	"image/color"
	"math"
)

// seed basic parameters
type seed struct {
	a         int
	m         int
	randomNum int
	div       float64
}

// addNoise applies a noise factor, like Adobe's grain filter in order to create a despeckle like image.
func addNoise(amount int, src *image.RGBA) {
	size := src.Bounds().Size()
	s := &seed{
		a:         16807,
		m:         0x7fffffff,
		randomNum: 1.0,
		div:       1.0 / 0x7fffffff,
	}
	for x := 0; x < size.X; x++ {
		for y := 0; y < size.Y; y++ {
			noise := (s.random() - 0.01) * float64(amount)
			r, g, b, a := src.At(x, y).RGBA()
			rf, gf, bf := float64(r>>8), float64(g>>8), float64(b>>8)

			// Check if color does not overflow the maximum limit after noise has been applied.
			if math.Abs(rf+noise) < 255 && math.Abs(gf+noise) < 255 && math.Abs(bf+noise) < 255 {
				rf += noise
				gf += noise
				bf += noise
			}
			r2 := Max(0, Min(255, uint8(rf)))
			g2 := Max(0, Min(255, uint8(gf)))
			b2 := Max(0, Min(255, uint8(bf)))

			src.Set(x, y, color.RGBA{R: r2, G: g2, B: b2, A: uint8(a)})
		}
	}
}

// nextLongRand retrieve the next long random number.
func (s *seed) nextLongRand(seed int) int {
	lo := s.a * (seed & 0xffff)
	hi := s.a * (seed >> 16)
	lo += (hi & 0x7fff) << 16

	if lo > s.m {
		lo &= s.m
		lo++
	}
	lo += hi >> 15
	if lo > s.m {
		lo &= s.m
		lo++
	}
	return lo
}

// random generates a random seed.
func (s *seed) random() float64 {
	s.randomNum = s.nextLongRand(s.randomNum)
	return float64(s.randomNum) * s.div
}
