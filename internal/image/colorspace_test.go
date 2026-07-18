package image

import (
	"image"
	"image/color"
	"testing"
)

func TestConvertToRGB_OpaqueRGBA(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			rgba.SetRGBA(x, y, color.RGBA{R: 100, G: 150, B: 200, A: 255})
		}
	}
	out := convertToRGB(rgba)
	if out != rgba {
		t.Error("opaque *image.RGBA should be returned as-is")
	}
}

func TestConvertToRGB_RGBAWithAlpha(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(0, 0, 2, 2))
	rgba.SetRGBA(0, 0, color.RGBA{R: 0, G: 0, B: 0, A: 128})
	out := convertToRGB(rgba)
	if out != rgba {
		t.Error("convertToRGB should mutate and return the same *image.RGBA when blending")
	}
	// blended pixel should be fully opaque
	if _, _, _, a := out.At(0, 0).RGBA(); a != 0xffff {
		t.Errorf("blended alpha = %d, want 0xffff", a)
	}
}

func TestConvertToRGB_NRGBAToRGB(t *testing.T) {
	nrgba := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	nrgba.SetNRGBA(0, 0, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
	nrgba.SetNRGBA(1, 1, color.NRGBA{R: 0, G: 0, B: 0, A: 0})
	out := convertToRGB(nrgba)
	if out.Bounds().Dx() != 2 {
		t.Errorf("bounds dx = %d, want 2", out.Bounds().Dx())
	}
	// transparent pixel should be composited over white
	r, g, b, a := out.At(1, 1).RGBA()
	if a != 0xffff {
		t.Errorf("alpha = %d, want 0xffff", a)
	}
	if r != 0xffff || g != 0xffff || b != 0xffff {
		t.Errorf("transparent composited = (%d,%d,%d), want white", r, g, b)
	}
}

func TestHasAlpha_NRGBA(t *testing.T) {
	nrgba := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	nrgba.SetNRGBA(0, 0, color.NRGBA{A: 255})
	nrgba.SetNRGBA(1, 1, color.NRGBA{A: 128})
	if !hasAlpha(nrgba, nrgba.Bounds()) {
		t.Error("NRGBA with partial alpha should report hasAlpha=true")
	}
}

func TestHasAlpha_NRGBA_Opaque(t *testing.T) {
	nrgba := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			nrgba.SetNRGBA(x, y, color.NRGBA{A: 255})
		}
	}
	if hasAlpha(nrgba, nrgba.Bounds()) {
		t.Error("fully opaque NRGBA should report hasAlpha=false")
	}
}

func TestHasAlpha_Gray(t *testing.T) {
	gray := image.NewGray(image.Rect(0, 0, 4, 4))
	if hasAlpha(gray, gray.Bounds()) {
		t.Error("*image.Gray should never report hasAlpha")
	}
}

func TestHasAlpha_PalettedWithAlpha(t *testing.T) {
	pal := color.Palette{
		color.RGBA{A: 255},
		color.RGBA{A: 0},
	}
	p := image.NewPaletted(image.Rect(0, 0, 2, 2), pal)
	p.SetColorIndex(0, 0, 0)
	p.SetColorIndex(1, 1, 1)
	if !hasAlpha(p, p.Bounds()) {
		t.Error("paletted image containing transparent index should report hasAlpha=true")
	}
}

func TestIsOpaqueRGBA(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		for x := 0; x < 2; x++ {
			rgba.SetRGBA(x, y, color.RGBA{A: 255})
		}
	}
	if !isOpaqueRGBA(rgba) {
		t.Error("all-opaque RGBA should be opaque")
	}
	rgba.SetRGBA(0, 0, color.RGBA{A: 200})
	if isOpaqueRGBA(rgba) {
		t.Error("RGBA with one non-opaque pixel should not be opaque")
	}
}
