package main

import (
	_ "image/png"
	_ "image/jpeg"
	"os"
	"image"
	"image/png"
	"log"
	"time"
	"fmt"
	"github.com/esimov/triangulator"
)

func main() {
	file, err := os.Open("Valve_original.png")
	defer file.Close()
	src, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}
	start := time.Now()
	blur := triangulator.Stackblur(src, uint32(src.Bounds().Dx()), uint32(src.Bounds().Dy()), 2)
	gray := triangulator.Grayscale(blur)
	sobel := triangulator.Sobel(gray, 2)
	triangulator.GetEdgePoints(sobel, 10)

	end := time.Since(start)
	fmt.Println(end)

	fq, err := os.Create("output.png")
	defer fq.Close()

	if err = png.Encode(fq, sobel); err != nil {
		log.Fatal(err)
	}
}