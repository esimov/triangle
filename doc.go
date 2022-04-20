/*
Package triangle is an image processing library which converts images to computer generated art using delaunay triangulation.

The package provides a command line utility supporting various customization options.
Check the supported commands by typing:

	$ triangle --help

Using Go interfaces the API can expose the result either as raster or vector type.

Example to generate triangulated image and output the result as a raster type:

	package main

	import (
		"fmt"
		"github.com/esimov/triangle/v2"
	)

	func main() {
		p := &triangle.Processor{
			// Initialize struct variables
		}

		img := &triangle.Image{*p}
		_, _, _, err := img.Draw(srcImg, p, func() {})
		if err != nil {
			fmt.Printf("Error on triangulation process: %s", err.Error())
		}
	}


Example to generate triangulated image and output the result as SVG:

	package main

	import (
		"fmt"
		"github.com/esimov/triangle/v2"
	)

	func main() {
		p := &triangle.Processor{
			// Initialize struct variables
		}

		svg := &triangle.SVG{
			Title:         "Delaunay image triangulator",
			Lines:         []triangle.Line{},
			Description:   "Convert images to computer generated art using delaunay triangulation.",
			StrokeWidth:   p.StrokeWidth,
			StrokeLineCap: "round", //butt, round, square
			Processor:     *p,
		}
		_, _, _, err := svg.Draw(srcImg, p, func() {
			// Call the closure function
		})
		if err != nil {
			fmt.Printf("Error on triangulation process: %s", err.Error())
		}
	}

*/
package triangle
