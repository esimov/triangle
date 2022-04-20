package triangle

import (
	"image"
	"image/color"
	"math"

	"golang.org/x/exp/constraints"
)

// Grayscale converts the image to grayscale mode.
func Grayscale(src *image.NRGBA) *image.NRGBA {
	dx, dy := src.Bounds().Max.X, src.Bounds().Max.Y
	dst := image.NewNRGBA(src.Bounds())
	for x := 0; x < dx; x++ {
		for y := 0; y < dy; y++ {
			r, g, b, a := src.At(x, y).RGBA()
			// check for swapped color channel order
			if r == 0 {
				r = a
			}
			lum := float32(r)*0.299 + float32(g)*0.587 + float32(b)*0.114
			pixel := color.Gray{uint8(lum / 256)}
			dst.Set(x, y, pixel)
		}
	}
	return dst
}

// ImgToNRGBA converts any image type to *image.NRGBA with min-point at (0, 0).
func ImgToNRGBA(img image.Image) *image.NRGBA {
	srcBounds := img.Bounds()
	if srcBounds.Min.X == 0 && srcBounds.Min.Y == 0 {
		if src0, ok := img.(*image.NRGBA); ok {
			return src0
		}
	}
	srcMinX := srcBounds.Min.X
	srcMinY := srcBounds.Min.Y

	dstBounds := srcBounds.Sub(srcBounds.Min)
	dstW := dstBounds.Dx()
	dstH := dstBounds.Dy()
	dst := image.NewNRGBA(dstBounds)

	switch src := img.(type) {
	case *image.NRGBA:
		rowSize := srcBounds.Dx() * 4
		for dstY := 0; dstY < dstH; dstY++ {
			di := dst.PixOffset(0, dstY)
			si := src.PixOffset(srcMinX, srcMinY+dstY)
			for dstX := 0; dstX < dstW; dstX++ {
				copy(dst.Pix[di:di+rowSize], src.Pix[si:si+rowSize])
			}
		}
	case *image.YCbCr:
		for dstY := 0; dstY < dstH; dstY++ {
			di := dst.PixOffset(0, dstY)
			for dstX := 0; dstX < dstW; dstX++ {
				srcX := srcMinX + dstX
				srcY := srcMinY + dstY
				siy := src.YOffset(srcX, srcY)
				sic := src.COffset(srcX, srcY)
				r, g, b := color.YCbCrToRGB(src.Y[siy], src.Cb[sic], src.Cr[sic])
				dst.Pix[di+0] = r
				dst.Pix[di+1] = g
				dst.Pix[di+2] = b
				dst.Pix[di+3] = 0xff
				di += 4
			}
		}
	default:
		for dstY := 0; dstY < dstH; dstY++ {
			di := dst.PixOffset(0, dstY)
			for dstX := 0; dstX < dstW; dstX++ {
				c := color.NRGBAModel.Convert(img.At(srcMinX+dstX, srcMinY+dstY)).(color.NRGBA)
				dst.Pix[di+0] = c.R
				dst.Pix[di+1] = c.G
				dst.Pix[di+2] = c.B
				dst.Pix[di+3] = c.A
				di += 4
			}
		}
	}

	return dst
}

// convolutionFilter applies a mathematical operation over the source image by taking
// the matrix table as input parameter and convolving the matrix values over the pixels data.
func convolutionFilter(matrix []float64, img *image.NRGBA, divisor float64) {
	var (
		divscalar float64

		width  = img.Bounds().Dx()
		height = img.Bounds().Dy()
		size   = math.Sqrt(float64(len(matrix)))
		dim    = int(size * 0.5)
	)

	divscalar = 1 / divisor
	if divscalar != 1 {
		for k := 0; k < len(matrix); k++ {
			matrix[k] *= divscalar
		}
	}
	copy := make([]int, len(img.Pix)/4)
	for i := 0; i < len(copy); i++ {
		copy[i] = int(img.Pix[i*4])
	}

	for y := 0; y < height; y++ {
		istep := y * width

		for x := 0; x < width; x++ {
			var r int

			for row := -dim; row <= dim; row++ {
				sy := y + row
				jstep := sy * width
				kstep := (row + dim) * int(size)

				if sy >= 0 && sy < height {
					for col := -dim; col <= dim; col++ {
						sx := x + col
						v := matrix[(col+dim)+kstep]
						if sx >= 0 && sx < width {
							r += int(float64(copy[sx+jstep]) * v)
						}
					}
				}
			}

			if r < 0 {
				r = 0
			} else if r > 255 {
				r = 255
			}

			img.Pix[(x+istep)<<2] = uint8(r) & 0xFF
		}
	}
}

// Min returns the smallest value between two numbers.
func Min[T constraints.Ordered](values ...T) T {
	var acc T = values[0]

	for _, v := range values {
		if v < acc {
			acc = v
		}
	}
	return acc
}

// Max returns the biggest value between two numbers.
func Max[T constraints.Ordered](values ...T) T {
	var acc T = values[0]

	for _, v := range values {
		if v > acc {
			acc = v
		}
	}
	return acc
}

// setBlurMatrix populates a matrix table with values used in conjunction with the convolution filter operator.
func setBlurMatrix(size int) []float64 {
	var (
		side   = size*2 + 1
		length = side * side
		matrix = make([]float64, length)
	)

	for i := 0; i < length; i++ {
		matrix[i] = 1
	}

	return matrix
}

// setEdgeMatrix populates a matrix table with values used in conjunction with the convolution filter operator.
func setEdgeMatrix(size int) []float64 {
	var (
		side   = size*2 + 1
		length = side * side
		center = int(length / 2)
		matrix = make([]float64, length)
	)

	for i := 0; i < length; i++ {
		if i == center {
			matrix[i] = float64(-length)
		} else {
			matrix[i] = 1
		}

	}
	return matrix
}
