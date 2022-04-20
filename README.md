
# ![Triangle logo](https://user-images.githubusercontent.com/883386/32769128-4d9625c6-c923-11e7-9a96-030f2f0efff3.png)

[![build](https://github.com/esimov/triangle/actions/workflows/build.yml/badge.svg)](https://github.com/esimov/triangle/actions/workflows/build.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/esimov/triangle.svg)](https://pkg.go.dev/github.com/esimov/triangle)
[![license](https://img.shields.io/github/license/esimov/triangle)](./LICENSE)
[![release](https://img.shields.io/badge/release-v2.0.0-blue.svg)](https://github.com/esimov/triangle/releases/tag/v2.0.0)
[![homebrew](https://img.shields.io/badge/homebrew-v1.2.4-orange.svg)](https://formulae.brew.sh/formula/triangle)

**▲ Triangle** is a tool for generating triangulated image using [delaunay triangulation](https://en.wikipedia.org/wiki/Delaunay_triangulation). It takes a source image and converts it to an abstract image composed of tiles of triangles.

![Sample image](https://github.com/esimov/triangle/blob/master/output/sample_3.png)

### The process
* First the image is blured out to smoth out the sharp pixel edges. The more blured an image is the more diffused the generated output will be.
* Second the resulted image is converted to grayscale mode.
* Then a [sobel](https://en.wikipedia.org/wiki/Sobel_operator) filter operator is applied on the grayscaled image to obtain the image edges. An optional threshold value is applied to filter out the representative pixels of the resulted image.
* A convolution filter operator is applied over the image data in order to adjust its final aspect prior running the delaunay triangulation process.
* Lastly the delaunay algorithm is applied on the pixels obtained from the previous step.

### Features

- [x] Can process recursively whole directories and subdirectories concurrently.
- [x] Supports various image types.
- [x] There is no need to specify the file type, the CLI tool can recognize automatically the input and output file type.
- [x] Can accept image URL as parameter for the `-in` flag.
- [x] Possibility to save the generated image as an **SVG** file.
- [x] The generated SVG file can be accessed from the Web browser directly.
- [x] Clean and intuitive API. The API not only that accepts image files but can also work with image data. This means that the [`Draw`](https://github.com/esimov/triangle/blob/65672f53a60a6a35f5e85bed69e46e97fe2d2def/process.go#L82) method can be invoked even on data streams. Check this [demo](https://github.com/esimov/pigo-wasm-demos#face-triangulator) for reference.
- [x] Support for pipe names (possibility to pipe in and pipe out the source and destination image).

#### TODO
- [ ] Standalone and native GUI application

Head over to this [subtopic](#key-features) to get a better understanding of the supported features.

## Installation and usage
```bash
$ go get -u -f github.com/esimov/triangle/cmd/triangle
$ go install
```
You can also download the binary file from the [releases](https://github.com/esimov/triangle/releases) folder.

## MacOS (Brew) install
The library can be installed via Homebrew too.

```bash
$ brew install triangle
```

## API usage
```go
proc := &triangle.Processor{
	// initialize processor struct
}

img := &triangle.Image{
	Processor: *proc,
}

input, err := os.Open("input.jpg")
if err != nil {
	log.Fatalf("error opening the source file: %v", err)
}

// decode image
src, err := tri.DecodeImage(input)
if err != nil {
	log.Fatalf("error decoding the image: %v", err)
}
res, _, _, err := img.Draw(src, proc, func() {})
if err != nil {
	log.Fatalf("error generating the triangles: %v", err)
}

output, err := os.Open("output.png")
if err != nil {
	log.Fatalf("error opening the destination file: %v", err)
}

// encode image
png.Encode(output, res)

```

## Supported commands

```bash
$ triangle --help
```
The following flags are supported:

| Flag | Default | Description |
| --- | --- | --- |
| `in` | n/a | Source image |
| `out` | n/a | Destination image |
| `bl` | 2 | Blur radius |
| `nf` | 0 | Noise factor |
| `bf` | 1 | Blur factor |
| `ef` | 6 | Edge factor |
| `pr` | 0.075 | Point rate |
| `pth` | 10 | Points threshold |
| `pts` | 2500 | Maximum number of points |
| `so` | 10 | Sobel filter threshold |
| `sl` | false | Use solid stroke color (yes/no) |
| `wf` | 0 | Wireframe mode (0: without stroke, 1: with stroke, 2: stroke only) |
| `sw` | 1 | Stroke width |
| `gr` | false | Output in grayscale mode |
| `web` | false | Open the SVG file in the web browser |
| `bg` | ' ' | Background color (specified as hex value) |
| `cw` | system spec. | Number of files to process concurrently

## Key features

#### Process multiple images from a directory concurrently
The CLI tool also let you process multiple images from a directory **concurrently**. You only need to provide the source and the destination folder by using the `-in` and `-out` flags.

```bash
$ triangle -in <input_folder> -out <output-folder>
```

You can provide also an image file URL for the `-in` flag.
```bash
$ triangle -in <image_url> -out <output-folder>
```

#### Pipe names
The CLI tool accepts also pipe names, which means you can use `stdin` and `stdout` without the need of providing a value for the `-in` and `-out` flag directly since these defaults to `-`. For this reason it's possible to use `curl` for example for downloading an image from the internet and invoke the triangulation process over it directly without the need of getting the image first and calling **▲ Triangle** afterwards.

Here are some examples using pipe names:
```bash
$ curl -s <image_url> | triangle > out.jpg
$ cat input/source.jpg | triangle > out.jpg
$ triangle -in input/source.jpg > out.jpg
$ cat input/source.jpg | triangle -out out.jpg
$ triangle -out out.jpg < input/source.jpg
```

#### Background color
You can specify a background color in case of transparent background images (`.png`) by using the `-bg` flag. This flag accepts a hexadecimal string value. For example setting the flag to `-bg=#ffffff00` will set the alpha channel of the resulted image transparent.

#### Output as image or SVG
By default the output is saved to an image file, but you can export the resulted vertices even to an SVG file. The CLI tool can recognize the output type directly from the file extension. This is a handy addition for those who wish to generate large images without guality loss.

```bash
$ triangle -in samples/input.jpg -out output.svg
```

Using with `-web` flag you can access the generated svg file directly on the web browser.

```bash
$ triangle -in samples/input.jpg -out output.svg -web=true
```

#### Supported output types
The following output file types are supported: `.jpg`, `.jpeg`, `.png`, `.bmp`, `.svg`.

### Tweaks
Setting a lower points threshold, the resulted image will be more like a cubic painting. You can even add a noise factor, generating a more artistic, grainy image.

Here are some examples you can experiment with:
```bash
$ triangle -in samples/input.jpg -out output.png -wf=0 -pts=3500 -stroke=2 -blur=2
$ triangle -in samples/input.jpg -out output.png -wf=2 -pts=5500 -stroke=1 -blur=10
```

## Examples

<a href="https://github.com/esimov/triangle/blob/master/output/sample_3.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_3.png" width=410/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_4.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_4.png" width=410/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_5.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_5.png" width=410/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_6.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_6.png" width=410/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_8.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_8.png" width=410/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_9.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_9.png" width=410/></a>
![Triangle1](https://github.com/esimov/triangle/blob/master/output/sample_0.png)
![Triangle2](https://github.com/esimov/triangle/blob/master/output/sample_1.png)
![Triangle3](https://github.com/esimov/triangle/blob/master/output/sample_11.png)

## License
Copyright © 2018 Endre Simo

This project is under the MIT License. See the [LICENSE](https://github.com/esimov/triangle/blob/master/LICENSE) file for the full license text.
