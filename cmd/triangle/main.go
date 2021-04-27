package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/esimov/triangle"
)

const helperBanner = `
     ▲ TRIANGLE

     Version: %s

`
const (
	httpAddress     = "http://localhost:8080"
	errorMsgColor   = "\x1b[0;31m"
	successMsgColor = "\x1b[0;32m"
	defaultMsgColor = "\x1b[0m"
)

type MessageType int

const (
	DefaultMessage MessageType = iota
	SuccessMessage
	ErrorMessage
)

// version indicates the current build version.
var version string

func main() {
	var (
		// Command line flags
		source          = flag.String("in", "", "Source image")
		destination     = flag.String("out", "", "Destination image")
		blurRadius      = flag.Int("blur", 4, "Blur radius")
		sobelThreshold  = flag.Int("sobel", 10, "Sobel filter threshold")
		pointsThreshold = flag.Int("th", 20, "Points threshold")
		maxPoints       = flag.Int("pts", 2500, "Maximum number of points")
		wireframe       = flag.Int("wf", 0, "Wireframe mode (0: without stroke, 1: with stroke, 2: stroke only)")
		noise           = flag.Int("noise", 0, "Noise factor")
		strokeWidth     = flag.Float64("stroke", 1, "Stroke width")
		isStrokeSolid   = flag.Bool("solid", false, "Use solid stroke color (yes/no)")
		grayscale       = flag.Bool("gray", false, "Output in grayscale mode")
		showInBrowser   = flag.Bool("web", false, "Open the SVG file in the web browser")
		bgColor         = flag.String("bg", "", "Background color (specified as hex value)")

		// Triangle related variables
		triangles []triangle.Triangle
		points    []triangle.Point
		err       error
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, fmt.Sprintf(helperBanner, version))
		flag.PrintDefaults()
	}
	flag.Parse()

	if len(*source) == 0 || len(*destination) == 0 {
		log.Fatal("Usage: triangle -in <source> -out <destination>")
	}

	fs, err := os.Stat(*source)
	if err != nil {
		log.Fatalf(
			decorateText("Failed to load the source image: %v", ErrorMessage),
			decorateText(err.Error(), DefaultMessage),
		)
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
		IsStrokeSolid:   *isStrokeSolid,
		Grayscale:       *grayscale,
		ShowInBrowser:   *showInBrowser,
		BgColor:         *bgColor,
	}

	// Supported input image file types.
	srcExts := []string{".jpg", ".jpeg", ".png"}

	// Supported output image file types.
	destExts := []string{".jpg", ".jpeg", ".png", ".svg"}

	switch mode := fs.Mode(); {
	case mode.IsDir():
		// Read source directory.
		files, err := ioutil.ReadDir(*source)
		if err != nil {
			log.Fatalf("Unable to read directory: %v", err)
		}

		// Read destination file or directory.
		dst, err := os.Stat(*destination)
		if err != nil {
			log.Fatalf(
				decorateText("Unable to get dir stats: %v", ErrorMessage),
				decorateText(err.Error(), DefaultMessage),
			)
		}

		// Check if the image destination is a directory or a file.
		if dst.Mode().IsRegular() {
			log.Fatalf(decorateText("Please specify a directory as destination!.", ErrorMessage))
		}
		output, err := filepath.Abs(*destination)
		if err != nil {
			log.Fatalf("Unable to get the absolute path: %v", err)
		}

		// Range over all the image files and save them into a slice.
		images := []string{}
		for _, f := range files {
			ext := filepath.Ext(f.Name())
			for _, iex := range srcExts {
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
		ext := filepath.Ext(*destination)
		if !inSlice(ext, destExts) {
			log.Fatalf(decorateText(fmt.Sprintf("File type not supported: %v", ext), ErrorMessage))
		}
		toProcess[*source] = *destination
	}

	for in, out := range toProcess {
		svg := &triangle.SVG{
			Title:         "Image triangulator",
			Lines:         []triangle.Line{},
			Description:   "Convert images to computer generated art using delaunay triangulation.",
			StrokeWidth:   p.StrokeWidth,
			StrokeLineCap: "round", //butt, round, square
			Processor:     *p,
		}

		tri := &triangle.Image{*p}

		file, err := os.Open(in)
		if err != nil {
			log.Fatalf("Unable to open the source file: %v", err)
		}

		s := new(spinner)
		s.start("Generating triangulated image...")
		start := time.Now()

		fq, err := os.Create(out)
		if err != nil {
			log.Fatalf("Unable to create the destination file: %v", err)
		}

		if filepath.Ext(out) == ".svg" {
			if p.ShowInBrowser {
				if len(toProcess) < 2 {
					_, triangles, points, err = svg.Draw(file, fq, func() {
						svg, err := os.OpenFile(out, os.O_CREATE|os.O_RDWR, 0755)
						if err != nil {
							log.Fatalf("Unable to open the destination file: %v", err)
						}

						b, err := ioutil.ReadAll(svg)
						if err != nil {
							log.Fatalf("Unable to read the SVG file: %v", err)
						}
						fmt.Printf("\n\tYou can access the generated image under the following url: %s ", decorateText(httpAddress, SuccessMessage))
						s.stop()

						handler := func(w http.ResponseWriter, r *http.Request) {
							w.Header().Set("Content-Type", "image/svg+xml")
							w.Write(b)
						}
						http.HandleFunc("/", handler)
						log.Fatal(http.ListenAndServe(strings.TrimPrefix(httpAddress, "http://"), nil))
					})
				} else {
					log.Fatal("Web browser command is supported only for a single file processing.")
				}
			} else {
				_, triangles, points, err = svg.Draw(file, fq, func() {})
				fq.Close()
			}
		} else {
			_, triangles, points, err = tri.Draw(file, fq, func() {})
			fq.Close()
		}
		s.stop()

		if err == nil {
			fmt.Printf("\nGenerated in: %s\n", decorateText(fmt.Sprintf("%.2fs", time.Since(start).Seconds()), SuccessMessage))
			fmt.Printf(fmt.Sprintf("%sTotal number of %s%d %striangles generated out of %s%d %vpoints\n",
				defaultMsgColor, successMsgColor, len(triangles), defaultMsgColor, successMsgColor, len(points), DefaultMessage))
			fmt.Printf(fmt.Sprintf("Saved on: %s %s✓\n\n", fq.Name(), successMsgColor))
		} else {
			fmt.Printf(decorateText(fmt.Sprintf("\nError generating the triangulated image: %s \n\tReason: %s\n", file.Name(), err.Error()), ErrorMessage))
		}
		file.Close()
	}
}

// decorateText show the message types in different colors
func decorateText(s string, msgType MessageType) string {
	switch msgType {
	case SuccessMessage:
		s = successMsgColor + s
	case ErrorMessage:
		s = errorMsgColor + s
	case DefaultMessage:
		s = defaultMsgColor + s
	default:
		return s
	}
	return s + "\x1b[0m"
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
					fmt.Printf("\r%s%s %c%s", message, successMsgColor, r, defaultMsgColor)
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

// inSlice checks if the item exists in the slice.
func inSlice(item string, slice []string) bool {
	for _, it := range slice {
		if it == item {
			return true
		}
	}
	return false
}
