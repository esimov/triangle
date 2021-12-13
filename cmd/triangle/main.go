package main

import (
	"errors"
	"flag"
	"fmt"
	"image"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/esimov/triangle"
	"github.com/esimov/triangle/utils"
	"golang.org/x/term"
)

const helperBanner = `
     ▲ TRIANGLE

     Version: %s

`

// pipeName is the file name that indicates stdin/stdout is being used.
const pipeName = "-"

// The default http address used for accessing the generated SVG file in case of -web flag is used.
const httpAddress = "http://localhost:8080"

// maxWorkers sets the maximum number of concurrently running workers.
const maxWorkers = 20

// result holds the relevant information about the triangulation process and the generated image.
type result struct {
	path      string
	triangles []triangle.Triangle
	points    []triangle.Point
	err       error
}

type MessageType int

// The message types used accross the CLI application.
const (
	DefaultMessage MessageType = iota
	SuccessMessage
	ErrorMessage
	TriangleMessage
)

var (
	// imgurl holds the file being accessed be it normal file or pipe name.
	imgurl *os.File
	// spinner used to instantiate and call the progress indicator.
	spinner *utils.Spinner
)

// version indicates the current build version.
var version string

func main() {
	var (
		// Command line flags
		source          = flag.String("in", pipeName, "Source image")
		destination     = flag.String("out", pipeName, "Destination image")
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
		workers         = flag.Int("c", runtime.NumCPU(), "Number of files to process concurrently")

		// File related variables
		fs  os.FileInfo
		err error

		flagsCheck bool
	)

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, fmt.Sprintf(helperBanner, version))
		flag.PrintDefaults()
	}
	flag.Parse()

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

	spinnerText := fmt.Sprintf("%s %s",
		decorateText("▲ TRIANGLE", TriangleMessage),
		decorateText("is generating the triangulated image...", DefaultMessage))

	spinner = utils.NewSpinner(spinnerText, time.Millisecond*200, true)

	// Supported input image file types.
	srcExts := []string{".jpg", ".jpeg", ".png"}

	// Supported output image file types.
	destExts := []string{".jpg", ".jpeg", ".png", ".svg"}

	// Check if source path is a local image or URL.
	if utils.IsValidUrl(*source) {
		src, err := utils.DownloadImage(*source)
		defer src.Close()
		defer os.Remove(src.Name())

		fs, err = src.Stat()
		if err != nil {
			log.Fatalf(
				decorateText("Failed to load the source image: %v", ErrorMessage),
				decorateText(err.Error(), DefaultMessage),
			)
		}
		img, err := os.Open(src.Name())
		if err != nil {
			log.Fatalf(
				decorateText("Unable to open the temporary image file: %v", ErrorMessage),
				decorateText(err.Error(), DefaultMessage),
			)
		}
		imgurl = img
	} else {
		// Check if the source is a pipe name or a regular file.
		if *source == pipeName {
			fs, err = os.Stdin.Stat()
		} else {
			fs, err = os.Stat(*source)
		}
		if err != nil {
			log.Fatalf(
				decorateText("Failed to load the source image: %v", ErrorMessage),
				decorateText(err.Error(), DefaultMessage),
			)
		}
	}
	// start counting the execution time.
	start := time.Now()

	switch mode := fs.Mode(); {
	case mode.IsDir():
		var wg sync.WaitGroup

		// Read destination file or directory.
		_, err := os.Stat(*destination)
		if err != nil {
			err = os.Mkdir(*destination, 0755)
			if err != nil {
				log.Fatalf(
					decorateText("Unable to get dir stats: %v\n", ErrorMessage),
					decorateText(err.Error(), DefaultMessage),
				)
			}
		}

		// Limit the concurrently running workers to maxWorkers.
		if *workers <= 0 || *workers > maxWorkers {
			*workers = runtime.NumCPU()
		}

		// Process recursively the image files from the specified directory concurrently.
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

		// Close the channel after the values are consumed.
		go func() {
			defer close(ch)
			wg.Wait()
		}()

		// Consume the channel values.
		for res := range ch {
			showProcessStatus(res.path, res.triangles, res.points, res.err)
		}

		if err := <-errc; err != nil {
			fmt.Fprintf(os.Stderr, decorateText(err.Error(), ErrorMessage))
		}

	case mode.IsRegular() || mode&os.ModeNamedPipe != 0: // check for regular files or pipe commands
		ext := filepath.Ext(*destination)
		if !inSlice(ext, destExts) && *destination != pipeName {
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
				fmt.Fprintf(os.Stderr, "\n\tYou can access the generated image under the following url: %s ", decorateText(httpAddress, SuccessMessage))

				handler := func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "image/svg+xml")
					w.Write(b)
				}
				http.HandleFunc("/", handler)
				log.Fatal(http.ListenAndServe(strings.TrimPrefix(httpAddress, "http://"), nil))
			}
		})
		flagsCheck = true

		showProcessStatus(*destination, triangles, points, err)
	}

	procTime := time.Since(start)
	if len(os.Args) <= 1 && !flagsCheck {
		log.Fatal("Usage: triangle -in <source> -out <destination>")
	}

	fmt.Fprintf(os.Stderr, "Execution time: %s\n", decorateText(fmt.Sprintf("%s", utils.FormatTime(procTime)), SuccessMessage))
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

			// Get base file extension.
			bfx := filepath.Ext(info.Name())
			for _, ext := range srcExts {
				if ext == bfx {
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
// calls the triangulator processor against the source image
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
// of triangles, points and the error in case if exists.
func processor(in, out string, proc *triangle.Processor, fn triangle.Fn) (
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

	src, dst, err := pathToFile(in, out, proc)
	if err != nil {
		return nil, nil, err
	}

	defer src.(*os.File).Close()
	defer dst.(*os.File).Close()

	// Capture CTRL-C signal and restore the cursor visibility back.
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-signalChan
		func() {
			spinner.RestoreCursor()
			os.Exit(1)
		}()
	}()

	// Start the progress indicator.
	spinner.Start()

	if filepath.Ext(out) == ".svg" {
		svg := &triangle.SVG{
			Title:         "Image triangulator",
			Lines:         []triangle.Line{},
			Description:   "Convert images to computer generated art using delaunay triangulation.",
			StrokeWidth:   proc.StrokeWidth,
			StrokeLineCap: "round", //butt, round, square
			Processor:     *proc,
		}
		_, triangles, points, err = draw(svg, src, dst, fn)
	} else {
		tri := &triangle.Image{
			Processor: *proc,
		}
		_, triangles, points, err = draw(tri, src, dst, fn)
	}
	stopMsg := fmt.Sprintf("%s %s",
		decorateText("▲ TRIANGLE", TriangleMessage),
		decorateText("is generating the triangulated image... ✔", DefaultMessage))
	spinner.StopMsg = stopMsg

	// Stop the progress indicator.
	spinner.Stop()

	return triangles, points, err
}

// draw calls the generic Draw function on each struct which implements this function.
func draw(drawer triangle.Drawer, src interface{}, dst interface{}, fn triangle.Fn) (
	image.Image,
	[]triangle.Triangle,
	[]triangle.Point,
	error,
) {
	return drawer.Draw(src, dst, fn)
}

// pathToFile converts the source and destination paths to readable and writable files.
func pathToFile(in, out string, proc *triangle.Processor) (io.Reader, io.Writer, error) {
	var (
		src io.Reader
		dst io.Writer
		err error
	)
	// Check if the source path is a local image or URL.
	if utils.IsValidUrl(in) {
		src = imgurl
	} else {
		// Check if the source is a pipe name or a regular file.
		if in == pipeName {
			if term.IsTerminal(int(os.Stdin.Fd())) {
				return nil, nil, errors.New("`-` should be used with a pipe for stdin")
			}
			src = os.Stdin
		} else {
			src, err = os.Open(in)
			if err != nil {
				return nil, nil, errors.New(
					fmt.Sprintf("unable to open the source file: %v", err),
				)
			}
		}
	}

	// Check if the destination is a pipe name or a regular file.
	if out == pipeName {
		if term.IsTerminal(int(os.Stdout.Fd())) {
			return nil, nil, errors.New("`-` should be used with a pipe for stdout")
		}
		dst = os.Stdout
	} else {
		dst, err = os.OpenFile(out, os.O_CREATE|os.O_WRONLY, 0755)
		if err != nil {
			return nil, nil, errors.New(
				fmt.Sprintf("unable to create the destination file: %v", err),
			)
		}
	}
	return src, dst, nil
}

// showProcessStatus displays the relavant information about the triangulation process.
func showProcessStatus(
	fname string,
	triangles []triangle.Triangle,
	points []triangle.Point,
	err error,
) {
	if err != nil {
		fmt.Fprintf(os.Stderr,
			decorateText("\nError generating the triangulated image: %s", ErrorMessage),
			decorateText(fmt.Sprintf("\n\tReason: %v\n", err.Error()), DefaultMessage),
		)
		os.Exit(0)
	} else {
		fmt.Fprintf(os.Stderr, fmt.Sprintf("\nTotal number of %s%d %striangles generated out of %s%d %vpoints\n",
			utils.SuccessColor, len(triangles), utils.DefaultColor, utils.SuccessColor, len(points), utils.DefaultColor),
		)
		if fname != pipeName {
			fmt.Fprintf(os.Stderr, fmt.Sprintf("Saved as: %s %s%s\n\n",
				decorateText(filepath.Base(fname), SuccessMessage),
				utils.SuccessColor,
				utils.DefaultColor,
			))
		}
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

// decorateText shows the message types in different colors.
func decorateText(s string, msgType MessageType) string {
	switch msgType {
	case TriangleMessage:
		s = utils.TriangleColor + s
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
