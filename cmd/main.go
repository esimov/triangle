package main

import (
	_ "image/png"
	_ "image/jpeg"
	"os"
	"image"
	"image/color"
	"log"
	"time"
	"fmt"
	tri "github.com/esimov/triangulator"
	"github.com/fogleman/gg"
)

type Canvas struct {
	*gg.Context
}

// Canvas constructor
func NewCanvas(ctx *gg.Context)*Canvas {
	return &Canvas{ctx}
}

func main() {
	file, err := os.Open("brunettes_women_blue_eyes_long_2560x1440_wallpapername.com.jpg")
	defer file.Close()
	src, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	delaunay := &tri.Delaunay{}

	start := time.Now()
	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	blur := tri.Stackblur(src, uint32(width), uint32(height), 2)
	gray := tri.Grayscale(blur)
	sobel := tri.Sobel(gray, 20)
	points := tri.GetEdgePoints(sobel, 50)

	//os.Exit(2)
	dst := toNRGBA(src)

	triangles := delaunay.Init(width, height).Insert(points).GetTriangles()
	fmt.Println("LEN:", len(triangles))

	ctx := gg.NewContext(width, height)

	for i := 0; i < len(triangles); i++ {
		t := triangles[i]
		p0, p1, p2 := t.Nodes[0], t.Nodes[1], t.Nodes[2]

		ctx.MoveTo(float64(p0.X), float64(p0.Y))
		ctx.LineTo(float64(p1.X), float64(p1.Y))
		ctx.LineTo(float64(p2.X), float64(p2.Y))
		ctx.LineTo(float64(p0.X), float64(p0.Y))

		cx := float64(p0.X + p1.X + p2.X) * 0.33333
		cy := float64(p0.Y + p1.Y + p2.Y) * 0.33333

		j := ((int(cx) | 0) + (int(cy) | 0) * width) * 4
		r, g, b := dst.Pix[j], dst.Pix[j+1], dst.Pix[j+2]
		rgb := gg.NewSolidPattern(color.RGBA{R:r, G:g, B:b, A:255})
		//fmt.Println(r, ":", g, ":", b)
		ctx.SetFillStyle(rgb)
		ctx.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R:0, G:0, B:0, A:0}))
		ctx.Fill()
	}

	if err = ctx.SavePNG("output.png"); err != nil {
		log.Fatal(err)
	}

	end := time.Since(start)
	fmt.Println(end)
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