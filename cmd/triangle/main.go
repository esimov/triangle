package main

import (
	"flag"
	"fmt"
	"image"
	"image/color"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	tri "github.com/esimov/triangle"
	"github.com/fogleman/gg"
)

const (
	WITHOUT_WIREFRAME = iota
	WITH_WIREFRAME
	WIREFRAME_ONLY
)

var (
	// Flags
	source          = flag.String("in", "", "Source")
	destination     = flag.String("out", "", "Destination")
	blurRadius      = flag.Int("blur", 4, "Blur radius")
	sobelThreshold  = flag.Int("sobel", 10, "Sobel filter threshold")
	pointsThreshold = flag.Int("points", 20, "Points threshold")
	maxPoints       = flag.Int("max", 2500, "Maximum number of points")
	wireframe       = flag.Int("wireframe", 0, "Wireframe mode")
	noise           = flag.Int("noise", 0, "Noise factor")
	lineWidth       = flag.Float64("width", 1, "Wireframe line width")
	isSolid         = flag.Bool("solid", false, "Solid line color")
	grayscale       = flag.Bool("gray", false, "Convert to grayscale")

	blur, gray, sobel, srcImg *image.NRGBA
	triangles                 []tri.Triangle
	points                    []tri.Point
	lineColor                 color.RGBA
)
var mu sync.Mutex

func init() {
	numcpu := runtime.NumCPU()
	runtime.GOMAXPROCS(numcpu) // Try to use all available CPUs.
}

func main() {
	var wg sync.WaitGroup
	flag.Parse()

	if len(*source) == 0 || len(*destination) == 0 {
		log.Fatal("usage: triangle -in input.jpg -out out.jpg")
	}

	type item struct {
		img *os.File
		err error
	}

	fs, err := os.Stat(*source)
	if err != nil {
		log.Fatal(err)
	}
	switch mode := fs.Mode(); {
	case mode.IsDir():
		// Supported image files.
		extensions := []string{".jpg", ".png"}

		// Read source directory.
		files, err := ioutil.ReadDir(*source)
		if err != nil {
			log.Fatal(err)
		}

		// Read destination file or directory.
		dst, err := os.Stat(*destination)
		if err != nil {
			log.Fatal(err)
		}

		// Check if the image destination is a directory or a file.
		// Abort the process in case of multiple image processing the destination is a file.
		if dst.Mode().IsRegular() {
			log.Fatal("Please specify a directory as destination!")
			os.Exit(2)
		}
		output, err := filepath.Abs(filepath.Base(*destination))
		if err != nil {
			log.Fatal(err)
		}

		// Range over all the image files and save them into a slice.
		images := []string{}
		for _, f := range files {
			ext := filepath.Ext(f.Name())
			for _, iex := range extensions {
				if ext == iex {
					images = append(images, f.Name())
				}
			}
		}

		// Process the image items in separate goroutines.
		ch := make(chan item, len(images))
		for _, img := range images {
			// Get the file base name.
			name := strings.TrimSuffix(img, filepath.Ext(img))
			dir, err := filepath.Abs(filepath.Base(*source))
			if err != nil {
				log.Fatal(err)
			}
			out := output + "/" + name + ".png"
			// Triangulate each image from the specified folder in separate goroutine.
			wg.Add(1)
			go func(in, out string) {
				// Signal the job is done.
				defer wg.Done()
				file, err := os.Open(in)

				if err != nil {
					log.Fatal(err)
				}
				output, err := process(file, out)
				// Send the processing item to the channel.
				ch <- item{output, err}
			}(dir + "/" + img, out)
		}

		// closer
		go func() {
			wg.Wait()
			close(ch)
		}()

		// Drain the channel.
		for f := range ch {
			if f.err != nil {
				fmt.Printf("Error converting image: %s: %s", f.img.Name(), err.Error())
			} else {
				fmt.Printf("Saved as: %s \x1b[92m✓\n\n", path.Base(f.img.Name()))
			}
		}
	case mode.IsRegular():
		file, err := os.Open(*source)
		if err != nil {
			log.Fatal(err)
		}
		f, processErr := process(file, *destination)
		if processErr == nil {
			fmt.Printf("Saved as: %s \x1b[92m✓\n\n", path.Base(*destination))
		} else {
			fmt.Printf("Error converting image: %s: %s", f.Name(), processErr.Error())
		}

		defer file.Close()
	}
}

// Triangulate the source image
func process(file io.Reader, output string) (*os.File, error) {
	mu.Lock()
	defer mu.Unlock()

	src, _, err := image.Decode(file)
	if err != nil {
		return nil, err
	}

	width, height := src.Bounds().Dx(), src.Bounds().Dy()
	ctx := gg.NewContext(width, height)
	ctx.DrawRectangle(0, 0, float64(width), float64(height))
	ctx.SetRGBA(1, 1, 1, 1)
	ctx.Fill()

	delaunay := &tri.Delaunay{}
	img := toNRGBA(src)

	start := time.Now()
	spinner("Generating triangulated image...")

	blur = tri.Stackblur(img, uint32(width), uint32(height), uint32(*blurRadius))
	gray = tri.Grayscale(blur)
	sobel = tri.SobelFilter(gray, float64(*sobelThreshold))
	points = tri.GetEdgePoints(sobel, *pointsThreshold, *maxPoints)
	triangles = delaunay.Init(width, height).Insert(points).GetTriangles()

	if *grayscale {
		srcImg = gray
	} else {
		srcImg = img
	}

	for i := 0; i < len(triangles); i++ {
		t := triangles[i]
		p0, p1, p2 := t.Nodes[0], t.Nodes[1], t.Nodes[2]

		ctx.Push()
		ctx.MoveTo(float64(p0.X), float64(p0.Y))
		ctx.LineTo(float64(p1.X), float64(p1.Y))
		ctx.LineTo(float64(p2.X), float64(p2.Y))
		ctx.LineTo(float64(p0.X), float64(p0.Y))

		cx := float64(p0.X+p1.X+p2.X) * 0.33333
		cy := float64(p0.Y+p1.Y+p2.Y) * 0.33333

		j := ((int(cx) | 0) + (int(cy) | 0) * width) * 4
		r, g, b := srcImg.Pix[j], srcImg.Pix[j+1], srcImg.Pix[j+2]

		if *isSolid {
			lineColor = color.RGBA{R: 0, G: 0, B: 0, A: 255}
		} else {
			lineColor = color.RGBA{R: r, G: g, B: b, A: 255}
		}

		switch *wireframe {
		case WITHOUT_WIREFRAME:
			ctx.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: r, G: g, B: b, A: 255}))
			ctx.FillPreserve()
			ctx.Fill()
		case WITH_WIREFRAME:
			ctx.SetFillStyle(gg.NewSolidPattern(color.RGBA{R: r, G: g, B: b, A: 255}))
			ctx.SetStrokeStyle(gg.NewSolidPattern(color.RGBA{R: 0, G: 0, B: 0, A: 20}))
			ctx.SetLineWidth(*lineWidth)
			ctx.FillPreserve()
			ctx.StrokePreserve()
			ctx.Stroke()
		case WIREFRAME_ONLY:
			ctx.SetStrokeStyle(gg.NewSolidPattern(lineColor))
			ctx.SetLineWidth(*lineWidth)
			ctx.StrokePreserve()
			ctx.Stroke()
		}
		ctx.Pop()
	}

	fq, err := os.Create(output)
	if err != nil {
		return nil, err
	}
	defer fq.Close()

	newimg := ctx.Image()
	// Apply a noise on the final image. This will give it a more artistic look.
	if *noise > 0 {
		noisyImg := tri.Noise(*noise, newimg, newimg.Bounds().Dx(), newimg.Bounds().Dy())
		if err = png.Encode(fq, noisyImg); err != nil {
			return nil, err
		}
	} else {
		if err = png.Encode(fq, newimg); err != nil {
			return nil, err
		}
	}

	end := time.Since(start)
	fmt.Printf("\nGenerated in: \x1b[92m%.2fs\n", end.Seconds())
	fmt.Printf("\x1b[39mTotal number of \x1b[92m%d \x1b[39mtriangles generated out of \x1b[92m%d \x1b[39mpoints\n", len(triangles), len(points))

	return fq, err
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

// Function to visualize the rendering progress
func spinner(message string) {
	go func() {
		for {
			for _, r := range `-\|/` {
				fmt.Printf("\r%s%s %c%s", message, "\x1b[92m", r, "\x1b[39m")
				time.Sleep(time.Millisecond * 100)
			}
		}
	}()
}
