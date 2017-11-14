
# ![Triangle logo](https://user-images.githubusercontent.com/883386/32769128-4d9625c6-c923-11e7-9a96-030f2f0efff3.png)

Triangle is a tool to create image arts using the [delaunay triangulation](https://en.wikipedia.org/wiki/Delaunay_triangulation) technique. It takes an image as input and it converts to abstract image composed of tiles of triangles.

![Sample image](https://github.com/esimov/triangle/blob/master/output/sample_3.png)

### The technique
* First the image is blured out to smothen the sharp pixel edges. The more blured an image is the more diffused the generated output will be.
* Second the resulted image is converted to grayscale mode. 
* Then a [sobel](https://en.wikipedia.org/wiki/Sobel_operator) filter operator is applied on the grayscaled image to obtain the image edges. An optional threshold value is applied to filter out the representative pixels of the resulting image.
* Lastly we apply the delaunay algorithm on the pixels obtained from the previous step.

```go
blur = tri.Stackblur(img, uint32(width), uint32(height), uint32(*blurRadius))
gray = tri.Grayscale(blur)
sobel = tri.SobelFilter(gray, float64(*sobelThreshold))
points = tri.GetEdgePoints(sobel, *pointsThreshold, *maxPoints)

triangles = delaunay.Init(width, height).Insert(points).GetTriangles()
```
## Installation and usage
```bash
$ go get github.com/esimov/triangle/cmd/triangle
$ go install
```
### Supported commands

```bash
$ triangle --help
```
The following flags are supported:

| Flag | Default | Description |
| --- | --- | --- |
| `in` | n/a | Input file |
| `out` | n/a | Output file |
| `blur` | 4 | Blur radius |
| `max` | 2500 | Maximum number of points |
| `noise` | 0 | Noise factor |
| `points` | 20 | Points threshold |
| `sobel` | 10 | Sobel filter threshold |
| `solid` | false | Solid line color |
| `wireframe` | 0 | Wireframe mode (without,with,both) |
| `width` | 1 | Wireframe line width |
| `gray` | false | Convert to grayscale |

Setting a lower points value, the resulted image will be more like a cubic painting. You can even add a noise factor, giving a more artistic, despeckle like result for the final image.  

In case the `gray` filter is set as `true` the resulting triangulated image will be converted to grayscale mode.

Here are some examples you can experiment with:
```bash
$ triangle -in samples/clown_4.jpg -out output.png -wireframe=0 -max=3500 -width=2 -blur=2
$ triangle -in samples/clown_4.jpg -out output.png -wireframe=2 -max=5500 -width=1 -blur=10
```

You can even transform multiple images from a specific folder with one command, concurently, by specifying as `-in` flag the source folder and as `-out` flag the destination folder. Example:

```bash
$ triangle -in ./samples/ -out ./ouput -wireframe=0 -max=3500 -width=2 -blur=2 -noise=4
```

Below are some of the generated images:

<a href="https://github.com/esimov/triangle/blob/master/output/sample_3.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_3.png" width=420/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_4.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_4.png" width=420/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_5.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_5.png" width=420/></a>
<a href="https://github.com/esimov/triangle/blob/master/output/sample_6.png"><img src="https://github.com/esimov/triangle/blob/master/output/sample_6.png" width=420/></a>
![Sample_0](https://github.com/esimov/triangle/blob/master/output/sample_0.png)
![Sample_1](https://github.com/esimov/triangle/blob/master/output/sample_1.png)
![Sample_11](https://github.com/esimov/triangle/blob/master/output/sample_11.png)
![Sample_8](https://github.com/esimov/triangle/blob/master/output/sample_8.png)


## License

This project is under the MIT License. See the [LICENSE](https://github.com/esimov/triangle/blob/master/LICENSE) file for the full license text.
