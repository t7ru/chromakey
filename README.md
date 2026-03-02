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

	// Define the chroma key color and threshold:
	// Threshold is squared Euclidean RGB distance
	// 70.0 is a suitable value for most.
	keyColor := color.RGBA{R: 223, G: 3, B: 223, A: 255} // #DF03DF
	threshold := 70.0

	// Remove the background
	transparentImg := chromakey.Remove(img, keyColor, threshold)

	// Erode 1 pixel of alpha to reduce residue
	// (I personally don't do this step, but it's available if you want to remove residues)
	// Only works on RGBA images; returns a new *image.RGBA.
	if rgba, ok := transparentImg.(*image.RGBA); ok {
		transparentImg = chromakey.Erode(rgba)
	}
}
```

## Features

- **Remove()**: Returns a new image with pixels matching the target color (within the provided RGB distance threshold) made transparent. Fast-paths exist for `*image.RGBA` and `*image.NRGBA`.
- **Erode()**: Removes exactly 1 pixel of alpha along all edges to reduce any color spill residue.

## Performance

The package avoids generic wrappers and aggressively utilizes Go's bounds check elimination patterns to process millions of pixels in just a few milliseconds.

Here are some benchmarks run on a mid-range Intel i5-11400H, Go 1.26.0:

| Operation | Image Type | Execution Time | Allocations |
| :--- | :--- | :--- | :--- |
| **Remove()** | `*image.RGBA` | **~4.8 ms** / op | 2 allocs |
| **Remove()** | `*image.NRGBA` | **~5.6 ms** / op | 2 allocs |
| **Erode()** | `*image.RGBA` | **~7.5 ms** / op | 2 allocs |
