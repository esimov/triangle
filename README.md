
# ![Triangle logo](https://user-images.githubusercontent.com/883386/32769128-4d9625c6-c923-11e7-9a96-030f2f0efff3.png)

[![Build Status](https://travis-ci.org/esimov/triangle.svg?branch=master)](https://travis-ci.org/esimov/triangle)
[![GoDoc](https://godoc.org/github.com/golang/gddo?status.svg)](https://godoc.org/github.com/esimov/triangle)
[![license](https://img.shields.io/github/license/esimov/triangle)](./LICENSE)
[![release](https://img.shields.io/badge/release-v1.2.0-blue.svg)](https://github.com/esimov/triangle/releases/tag/v1.2.0)
[![homebrew](https://img.shields.io/badge/homebrew-v1.1.2-orange.svg)](https://github.com/esimov/homebrew-triangle)

**▲ Triangle** is a tool to generate triangulated image using [delaunay triangulation](https://en.wikipedia.org/wiki/Delaunay_triangulation). It takes a source image and converts it to an abstract image composed of tiles of triangles.

![Sample image](https://github.com/esimov/triangle/blob/master/output/sample_3.png)

### The process
* First the image is blured out to smothen sharp pixel edges. The more blured an image is the more diffused the generated output will be.
* Second the resulted image is converted to grayscale mode. 
* Then a [sobel](https://en.wikipedia.org/wiki/Sobel_operator) filter operator is applied on the grayscaled image to obtain the image edges. An optional threshold value is applied to filter out the representative pixels of the resulted image.
* Lastly the delaunay algorithm is applied on the pixels obtained from the previous step.

```go
blur = tri.Stackblur(img, uint32(width), uint32(height), uint32(*blurRadius))
gray = tri.Grayscale(blur)
sobel = tri.SobelFilter(gray, float64(*sobelThreshold))
points = tri.GetEdgePoints(sobel, *pointsThreshold, *maxPoints)

triangles = delaunay.Init(width, height).Insert(points).GetTriangles()
```
## Installation and usage
```bash
$ go get -u -f github.com/esimov/triangle/cmd/triangle
$ go install
```
## MacOS (Brew) install
The library can be installed via Homebrew too or by downloading the binary file from the [releases](https://github.com/esimov/triangle/releases) folder.

```bash
$ brew install triangle
```

### Supported commands

```bash
$ triangle --help
```
The following flags are supported:

| Flag | Default | Description |
| --- | --- | --- |
| `in` | n/a | Source image |
| `out` | n/a | Destination image |
| `blur` | 4 | Blur radius |
| `pts` | 2500 | Maximum number of points |
| `noise` | 0 | Noise factor |
| `th` | 20 | Points threshold |
| `sobel` | 10 | Sobel filter threshold |
| `solid` | false | Use solid stroke color (yes/no) |
| `wf` | 0 | Wireframe mode (0: without stroke, 1: with stroke, 2: stroke only) |
| `stroke` | 1 | Stroke width |
| `gray` | false | Output in grayscale mode |
| `web` | false | Open the SVG file in the web browser |
| `bg` | ' ' | Background color (specified as hex value) |
| `w` | system spec. | Number of files to process concurrently (workers)

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
The following output types are supported: `.jpg`, `.jpeg`, `.png`, `.svg`.

### Process multiple images from a directory concurrently
The CLI tool also let you process multiple images from a directory **concurrently**. You only need to provide the source and the destination folder by using the `-in` and `-out` flags.

```bash
$ triangle -in <input_folder> -out <output-folder>
```
### Tweaks
Setting a lower points threshold, the resulted image will be more like a cubic painting. You can even add a noise factor, generating a more artistic, grainy image.

Here are some examples you can experiment with:
```bash
$ triangle -in samples/input.jpg -out output.png -wf=0 -pts=3500 -stroke=2 -blur=2
$ triangle -in samples/input.jpg -out output.png -wf=2 -pts=5500 -stroke=1 -blur=10
```

### Examples

<a href="https://github.com/esimov/triangle/blob/master/output/sample_3.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_3.png" width=410/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_4.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_4.png" width=410/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_5.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_5.png" width=410/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_6.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_6.png" width=410/></a>
![Sample_0](https://github.com/esimov/triangle/blob/master/output/sample_0.png)
![Sample_1](https://github.com/esimov/triangle/blob/master/output/sample_1.png)
![Sample_11](https://github.com/esimov/triangle/blob/master/output/sample_11.png)
![Sample_8](https://github.com/esimov/triangle/blob/master/output/sample_8.png)


## License
Copyright © 2018 Endre Simo

This project is under the MIT License. See the [LICENSE](https://github.com/esimov/triangle/blob/master/LICENSE) file for the full license text.
