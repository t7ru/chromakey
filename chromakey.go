// Package chromakey provides high-performance chroma key
// background removal and edge erosion for images.
package chromakey

import (
	"image"
	"image/color"
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
