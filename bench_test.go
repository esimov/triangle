package triangle

import (
	"bytes"
	"image"
	_ "image/png"
	"os"
	"testing"
)

func BenchmarkDraw(b *testing.B) {
	buf, err := os.ReadFile("./output//sample_0.png")
	if err != nil {
		b.Skipf("Failed opening test file: %v", err)
	}
	img, _, err := image.Decode(bytes.NewBuffer(buf))
	if err != nil {
		b.Skipf("Failed decoding image: %v", err)
	}
	p := Image{
		Processor: Processor{
			MaxPoints:       2500,
			BlurRadius:      4,
			SobelThreshold:  10,
			PointsThreshold: 20,
			StrokeWidth:     0,
			Wireframe:       0,
		},
	}
	var out image.Image
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _, err = p.Draw(img, out, func() {})
		if err != nil {
			b.Fatalf("Failed drawing triangle benchmark image: %v", err)
		}
	}
}
