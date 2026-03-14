package chromakey

import (
	"image"
	"image/color"
	"image/draw"
	_ "image/png"
	"math/rand"
	"os"
	"testing"
	"time"
)

// test Remove
func TestRemove(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	green := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}

	for y := range 3 {
		for x := range 3 {
			if x == 1 && y == 1 {
				img.Set(x, y, red)
			} else {
				img.Set(x, y, green)
			}
		}
	}

	result := Remove(img, green, 7000.0)

	if _, _, _, a := result.At(1, 1).RGBA(); a == 0 {
		t.Errorf("Expected center pixel to be preserved (opaque), but it was made transparent")
	}

	if _, _, _, a := result.At(0, 0).RGBA(); a != 0 {
		t.Errorf("Expected corner pixel to be removed (transparent), but it remained opaque")
	}
}

func TestRemove_File(t *testing.T) {
	file, err := os.Open("test.png")
	if err != nil {
		t.Skip("test.png not found")
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("Failed to decode PNG image: %v", err)
	}

	keyColor := color.RGBA{R: 0xDF, G: 0x03, B: 0xDF, A: 0xFF}
	result := Remove(img, keyColor, 7000.0)

	if result == nil || result.Bounds() != img.Bounds() {
		t.Errorf("Remove failed on file input")
	}
}

// test RemoveRange
func TestRemoveRange_Logic(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 3, 3))
	bg := color.RGBA{R: 0, G: 255, B: 0, A: 255}
	fg := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	edge := color.RGBA{R: 100, G: 200, B: 0, A: 255}

	img.Set(0, 0, bg)
	img.Set(1, 1, edge)
	img.Set(2, 2, fg)

	result := RemoveRange(img, bg, 1000.0, 7000.0)

	if _, _, _, a := result.At(0, 0).RGBA(); a != 0 {
		t.Errorf("Expected background pixel to be fully transparent, got alpha %d", a>>8)
	}

	if _, _, _, a := result.At(2, 2).RGBA(); a>>8 != 255 {
		t.Errorf("Expected foreground pixel to be fully opaque, got alpha %d", a>>8)
	}

	_, _, _, edgeA := result.At(1, 1).RGBA()
	edgeA8 := edgeA >> 8
	if edgeA8 == 0 || edgeA8 == 255 {
		t.Errorf("Expected edge pixel to be semi-transparent, got alpha %d", edgeA8)
	}
}

func TestRemoveRange_File(t *testing.T) {
	file, err := os.Open("test.png")
	if err != nil {
		t.Skip("test.png not found")
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		t.Fatalf("Failed to decode PNG image: %v", err)
	}

	keyColor := color.RGBA{R: 0xDF, G: 0x03, B: 0xDF, A: 0xFF}
	result := RemoveRange(img, keyColor, 1000.0, 7000.0)

	if result == nil || result.Bounds() != img.Bounds() {
		t.Errorf("RemoveRange failed on file input")
	}
}

// test Erode
func TestErode(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 5, 5))
	opaque := color.RGBA{R: 255, G: 255, B: 255, A: 255}

	for y := range 3 {
		for x := range 3 {
			img.SetRGBA(x+1, y+1, opaque)
		}
	}

	result := Erode(img)

	if _, _, _, a := result.At(2, 2).RGBA(); a == 0 {
		t.Errorf("Expected absolute center pixel to remain opaque")
	}

	if _, _, _, a := result.At(1, 1).RGBA(); a != 0 {
		t.Errorf("Expected edge of the opaque square to be eroded (transparent)")
	}
}

// junkie
func makeTestRGBA(w, h int, seed int64) *image.RGBA {
	r := rand.New(rand.NewSource(seed))
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			var c color.RGBA
			if r.Intn(10) == 0 {
				c = color.RGBA{0xDF, 0x03, 0xDF, 0xFF}
			} else {
				c = color.RGBA{uint8(r.Intn(256)), uint8(r.Intn(256)), uint8(r.Intn(256)), 0xFF}
			}
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

type fallbackImg struct {
	image.Image
}

// benches remove
func BenchmarkRemove_RGBA_1080p(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	b.ResetTimer()
	for b.Loop() {
		Remove(img, key, 7000.0)
	}
}

func BenchmarkRemove_NRGBA_1080p(b *testing.B) {
	img := image.NewNRGBA(image.Rect(0, 0, 1920, 1080))
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	b.ResetTimer()
	for b.Loop() {
		Remove(img, key, 7000.0)
	}
}

func BenchmarkRemove_Fallback_1080p(b *testing.B) {
	img := fallbackImg{image.NewRGBA(image.Rect(0, 0, 1920, 1080))}
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	b.ResetTimer()
	for b.Loop() {
		Remove(img, key, 7000.0)
	}
}

func BenchmarkRemove_1024x1024(b *testing.B) {
	img := makeTestRGBA(1024, 1024, time.Now().UnixNano())
	key := color.RGBA{0xDF, 0x03, 0xDF, 0xFF}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = Remove(img, key, 7000.0)
	}
}

// benches RemoveRange
func BenchmarkRemoveRange_RGBA_1080p(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	b.ResetTimer()
	for b.Loop() {
		RemoveRange(img, key, 1000.0, 7000.0)
	}
}

func BenchmarkRemoveRange_NRGBA_1080p(b *testing.B) {
	img := image.NewNRGBA(image.Rect(0, 0, 1920, 1080))
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	b.ResetTimer()
	for b.Loop() {
		RemoveRange(img, key, 1000.0, 7000.0)
	}
}

func BenchmarkRemoveRange_Fallback_1080p(b *testing.B) {
	img := fallbackImg{image.NewRGBA(image.Rect(0, 0, 1920, 1080))}
	key := color.RGBA{R: 0, G: 255, B: 0, A: 255}

	b.ResetTimer()
	for b.Loop() {
		RemoveRange(img, key, 1000.0, 7000.0)
	}
}

func BenchmarkRemoveRange_1024x1024(b *testing.B) {
	img := makeTestRGBA(1024, 1024, time.Now().UnixNano())
	key := color.RGBA{0xDF, 0x03, 0xDF, 0xFF}
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = RemoveRange(img, key, 1000.0, 7000.0)
	}
}

// benches Erode
func BenchmarkErode_1080p(b *testing.B) {
	img := image.NewRGBA(image.Rect(0, 0, 1920, 1080))

	opaque := color.RGBA{R: 255, G: 255, B: 255, A: 255}
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i] = opaque.R
		img.Pix[i+1] = opaque.G
		img.Pix[i+2] = opaque.B
		img.Pix[i+3] = opaque.A
	}

	b.ResetTimer()
	for b.Loop() {
		Erode(img)
	}
}

func BenchmarkErode_1024x1024(b *testing.B) {
	src := makeTestRGBA(1024, 1024, 42)
	draw.Draw(src, image.Rect(0, 0, 256, 256), &image.Uniform{C: color.RGBA{0, 0, 0, 0}}, image.Point{}, draw.Src)
	b.ReportAllocs()
	b.ResetTimer()
	for b.Loop() {
		_ = Erode(src)
	}
}
