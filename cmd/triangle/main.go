package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/esimov/triangle"
	"github.com/esimov/triangle/utils"
)

const helperBanner = `
     ▲ TRIANGLE

     Version: %s

`

// The default http address used for accessing the generated SVG file in case of -web flag is used.
const httpAddress = "http://localhost:8080"

type MessageType int

type result struct {
	path      string
	triangles []triangle.Triangle
	points    []triangle.Point
	err       error
}

// The message types used accross the CLI application.
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
		workers         = flag.Int("w", runtime.NumCPU(), "Number of files to process concurrently")
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

	s := new(utils.Spinner)
	s.Start("Generating the triangulated image...")
	start := time.Now()

	switch mode := fs.Mode(); {
	case mode.IsDir():
		var wg sync.WaitGroup

		// Read destination file or directory.
		dst, err := os.Stat(*destination)
		if err != nil {
			log.Fatalf(
				decorateText("Unable to get dir stats: %v", ErrorMessage),
				decorateText(err.Error(), DefaultMessage),
			)
		}

		//@TODO create destination directory in case it does not exists.

		// Check if the image destination is a directory or a file.
		if dst.Mode().IsRegular() {
			log.Fatalf(decorateText("Please specify a directory as destination!.", ErrorMessage))
		}

		// Process image files from directory concurrently.
		ch := make(chan result)
		done := make(chan interface{})
		defer close(done)

		paths, errc := walkDir(done, *source, srcExts)

		wg.Add(*workers)
		for i := 0; i < *workers; i++ {
			go func() {
				defer wg.Done()
				consumer(done, paths, *destination, p, ch)
			}()
		}

		go func() {
			defer close(ch)
			wg.Wait()
		}()

		for res := range ch {
			showProcessStatus(res.path, res.triangles, res.points, res.err)
		}

		if err := <-errc; err != nil {
			fmt.Println(decorateText(err.Error(), ErrorMessage))
		}

	case mode.IsRegular():
		ext := filepath.Ext(*destination)
		if !inSlice(ext, destExts) {
			log.Fatalf(decorateText(fmt.Sprintf("File type not supported: %v", ext), ErrorMessage))
		}

		triangles, points, err := processor(*source, *destination, p, func() {
			if p.ShowInBrowser {
				svg, err := os.OpenFile(*destination, os.O_CREATE|os.O_RDWR, 0755)
				if err != nil {
					log.Fatalf("Unable to open the destination file: %v", err)
				}

				b, err := ioutil.ReadAll(svg)
				if err != nil {
					log.Fatalf("Unable to read the SVG file: %v", err)
				}
				fmt.Printf("\n\tYou can access the generated image under the following url: %s ", decorateText(httpAddress, SuccessMessage))

				handler := func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "image/svg+xml")
					w.Write(b)
				}
				http.HandleFunc("/", handler)
				log.Fatal(http.ListenAndServe(strings.TrimPrefix(httpAddress, "http://"), nil))
			}
		})

		showProcessStatus(*destination, triangles, points, err)
	}

	procTime := time.Since(start)
	s.Stop()

	fmt.Printf("Generated in: %s\n", decorateText(fmt.Sprintf("%.2fs", procTime.Seconds()), SuccessMessage))
}

// walkDir starts a goroutine to walk the specified directory tree
// and send the path of each regular file on the string channel.
// It sends the result of the walk on the error channel.
// It terminates in case done channel is closed.
func walkDir(
	done <-chan interface{},
	src string,
	srcExts []string,
) (<-chan string, <-chan error) {
	pathChan := make(chan string)
	errChan := make(chan error, 1)

	go func() {
		// Close the paths channel after Walk returns.
		defer close(pathChan)

		errChan <- filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
			isFileSupported := false
			if err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}

			// Get the file base name.
			fx := filepath.Ext(info.Name())
			for _, ext := range srcExts {
				if ext == fx {
					isFileSupported = true
					break
				}
			}

			if isFileSupported {
				select {
				case <-done:
					return errors.New("directory walk cancelled")
				case pathChan <- path:
				}
			}
			return nil
		})
	}()
	return pathChan, errChan
}

// consumer reads the path names from the paths channel and
// calls the triangulator processor against the source image,
// then sends the results on a new channel.
func consumer(
	done <-chan interface{},
	paths <-chan string,
	dest string,
	proc *triangle.Processor,
	res chan<- result,
) {
	for path := range paths {
		dest := filepath.Join(dest, filepath.Base(path))
		triangles, points, err := processor(path, dest, proc, func() {})

		select {
		case <-done:
			return
		case res <- result{
			path:      path,
			triangles: triangles,
			points:    points,
			err:       err,
		}:
		}
	}
}

// processor triangulates the source image and returns the number
// of triangles, points and the error in case it exists.
func processor(src, dst string, proc *triangle.Processor, fn func()) (
	[]triangle.Triangle,
	[]triangle.Point,
	error,
) {
	var (
		// Triangle related variables
		triangles []triangle.Triangle
		points    []triangle.Point
		err       error
	)

	svg := &triangle.SVG{
		Title:         "Image triangulator",
		Lines:         []triangle.Line{},
		Description:   "Convert images to computer generated art using delaunay triangulation.",
		StrokeWidth:   proc.StrokeWidth,
		StrokeLineCap: "round", //butt, round, square
		Processor:     *proc,
	}

	tri := &triangle.Image{*proc}

	file, err := os.Open(src)
	if err != nil {
		log.Fatalf("Unable to open the source file: %v", err)
	}
	defer file.Close()

	fs, err := os.Create(dst)
	if err != nil {
		log.Fatalf("Unable to create the destination file: %v", err)
	}
	defer fs.Close()

	if filepath.Ext(dst) == ".svg" {
		_, triangles, points, err = svg.Draw(file, fs, fn)
	} else {
		_, triangles, points, err = tri.Draw(file, fs, fn)
	}

	return triangles, points, err
}

// showProcessStatus displays the relavant information about the triangulation process.
func showProcessStatus(
	fn string,
	triangles []triangle.Triangle,
	points []triangle.Point,
	err error,
) {
	if err != nil {
		fmt.Printf(
			decorateText("\nError generating the triangulated image: %s", ErrorMessage),
			decorateText(fmt.Sprintf("\n\tReason: %v\n", err.Error()), DefaultMessage),
		)
	} else {
		fmt.Printf(fmt.Sprintf("\nTotal number of %s%d %striangles generated out of %s%d %vpoints\n",
			utils.SuccessColor, len(triangles), utils.DefaultColor, utils.SuccessColor, len(points), utils.DefaultColor),
		)
		fmt.Printf(fmt.Sprintf("Saved as: %s %s✓%s\n\n",
			decorateText(filepath.Base(fn), SuccessMessage),
			utils.SuccessColor,
			utils.DefaultColor,
		))
	}
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

// decorateText show the message types in different colors
func decorateText(s string, msgType MessageType) string {
	switch msgType {
	case SuccessMessage:
		s = utils.SuccessColor + s
	case ErrorMessage:
		s = utils.ErrorColor + s
	case DefaultMessage:
		s = utils.DefaultColor + s
	default:
		return s
	}
	return s + "\x1b[0m"
}
