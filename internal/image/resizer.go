package image

import (
	"image"

	"github.com/disintegration/imaging"
)

const maxDimension = 2048

func ResizeIfNeeded(img image.Image, dataSize int) image.Image {
	return ResizeIfNeededWithMax(img, dataSize, maxDimension)
}

func ResizeIfNeededWithMax(img image.Image, dataSize int, maxDim int) image.Image {
	if maxDim <= 0 {
		maxDim = maxDimension
	}
	bounds := img.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	if w <= maxDim && h <= maxDim && dataSize <= 5*1024*1024 {
		return img
	}

	var newW, newH int
	if w > h {
		newW = maxDim
		newH = h * maxDim / w
	} else {
		newH = maxDim
		newW = w * maxDim / h
	}

	if newW > w {
		newW = w
	}
	if newH > h {
		newH = h
	}

	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}

	if newW == w && newH == h {
		return img
	}

	return imaging.Resize(img, newW, newH, imaging.Lanczos)
}
