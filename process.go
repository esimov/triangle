package triangle

import (
	"image"
	"image/color"
	"image/png"
	"io"
	"os"

	"github.com/fogleman/gg"
)

const (
	WITHOUT_WIREFRAME = iota
	WITH_WIREFRAME
	WIREFRAME_ONLY
)

// Processor : type with processing options
type Processor struct {
	BlurRadius      int
	SobelThreshold  int
	PointsThreshold int
	MaxPoints       int
	Wireframe       int
	Noise           int
	LineWidth       float64
	IsSolid         bool
	Grayscale       bool
}

// Process : Triangulate the source image
func (p *Processor) Process(file io.Reader, output string) (*os.File, []Triangle, []Point, error) {
	src, _, err := image.Decode(file)
	if err != nil {
		return nil, nil, nil, err
	}

	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	ctx := gg.NewContext(width, height)
	ctx.DrawRectangle(0, 0, float64(width), float64(height))
	ctx.SetRGBA(1, 1, 1, 1)
	ctx.Fill()

	delaunay := &Delaunay{}
	img := toNRGBA(src)

	blur := Stackblur(img, uint32(width), uint32(height), uint32(p.BlurRadius))
	gray := Grayscale(blur)
	sobel := SobelFilter(gray, float64(p.SobelThreshold))
	points := GetEdgePoints(sobel, p.PointsThreshold, p.MaxPoints)
	triangles := delaunay.Init(width, height).Insert(points).GetTriangles()

	var srcImg *image.NRGBA
	if p.Grayscale {
		srcImg = gray
	} else {
		srcImg = img
	}

	for _, t := range triangles {
		p0, p1, p2 := t.Nodes[0], t.Nodes[1], t.Nodes[2]

		ctx.Push()
		ctx.MoveTo(float64(p0.X), float64(p0.Y))
		ctx.LineTo(float64(p1.X), float64(p1.Y))
		ctx.LineTo(float64(p2.X), float64(p2.Y))
		ctx.LineTo(float64(p0.X), float64(p0.Y))

		cx := float64(p0.X+p1.X+p2.X) * 0.33333
		cy := float64(p0.Y+p1.Y+p2.Y) * 0.33333

		j := ((int(cx) | 0) + (int(cy)|0)*width) * 4
		r, g, b := srcImg.Pix[j], srcImg.Pix[j+1], srcImg.Pix[j+2]

		var lineColor color.RGBA
		if p.IsSolid {
			lineColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
		} else {
			lineColor = color.RGBA{R: r, G: g, B: b, A: 255}
		}

		switch p.Wireframe {
		case WITHOUT_WIREFRAME:
			ctx.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: r, G: g, B: b, A: 255}))
			ctx.FillPreserve()
			ctx.Fill()
		case WITH_WIREFRAME:
			ctx.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: r, G: g, B: b, A: 255}))
			ctx.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 0, G: 0, B: 0, A: 20}))
			ctx.SetLineWidth(p.LineWidth)
			ctx.FillPreserve()
			ctx.StrokePreserve()
			ctx.Stroke()
		case WIREFRAME_ONLY:
			ctx.SetStrokeStyle(gg.NewSolidPattern(lineColor))
			ctx.SetLineWidth(p.LineWidth)
			ctx.StrokePreserve()
			ctx.Stroke()
		}
		ctx.Pop()
	}

	fq, err := os.Create(output)
	if err != nil {
		return nil, nil, nil, err
	}
	defer fq.Close()

	newimg := ctx.Image()
	// Apply a noise on the final image. This will give it a more artistic look.
	if p.Noise > 0 {
		noisyImg := Noise(p.Noise, newimg, newimg.Bounds().Dx(), newimg.Bounds().Dy())
		if err = png.Encode(fq, noisyImg); err != nil {
			return nil, nil, nil, err
		}
	} else {
		if err = png.Encode(fq, newimg); err != nil {
			return nil, nil, nil, err
		}
	}

	return fq, triangles, points, err
}

// toNRGBA converts any image type to *image.NRGBA with min-point at (0, 0).
func toNRGBA(img image.Image) *image.NRGBA {
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
	case *image.Gray:
		for dstY := 0; dstY < dstH; dstY++ {
			di := dst.PixOffset(0, dstY)
			si := src.PixOffset(srcMinX, srcMinY+dstY)
			for dstX := 0; dstX < dstW; dstX++ {
				c := src.Pix[si]
				dst.Pix[di+0] = c
				dst.Pix[di+1] = c
				dst.Pix[di+2] = c
				dst.Pix[di+3] = 0xff
				di += 4
				si += 2
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
