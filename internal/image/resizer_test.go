package image

import (
	"image"
	"testing"
)

func TestResizeIfNeededWithMax_NoResize(t *testing.T) {
	img := createTestImage(512, 512)
	if got := ResizeIfNeededWithMax(img, 1000, 1024); got != img {
		t.Error("image below threshold should not be resized")
	}
}

func TestResizeIfNeededWithMax_DefaultFallback(t *testing.T) {
	img := createTestImage(4096, 3072)
	// maxDim <= 0 should fall back to default maxDimension (2048)
	got := ResizeIfNeededWithMax(img, 1024, 0)
	b := got.Bounds()
	if b.Dx() != 2048 {
		t.Errorf("fallback dims = %dx%d, want 2048x1536", b.Dx(), b.Dy())
	}
}

func TestResizeIfNeededWithMax_Shrinks(t *testing.T) {
	img := createTestImage(2048, 1024)
	got := ResizeIfNeededWithMax(img, 1024, 512)
	b := got.Bounds()
	if b.Dx() != 512 || b.Dy() != 256 {
		t.Errorf("resize dims = %dx%d, want 512x256", b.Dx(), b.Dy())
	}
}

func TestResizeIfNeededWithMax_BoundaryNoResize(t *testing.T) {
	img := createTestImage(512, 512)
	if got := ResizeIfNeededWithMax(img, 1000, 512); got != img {
		t.Error("image exactly at maxDim should not be resized")
	}
}

func TestNewProcessor_Defaults(t *testing.T) {
	p := NewProcessor()
	if p.MaxDimension != maxDimension {
		t.Errorf("default MaxDimension = %d, want %d", p.MaxDimension, maxDimension)
	}
}

func TestNewProcessorWithMax(t *testing.T) {
	p := NewProcessorWithMax(1024)
	if p.MaxDimension != 1024 {
		t.Errorf("MaxDimension = %d, want 1024", p.MaxDimension)
	}
	if p2 := NewProcessorWithMax(0); p2.MaxDimension != maxDimension {
		t.Errorf("MaxDimension(0) = %d, want default %d", p2.MaxDimension, maxDimension)
	}
}

func TestApplyEXIFOrientation_OrientationCases(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for _, o := range []uint16{2, 3, 4, 5, 6, 7, 8} {
		got := ApplyEXIFOrientation(img, buildEXIFJPEG(o))
		if got == nil {
			t.Errorf("orientation %d produced nil image", o)
		}
	}
}
