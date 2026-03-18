# chromakey

A high-performance, zero-dependency Go package for chroma key background removal and alpha edge erosion. Originally designed for [pseudo3d](https://github.com/paradoxum-wikis/pseudo3d) to process batches of 4K images at extremely high speed.

| Before | After |
| ------ | ----- |
| ![](https://bin.t7ru.link/fol/158781021_0_1773613368753.png) | ![](https://bin.t7ru.link/fol/oneshot.png) |
| ![](https://bin.t7ru.link/fol/accel1.png) | ![](https://bin.t7ru.link/fol/accel.png) |
| ![](https://bin.t7ru.link/fol/onesho2.png?vv) | ![](https://bin.t7ru.link/fol/onesho.png?vv) |

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

- **Remove()**: Removes pixels within the given BT.601 chroma distance of the key color. Fast-paths exist for `*image.RGBA` and `*image.NRGBA`.
- **RemoveRange()**: Soft chroma key using BT.601 chroma distance. Pixels within `minThreshold` become fully transparent, pixels beyond `maxThreshold` are kept, and pixels in between receive proportional transparency and color spill suppression.
- **Erode()**: Removes exactly 1 pixel of alpha along all edges by clearing any opaque pixel adjacent to a fully transparent pixel.

## Performance

The package avoids generic wrappers and aggressively utilizes Go's bounds check elimination patterns and (beautiful) concurrency to process millions of pixels in just a few milliseconds.

Here are some benchmarks run on a mid-range Intel i5-11400H, Go 1.26.0:

| Operation | Image Type | Execution Time | Allocations |
| --- | --- | --- | --- |
| **Remove()** | `*image.NRGBA` | **~0.7 ms** / op | 27 allocs |
| **Remove()** | `*image.RGBA` | **~1.6 ms** / op | 27 allocs |
| **RemoveRange()** | `*image.RGBA` | **~1.9 ms** / op | 27 allocs |
| **Erode()** | `*image.RGBA` | **~7.5 ms**c / op | 2 allocs |
