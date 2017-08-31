package main

import (
	_ "image/png"
	_ "image/jpeg"
	"os"
	"image"
	"image/png"
	"log"
)

func main() {
	file, err := os.Open("lena_512.png")
	defer file.Close()
	src, _, err := image.Decode(file)
	if err != nil {
		panic(err)
	}

	dst := StackBlur(src, uint32(src.Bounds().Dx()), uint32(src.Bounds().Dy()), 10)

	fq, err := os.Create("output.png")
	defer fq.Close()

	if err = png.Encode(fq, dst); err != nil {
		log.Fatal(err)
	}
}