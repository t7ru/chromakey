// Package chromakey provides high-performance chroma key
// background removal and edge erosion for images.
package chromakey

import (
	"image"
	"image/color"
	_ "image/png"
	"runtime"
	"sync"
)

// Remove returns a new RGBA image where pixels whose RGB values are within
// the given threshold of keyColor are made fully transparent.
//
// threshold is the squared Euclidean RGB distance (dr^2 + dg^2 + db^2).
func Remove(img image.Image, keyColor color.Color, threshold float64) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)
	thresh := int32(threshold)

	kr32, kg32, kb32, _ := keyColor.RGBA()
	kr := int32(kr32 >> 8)
	kg := int32(kg32 >> 8)
	kb := int32(kb32 >> 8)

	switch src := img.(type) {
	case *image.RGBA:
		width4 := bounds.Dx() * 4
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			i := src.PixOffset(bounds.Min.X, y)
			j := newImg.PixOffset(bounds.Min.X, y)
			rowSrc := src.Pix[i : i+width4]
			rowDst := newImg.Pix[j : j+width4]

			for x := 0; x < width4; x += 4 {
				r, g, b, a := rowSrc[x], rowSrc[x+1], rowSrc[x+2], rowSrc[x+3]
				dr := int32(r) - kr
				dg := int32(g) - kg
				db := int32(b) - kb
				if dr*dr+dg*dg+db*db < thresh {
					rowDst[x] = 0
					rowDst[x+1] = 0
					rowDst[x+2] = 0
					rowDst[x+3] = 0
				} else {
					rowDst[x] = r
					rowDst[x+1] = g
					rowDst[x+2] = b
					rowDst[x+3] = a
				}
			}
		}
	case *image.NRGBA:
		width4 := bounds.Dx() * 4
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			i := src.PixOffset(bounds.Min.X, y)
			j := newImg.PixOffset(bounds.Min.X, y)
			rowSrc := src.Pix[i : i+width4]
			rowDst := newImg.Pix[j : j+width4]

			for x := 0; x < width4; x += 4 {
				a := rowSrc[x+3]
				var r, g, b uint8
				// Convert unpremultiplied NRGBA to premultiplied RGB
				// so distance comparison matches the RGBA behavior.
				switch a {
				case 0xff:
					r, g, b = rowSrc[x], rowSrc[x+1], rowSrc[x+2]
				case 0:
					r, g, b = 0, 0, 0
				default:
					r32 := uint32(rowSrc[x])
					r32 |= r32 << 8
					r32 *= uint32(a)
					r32 /= 0xff

					g32 := uint32(rowSrc[x+1])
					g32 |= g32 << 8
					g32 *= uint32(a)
					g32 /= 0xff

					b32 := uint32(rowSrc[x+2])
					b32 |= b32 << 8
					b32 *= uint32(a)
					b32 /= 0xff

					r, g, b = uint8(r32>>8), uint8(g32>>8), uint8(b32>>8)
				}

				dr := int32(r) - kr
				dg := int32(g) - kg
				db := int32(b) - kb
				if dr*dr+dg*dg+db*db < thresh {
					rowDst[x] = 0
					rowDst[x+1] = 0
					rowDst[x+2] = 0
					rowDst[x+3] = 0
				} else {
					rowDst[x] = r
					rowDst[x+1] = g
					rowDst[x+2] = b
					rowDst[x+3] = a
				}
			}
		}
	default:
		width4 := bounds.Dx() * 4
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			j := newImg.PixOffset(bounds.Min.X, y)
			rowDst := newImg.Pix[j : j+width4]

			dstX := 0
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				r32, g32, b32, a32 := src.At(x, y).RGBA()
				r := uint8(r32 >> 8)
				g := uint8(g32 >> 8)
				b := uint8(b32 >> 8)
				a := uint8(a32 >> 8)

				dr := int32(r) - kr
				dg := int32(g) - kg
				db := int32(b) - kb

				if dr*dr+dg*dg+db*db < thresh {
					rowDst[dstX] = 0
					rowDst[dstX+1] = 0
					rowDst[dstX+2] = 0
					rowDst[dstX+3] = 0
				} else {
					rowDst[dstX] = r
					rowDst[dstX+1] = g
					rowDst[dstX+2] = b
					rowDst[dstX+3] = a
				}
				dstX += 4
			}
		}
	}
	return newImg
}

// Erode removes 1 pixel of alpha by clearing any
// opaque pixel that touches a fully transparent pixel.
func Erode(img *image.RGBA) *image.RGBA {
	bounds := img.Bounds()
	refined := image.NewRGBA(bounds)
	stride := img.Stride
	srcPix := img.Pix
	dstPix := refined.Pix

	width4 := bounds.Dx() * 4
	minX := bounds.Min.X
	minY := bounds.Min.Y
	maxY := bounds.Max.Y

	for y := minY; y < maxY; y++ {
		off := img.PixOffset(minX, y)
		end := off + width4

		rowSrc := srcPix[off:end]
		rowDst := dstPix[off:end]

		for x := 0; x < width4; x += 4 {
			if rowSrc[x+3] == 0 {
				continue
			}
			isEdge := (x > 0 && rowSrc[x-1] == 0) ||
				(x < width4-4 && rowSrc[x+7] == 0) ||
				(y > minY && srcPix[off+x-stride+3] == 0) ||
				(y < maxY-1 && srcPix[off+x+stride+3] == 0)

			if !isEdge {
				copy(rowDst[x:x+4], rowSrc[x:x+4])
			}
		}
	}
	return refined
}

// chromaCb and chromaCr compute BT.601 chroma components (range ~16-240).
// Used for luminance independent keying so dark pixels always have neutral
// chroma and are never falsely pulled into the removal range.
func chromaCb(r, g, b int32) int32 {
	return ((-38*r - 74*g + 112*b + 128) >> 8) + 128
}

func chromaCr(r, g, b int32) int32 {
	return ((112*r - 94*g - 18*b + 128) >> 8) + 128
}

func clampF32(v float32) uint8 {
	return uint8(max(float32(0), min(float32(255), v)))
}

// RemoveRange returns a new RGBA image applying a soft chroma key.
// Pixels within minThreshold of keyColor become fully transparent,
// and those beyond maxThreshold remain unchanged.
// Intermediate pixels will receive proportional
// transparency and color spill suppression.
//
// thresholds are the squared Euclidean distance of BT.601 chroma components (dCb^2 + dCr^2).
func RemoveRange(img image.Image, keyColor color.Color, minThreshold, maxThreshold float64) image.Image {
	bounds := img.Bounds()
	newImg := image.NewRGBA(bounds)

	minThresh := int32(minThreshold)
	maxThresh := int32(maxThreshold)

	threshDiff := maxThresh - minThresh
	if threshDiff <= 0 {
		threshDiff = 1
	}
	threshDiffU32 := uint32(threshDiff)
	threshDiffF := float32(threshDiff)
	recip := uint64(0x100000000) / uint64(threshDiffU32)

	kr32, kg32, kb32, _ := keyColor.RGBA()
	kr := int32(kr32 >> 8)
	kg := int32(kg32 >> 8)
	kb := int32(kb32 >> 8)
	krf := float32(kr)
	kgf := float32(kg)
	kbf := float32(kb)

	keyCb := chromaCb(kr, kg, kb)
	keyCr := chromaCr(kr, kg, kb)

	numWorkers := runtime.NumCPU()
	height := bounds.Dy()
	rowsPerWorker := (height + numWorkers - 1) / numWorkers

	switch src := img.(type) {
	case *image.RGBA:
		var wg sync.WaitGroup
		for w := range numWorkers {
			startY := bounds.Min.Y + w*rowsPerWorker
			endY := min(startY+rowsPerWorker, bounds.Max.Y)
			if startY >= bounds.Max.Y {
				break
			}
			wg.Add(1)
			go func(startY, endY int) {
				defer wg.Done()
				width4 := bounds.Dx() * 4
				for y := startY; y < endY; y++ {
					i := src.PixOffset(bounds.Min.X, y)
					j := newImg.PixOffset(bounds.Min.X, y)
					rowSrc := src.Pix[i : i+width4]
					rowDst := newImg.Pix[j : j+width4]

					for x := 0; x < width4; x += 4 {
						r, g, b, a := rowSrc[x], rowSrc[x+1], rowSrc[x+2], rowSrc[x+3]
						ri, gi, bi := int32(r), int32(g), int32(b)

						cb := chromaCb(ri, gi, bi)
						cr := chromaCr(ri, gi, bi)
						dcb := cb - keyCb
						dcr := cr - keyCr
						dist := dcb*dcb + dcr*dcr

						if dist >= maxThresh {
							rowDst[x], rowDst[x+1], rowDst[x+2], rowDst[x+3] = r, g, b, a
						} else if dist <= minThresh {
							rowDst[x], rowDst[x+1], rowDst[x+2], rowDst[x+3] = 0, 0, 0, 0
						} else {
							ratioNum := uint64(dist - minThresh)
							newA := uint8((uint64(a) * ratioNum * recip) >> 32)
							spillFrac := 1.0 - float32(dist-minThresh)/threshDiffF
							rowDst[x] = clampF32(float32(r) - spillFrac*krf)
							rowDst[x+1] = clampF32(float32(g) - spillFrac*kgf)
							rowDst[x+2] = clampF32(float32(b) - spillFrac*kbf)
							rowDst[x+3] = newA
						}
					}
				}
			}(startY, endY)
		}
		wg.Wait()

	case *image.NRGBA:
		var wg sync.WaitGroup
		for w := range numWorkers {
			startY := bounds.Min.Y + w*rowsPerWorker
			endY := min(startY+rowsPerWorker, bounds.Max.Y)
			if startY >= bounds.Max.Y {
				break
			}
			wg.Add(1)
			go func(startY, endY int) {
				defer wg.Done()
				width4 := bounds.Dx() * 4
				for y := startY; y < endY; y++ {
					i := src.PixOffset(bounds.Min.X, y)
					j := newImg.PixOffset(bounds.Min.X, y)
					rowSrc := src.Pix[i : i+width4]
					rowDst := newImg.Pix[j : j+width4]

					for x := 0; x < width4; x += 4 {
						a := rowSrc[x+3]
						var ri, gi, bi int32

						switch a {
						case 0xff:
							ri, gi, bi = int32(rowSrc[x]), int32(rowSrc[x+1]), int32(rowSrc[x+2])
						case 0:
							rowDst[x], rowDst[x+1], rowDst[x+2], rowDst[x+3] = 0, 0, 0, 0
							continue
						default:
							r32 := uint32(rowSrc[x])
							r32 |= r32 << 8
							r32 = r32 * uint32(a) / 0xff
							g32 := uint32(rowSrc[x+1])
							g32 |= g32 << 8
							g32 = g32 * uint32(a) / 0xff
							b32 := uint32(rowSrc[x+2])
							b32 |= b32 << 8
							b32 = b32 * uint32(a) / 0xff
							ri, gi, bi = int32(r32>>8), int32(g32>>8), int32(b32>>8)
						}

						cb := chromaCb(ri, gi, bi)
						cr := chromaCr(ri, gi, bi)
						dcb := cb - keyCb
						dcr := cr - keyCr
						dist := dcb*dcb + dcr*dcr

						if dist >= maxThresh {
							rowDst[x], rowDst[x+1], rowDst[x+2], rowDst[x+3] = rowSrc[x], rowSrc[x+1], rowSrc[x+2], a
						} else if dist <= minThresh {
							rowDst[x], rowDst[x+1], rowDst[x+2], rowDst[x+3] = 0, 0, 0, 0
						} else {
							ratioNum := uint64(dist - minThresh)
							newA := uint8((uint64(a) * ratioNum * recip) >> 32)
							spillFrac := 1.0 - float32(dist-minThresh)/threshDiffF
							rowDst[x] = clampF32(float32(rowSrc[x]) - spillFrac*krf)
							rowDst[x+1] = clampF32(float32(rowSrc[x+1]) - spillFrac*kgf)
							rowDst[x+2] = clampF32(float32(rowSrc[x+2]) - spillFrac*kbf)
							rowDst[x+3] = newA
						}
					}
				}
			}(startY, endY)
		}
		wg.Wait()

	default:
		var wg sync.WaitGroup
		for w := range numWorkers {
			startY := bounds.Min.Y + w*rowsPerWorker
			endY := min(startY+rowsPerWorker, bounds.Max.Y)
			if startY >= bounds.Max.Y {
				break
			}
			wg.Add(1)
			go func(startY, endY int) {
				defer wg.Done()
				width4 := bounds.Dx() * 4
				for y := startY; y < endY; y++ {
					j := newImg.PixOffset(bounds.Min.X, y)
					rowDst := newImg.Pix[j : j+width4]
					dstX := 0
					for x := bounds.Min.X; x < bounds.Max.X; x++ {
						r32, g32, b32, a32 := src.At(x, y).RGBA()
						r := uint8(r32 >> 8)
						g := uint8(g32 >> 8)
						b := uint8(b32 >> 8)
						a := uint8(a32 >> 8)

						cb := chromaCb(int32(r), int32(g), int32(b))
						cr := chromaCr(int32(r), int32(g), int32(b))
						dcb := cb - keyCb
						dcr := cr - keyCr
						dist := dcb*dcb + dcr*dcr

						if dist >= maxThresh {
							rowDst[dstX], rowDst[dstX+1], rowDst[dstX+2], rowDst[dstX+3] = r, g, b, a
						} else if dist <= minThresh {
							rowDst[dstX], rowDst[dstX+1], rowDst[dstX+2], rowDst[dstX+3] = 0, 0, 0, 0
						} else {
							ratioNum := uint64(dist - minThresh)
							newA := uint8((uint64(a) * ratioNum * recip) >> 32)
							spillFrac := 1.0 - float32(dist-minThresh)/threshDiffF
							rowDst[dstX] = clampF32(float32(r) - spillFrac*krf)
							rowDst[dstX+1] = clampF32(float32(g) - spillFrac*kgf)
							rowDst[dstX+2] = clampF32(float32(b) - spillFrac*kbf)
							rowDst[dstX+3] = newA
						}
						dstX += 4
					}
				}
			}(startY, endY)
		}
		wg.Wait()
	}
	return newImg
}
