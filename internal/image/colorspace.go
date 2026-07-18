package image

import (
	"image"
	"image/draw"
)

func convertToRGB(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		if isOpaqueRGBA(rgba) {
			return rgba
		}
		return blendAlphaRGBA(rgba)
	}

	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)

	if hasAlpha(img, bounds) {
		return blendAlphaRGBA(rgba)
	}

	return rgba
}

func isOpaqueRGBA(rgba *image.RGBA) bool {
	pix := rgba.Pix
	for i := 3; i < len(pix); i += 4 {
		if pix[i] != 255 {
			return false
		}
	}
	return true
}

// hasAlpha reports whether img has any non-opaque pixels. It uses fast Pix-based
// access for the image types produced by the std decoders (RGBA/NRGBA/Paletted/Gray)
// and only falls back to the per-pixel At() loop for exotic image.Image impls.
func hasAlpha(img image.Image, bounds image.Rectangle) bool {
	switch m := img.(type) {
	case *image.NRGBA:
		pix := m.Pix
		for i := 3; i < len(pix); i += 4 {
			if pix[i] != 255 {
				return true
			}
		}
		return false
	case *image.Paletted:
		for _, i := range m.Pix {
			if int(i) < len(m.Palette) {
				_, _, _, a := m.Palette[i].RGBA()
				if a != 0xffff {
					return true
				}
			}
		}
		return false
	case *image.Gray, *image.Gray16, *image.YCbCr:
		return false
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a != 65535 {
				return true
			}
		}
	}
	return false
}

func blendAlphaRGBA(rgba *image.RGBA) *image.RGBA {
	pix := rgba.Pix
	for i := 0; i < len(pix); i += 4 {
		a := pix[i+3]
		if a == 0 {
			pix[i] = 255
			pix[i+1] = 255
			pix[i+2] = 255
			pix[i+3] = 255
			continue
		}
		if a == 255 {
			continue
		}
		alpha := float32(a) / 255.0
		invAlpha := 1.0 - alpha
		pix[i] = uint8(float32(pix[i])*alpha + 255.0*invAlpha)
		pix[i+1] = uint8(float32(pix[i+1])*alpha + 255.0*invAlpha)
		pix[i+2] = uint8(float32(pix[i+2])*alpha + 255.0*invAlpha)
		pix[i+3] = 255
	}
	return rgba
}
