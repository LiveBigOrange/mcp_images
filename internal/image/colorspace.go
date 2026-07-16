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

func hasAlpha(img image.Image, bounds image.Rectangle) bool {
	dx := bounds.Dx()
	if dx > 32 {
		dx = 32
	}

	midY := bounds.Min.Y + bounds.Dy()/2
	rows := []int{bounds.Min.Y, midY, bounds.Max.Y - 1}

	for _, y := range rows {
		for x := bounds.Min.X; x < bounds.Min.X+dx; x++ {
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
