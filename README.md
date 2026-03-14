# chromakey

A high-performance, zero-dependency Go package for chroma key background removal and alpha edge erosion. Originally designed for [pseudo3d](https://github.com/paradoxum-wikis/pseudo3d) to process batches of 4K images at extremely high speed.

## Installation

```bash
go get github.com/t7ru/chromakey@latest
```

## Usage

```go
package main

import (
	"image"
	"image/color"
	_ "image/png"
	"log"
	"os"

	"github.com/t7ru/chromakey"
)

func main() {
	// Load an image
	file, err := os.Open("input.png")
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Fatal(err)
	}

	// Define the chroma key color
	keyColor := color.RGBA{R: 223, G: 3, B: 223, A: 255} // #DF03DF

	// Option 1: Hard Removal
	// Threshold is squared Euclidean RGB distance.
	transparentImg := chromakey.Remove(img, keyColor, 7000.0)

	// Option 2: Range Removal
	// Threshold is YCbCr chroma distance.
	transparentImg = chromakey.RemoveRange(img, keyColor, 1000.0, 7000.0)

	// Erode 1 pixel of alpha to reduce residue
	// (I personally don't do this step, but it's available if you want to remove residues)
	// Only works on RGBA images; returns a new *image.RGBA.
	if rgba, ok := transparentImg.(*image.RGBA); ok {
		transparentImg = chromakey.Erode(rgba)
	}
}
```

## Functions

- **Remove()**: Returns a new image with pixels matching the target color (within the provided RGB distance threshold) made transparent. Fast-paths exist for `*image.RGBA` and `*image.NRGBA`.
- **RemoveRange()**: Applies a soft chroma key using YCbCr distance. Supports semi-transparency gradients and includes color spill suppression.
- **Erode()**: Removes exactly 1 pixel of alpha along all edges by clearing any opaque pixel that touches a fully transparent pixel.

## Performance

The package avoids generic wrappers and aggressively utilizes Go's bounds check elimination patterns and (beautiful) concurrency to process millions of pixels in just a few milliseconds.

Here are some benchmarks run on a mid-range Intel i5-11400H, Go 1.26.0:

| Operation | Image Type | Execution Time | Allocations |
| --- | --- | --- | --- |
| **Remove()** | `*image.NRGBA` | **~0.7 ms** / op | 27 allocs |
| **Remove()** | `*image.RGBA` | **~1.6 ms** / op | 27 allocs |
| **RemoveRange()** | `*image.RGBA` | **~1.9 ms** / op | 27 allocs |
| **Erode()** | `*image.RGBA` | **~7.5 ms** / op | 2 allocs |
