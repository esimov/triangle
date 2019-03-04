package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/esimov/triangle"
)

const WebBrowserUrl = "localhost:8080"

var (
	// Command line flags
	source          = flag.String("in", "", "Source")
	destination     = flag.String("out", "", "Destination")
	blurRadius      = flag.Int("blur", 4, "Blur radius")
	sobelThreshold  = flag.Int("sobel", 10, "Sobel filter threshold")
	pointsThreshold = flag.Int("points", 20, "Points threshold")
	maxPoints       = flag.Int("max", 2500, "Maximum number of points")
	wireframe       = flag.Int("wireframe", 0, "Wireframe mode")
	noise           = flag.Int("noise", 0, "Noise factor")
	strokeWidth     = flag.Float64("stroke", 1, "Stroke width")
	isSolid         = flag.Bool("solid", false, "Solid line color")
	grayscale       = flag.Bool("gray", false, "Convert to grayscale")
	outputToSVG     = flag.Bool("svg", false, "Save as SVG")
	outputInWeb     = flag.Bool("web", false, "Output SVG in browser")
)

func main() {
	var (
		triangles []triangle.Triangle
		points    []triangle.Point
		err       error
	)
	flag.Parse()

	if len(*source) == 0 || len(*destination) == 0 {
		log.Fatal("Usage: triangle -in input.jpg -out out.jpg")
	}

	fs, err := os.Stat(*source)
	if err != nil {
		log.Fatalf("Unable to open source: %v", err)
	}

	toProcess := make(map[string]string)

	p := &triangle.Processor{
		BlurRadius:      *blurRadius,
		SobelThreshold:  *sobelThreshold,
		PointsThreshold: *pointsThreshold,
		MaxPoints:       *maxPoints,
		Wireframe:       *wireframe,
		Noise:           *noise,
		StrokeWidth:     *strokeWidth,
		IsSolid:         *isSolid,
		Grayscale:       *grayscale,
		OutputToSVG:     *outputToSVG,
		OutputInWeb:     *outputInWeb,
	}

	switch mode := fs.Mode(); {
	case mode.IsDir():
		// Supported image files.
		extensions := []string{".jpg", ".png"}

		// Read source directory.
		files, err := ioutil.ReadDir(*source)
		if err != nil {
			log.Fatalf("Unable to read dir: %v", err)
		}

		// Read destination file or directory.
		dst, err := os.Stat(*destination)
		if err != nil {
			log.Fatalf("Unable to get dir stats: %v", err)
		}

		// Check if the image destination is a directory or a file.
		if dst.Mode().IsRegular() {
			log.Fatal("Please specify a directory as destination!")
			os.Exit(2)
		}
		output, err := filepath.Abs(*destination)
		if err != nil {
			log.Fatalf("Unable to get absolute path: %v", err)
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

		// Process images from directory.
		for _, img := range images {
			// Get the file base name.
			name := strings.TrimSuffix(img, filepath.Ext(img))
			dir := strings.TrimRight(*source, "/")
			out := output + "/" + name + ".png"
			in := dir + "/" + img

			toProcess[in] = out
		}

	case mode.IsRegular():
		toProcess[*source] = *destination
	}

	for in, out := range toProcess {
		svg := &triangle.SVG{
			Title:         "Delaunay image triangulator",
			Lines:         []triangle.Line{},
			Description:   "Convert images to computer generated art using delaunay triangulation.",
			StrokeWidth:   p.StrokeWidth,
			StrokeLineCap: "round", //butt, round, square
			Processor:     *p,
		}

		img := &triangle.Image{*p}

		file, err := os.Open(in)
		if err != nil {
			log.Fatalf("Unable to open source file: %v", err)
		}

		s := new(spinner)
		s.start("Generating triangulated image...")
		start := time.Now()

		fq, err := os.Create(out)
		if err != nil {
			log.Fatalf("Unable to create the output file: %v", err)
		}

		if p.OutputToSVG {
			if p.OutputInWeb {
				if len(toProcess) < 2 {
					_, triangles, points, err = svg.Draw(file, fq, func() {
						svg, err := os.OpenFile(out, os.O_CREATE|os.O_RDWR, 0755)
						if err != nil {
							log.Fatalf("Unable to open output file: %v", err)
						}

						b, err := ioutil.ReadAll(svg)
						if err != nil {
							log.Fatalf("Unable to read svg file: %v", err)
						}
						fmt.Printf("\n\rPlease access %s", WebBrowserUrl)
						s.stop()

						handler := func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("Content-Type", "image/svg+xml")
							w.Write(b)
						}
						http.HandleFunc("/", handler)
						log.Fatal(http.ListenAndServe(WebBrowserUrl, nil))
					})
				} else {
					log.Fatal("Web browser command is supported only for a single file processing.")
				}
			} else {
				_, triangles, points, err = svg.Draw(file, fq, func() {})
				fq.Close()
			}
		} else {
			_, triangles, points, err = img.Draw(file, fq, func() {})
			fq.Close()
		}
		s.stop()

		if err == nil {
			fmt.Printf("\nGenerated in: \x1b[92m%.2fs\n", time.Since(start).Seconds())
			fmt.Printf("\x1b[39mTotal number of \x1b[92m%d \x1b[39mtriangles generated out of \x1b[92m%d \x1b[39mpoints\n", len(triangles), len(points))
			fmt.Printf("Saved as: %s \x1b[92m✓\n\n", path.Base(out))
		} else {
			fmt.Printf("\nError converting image: %s: %s", file.Name(), err.Error())
		}
		file.Close()
	}
}

type spinner struct {
	stopChan chan struct{}
}

// Start process
func (s *spinner) start(message string) {
	s.stopChan = make(chan struct{}, 1)

	go func() {
		for {
			for _, r := range `-\|/` {
				select {
				case <-s.stopChan:
					return
				default:
					fmt.Printf("\r%s%s %c%s", message, "\x1b[92m", r, "\x1b[39m")
					time.Sleep(time.Millisecond * 100)
				}
			}
		}
	}()
}

// Stop process
func (s *spinner) stop() {
	s.stopChan <- struct{}{}
}
