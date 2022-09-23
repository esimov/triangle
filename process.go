package triangle

import (
	"errors"
	"image"
	"image/color"
	"image/draw"
	"io"

	"github.com/fogleman/gg"
)

const (
	// WithoutWireframe - generates triangles without stroke
	WithoutWireframe = iota
	// WithWireframe - generates triangles with stroke
	WithWireframe
	// WireframeOnly - generates triangles only with wireframe
	WireframeOnly
)

// Processor encompasses all of the currently supported processing options.
type Processor struct {
	// BlurRadius defines the intensity of the applied blur filter.
	BlurRadius int
	// SobelThreshold defines the threshold intesinty of the sobel edge detector.
	// By increasing this value the contours of the detected objects will be more evident.
	SobelThreshold int
	// PointsThreshold defines the threshold of computed pixel value below a point is generated.
	PointsThreshold int
	// PointRate defines the point rate by which the generated polygons will be multiplied by.
	// The lower this value the bigger the polygons will be.
	PointRate float64
	// BlurFactor defines the factor used to populate the matrix table in conjunction with the convolution filter operator.
	// This value will affect the outcome of the final triangulated image.
	BlurFactor int
	// EdgeFactor defines the factor used to populate the matrix table in conjunction with the convolution filter operator.
	// The bigger this value is the more cubic alike will be the final image.
	EdgeFactor int
	// MaxPoints holds the maximum number of generated points the vertices/triangles will be generated from.
	MaxPoints int
	// Wireframe defines the visual appearence of the generated vertices (WithoutWireframe|WithWireframe|WireframeOnly).
	Wireframe int
	// Noise defines the intensity of the noise factor used to give a noisy, despeckle like touch of the final image.
	Noise int
	// StrokeWidth defines the contour width in case of using WithWireframe | WireframeOnly mode.
	StrokeWidth float64
	// IsStrokeSolid - when this is set as true, the applied stroke color will be black.
	IsStrokeSolid bool
	// Grayscale will generate the output in grayscale mode.
	Grayscale bool
	// OutputToSVG saves the generated triangles to an SVG file.
	OutputToSVG bool
	// ShowInBrowser shows the generated svg file in the browser.
	ShowInBrowser bool
	// BgColor defines the background color in case of using transparent images as source files.
	// By default the background is transparent, but it can be changed using a hexadecimal format, like #fff or #ffff00.
	BgColor string
}

// Line defines the SVG line parameters.
type Line struct {
	P0          Node
	P1          Node
	P2          Node
	P3          Node
	FillColor   color.RGBA
	StrokeColor color.RGBA
}

// Image extends the Processor struct.
type Image struct {
	Processor
}

// SVG extends the Processor struct with the SVG parameters.
type SVG struct {
	Width         int
	Height        int
	Title         string
	Lines         []Line
	Color         color.RGBA
	Description   string
	StrokeLineCap string
	StrokeWidth   float64
	Processor
}

// Fn is a callback function used on SVG generation.
type Fn func()

// Drawer interface defines the Draw method.
// This interface should be implemented by every struct which declares a Draw method.
// By using this method the image can be triangulated as raster type or SVG.
type Drawer interface {
	Draw(image.Image, Processor, Fn) (image.Image, []Triangle, []Point, error)
}

// Draw triangulates the source image and outputs the result to a raster type.
// It returns the number of triangles generated, the number of points and the error in case exists.
func (im *Image) Draw(src image.Image, proc Processor, fn Fn) (image.Image, []Triangle, []Point, error) {
	var (
		err         error
		strokeColor color.RGBA
	)

	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	if width <= 1 || height <= 1 {
		err = errors.New("The image width and height must be greater than 1px.\n")
		return nil, nil, nil, err
	}

	// Define a new context and fill it with a background color.
	ctx := gg.NewContext(width, height)
	ctx.DrawRectangle(0, 0, float64(width), float64(height))

	if im.BgColor != "" {
		ctx.SetRGBA(1, 1, 1, 1)
	} else {
		ctx.SetRGBA(0, 0, 0, 0)
	}
	ctx.Fill()

	img, triangles, points := genTriangles(src, proc)
	if len(triangles) == 0 {
		return img, nil, nil, err
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

		j := (int(cx) + int(cy)*width) * 4
		r, g, b, a := img.Pix[j], img.Pix[j+1], img.Pix[j+2], img.Pix[j+3]
		if im.IsStrokeSolid {
			strokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
		} else {
			strokeColor = color.RGBA{R: r, G: g, B: b, A: 255}
		}

		switch im.Wireframe {
		case WithoutWireframe:
			if a != 0 {
				ctx.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: r, G: g, B: b, A: 255}))
			} else if im.BgColor != "" {
				ctx.SetHexColor(im.BgColor)
			}
			ctx.FillPreserve()
			ctx.Fill()
		case WithWireframe:
			if a != 0 {
				ctx.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: r, G: g, B: b, A: 255}))
				ctx.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 0, G: 0, B: 0, A: 20}))
			} else if im.BgColor != "" {
				ctx.SetHexColor(im.BgColor)
			}
			ctx.SetLineWidth(im.StrokeWidth)
			ctx.FillPreserve()
			ctx.StrokePreserve()
			ctx.Stroke()
		case WireframeOnly:
			if a != 0 {
				ctx.SetStrokeStyle(gg.NewSolidPattern(strokeColor))
			} else if im.BgColor != "" {
				ctx.SetHexColor(im.BgColor)
			}
			ctx.SetLineWidth(im.StrokeWidth)
			ctx.StrokePreserve()
			ctx.Stroke()
		}
		ctx.Pop()
	}

	newImg := ctx.Image()

	// Apply a noise on the final image.
	if im.Noise > 0 {
		addNoise(im.Noise, newImg.(*image.RGBA))
	}
	fn()
	return newImg, triangles, points, err
}

// DecodeImage calls the decodeImage utility function which
// decodes an image file type to the generic image.Image type.
func (im *Image) DecodeImage(input io.Reader) (image.Image, error) {
	return decodeImage(input)
}

// Draw triangulates the source image and outputs the result to an SVG file.
// It has the same method signature as the rester Draw method, only that accepts a callback function
// for further processing, like opening the generated SVG file in the web browser.
// It returns the number of triangles generated, the number of points and the error in case exists.
func (svg *SVG) Draw(src image.Image, proc Processor, fn Fn) (image.Image, []Triangle, []Point, error) {
	var (
		err         error
		lines       []Line
		fillColor   color.RGBA
		strokeColor color.RGBA
	)

	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	if width <= 1 || height <= 1 {
		err := errors.New("The image width and height must be greater than 1px.\n")
		return nil, nil, nil, err
	}

	ctx := gg.NewContext(width, height)
	ctx.DrawRectangle(0, 0, float64(width), float64(height))
	ctx.SetRGBA(1, 1, 1, 1)
	ctx.Fill()

	img, triangles, points := genTriangles(src, proc)
	if len(triangles) == 0 {
		return img, nil, nil, err
	}

	for _, t := range triangles {
		p0, p1, p2 := t.Nodes[0], t.Nodes[1], t.Nodes[2]
		cx := float64(p0.X+p1.X+p2.X) * 0.33333
		cy := float64(p0.Y+p1.Y+p2.Y) * 0.33333

		j := ((int(cx) | 0) + (int(cy)|0)*width) * 4
		r, g, b := img.Pix[j], img.Pix[j+1], img.Pix[j+2]

		if svg.IsStrokeSolid {
			strokeColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
		} else {
			strokeColor = color.RGBA{R: r, G: g, B: b, A: 255}
		}

		switch svg.Wireframe {
		case WithoutWireframe, WithWireframe:
			fillColor = color.RGBA{R: r, G: g, B: b, A: 255}
		case WireframeOnly:
			fillColor = color.RGBA{R: 255, G: 255, B: 255, A: 255}
		}
		lines = append(lines, []Line{
			{
				Node{p0.X, p0.Y},
				Node{p1.X, p1.Y},
				Node{p2.X, p2.Y},
				Node{p0.X, p0.Y},
				fillColor,
				strokeColor,
			},
		}...)
	}
	svg.Width = width
	svg.Height = height
	svg.Lines = lines

	// Trigger the callback function after the generation is completed.
	fn()
	return img, triangles, points, err
}

// DecodeImage calls the decodeImage utility function which
// decodes an image file type to the generic image.Image type.
func (svg *SVG) DecodeImage(input io.Reader) (image.Image, error) {
	return decodeImage(input)
}

// decodeImage decodes an input argument of type io.Reader to an image.
func decodeImage(input io.Reader) (image.Image, error) {
	src, _, err := image.Decode(input)
	if err != nil {
		return nil, err
	}
	return src, nil
}

// genTriangles generates the triangles and returns the triangles and points slices.
func genTriangles(src image.Image, p Processor) (*image.NRGBA, []Triangle, []Point) {
	var srcImg *image.NRGBA
	delaunay := &Delaunay{}

	img := ImgToNRGBA(src)
	w, h := img.Bounds().Max.X, img.Bounds().Max.Y

	newimg := image.NewNRGBA(img.Bounds())
	draw.Draw(newimg, img.Bounds(), img, image.Point{}, draw.Src)

	blur := StackBlur(img, uint32(p.BlurRadius))
	if p.MaxPoints < 1 {
		return blur, nil, nil
	}

	gray := Grayscale(blur)
	if p.Grayscale {
		srcImg = gray
	} else {
		srcImg = newimg
	}

	blurMatrix := setBlurMatrix(p.BlurFactor)
	edgeMatrix := setEdgeMatrix(p.EdgeFactor)

	convolutionFilter(blurMatrix, img, float64(len(blurMatrix)))
	convolutionFilter(edgeMatrix, img, float64(p.EdgeFactor))

	points := p.GetPoints(img, p.PointsThreshold, p.MaxPoints)
	triangles := delaunay.Init(w, h).Insert(points).GetTriangles()

	return srcImg, triangles, points
}
