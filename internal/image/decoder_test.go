package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestDecode_JPEG(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(0, 0, 8, 8))
	for y := 0; y < 8; y++ {
		for x := 0; x < 8; x++ {
			rgba.SetRGBA(x, y, color.RGBA{R: 200, G: 100, B: 50, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatal(err)
	}
	img, format, err := Decode(buf.Bytes(), "jpeg")
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("format = %q, want jpeg", format)
	}
	b := img.Bounds()
	if b.Dx() != 8 || b.Dy() != 8 {
		t.Errorf("decoded dims = %dx%d, want 8x8", b.Dx(), b.Dy())
	}
}

func TestDecode_AutoDetectNoHint(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 80})
	img, format, err := Decode(buf.Bytes(), "")
	if err != nil {
		t.Fatalf("Decode failed: %v", err)
	}
	if format != "jpeg" {
		t.Errorf("auto-detected format = %q, want jpeg", format)
	}
	if img == nil {
		t.Fatal("decoded image is nil")
	}
}

func TestDecode_BadHintFallsBack(t *testing.T) {
	rgba := image.NewRGBA(image.Rect(0, 0, 4, 4))
	var buf bytes.Buffer
	jpeg.Encode(&buf, rgba, &jpeg.Options{Quality: 80})
	// hint says png but data is jpeg — should fall through to image.Decode auto-detect
	img, _, err := Decode(buf.Bytes(), "png")
	if err != nil {
		t.Fatalf("expected fallback decode success, got: %v", err)
	}
	if img == nil {
		t.Fatal("fallback decoded image is nil")
	}
}

func TestDecode_InvalidData(t *testing.T) {
	_, _, err := Decode([]byte("definitely not an image"), "")
	if err == nil {
		t.Fatal("expected error decoding garbage")
	}
}

func TestDecode_UnsupportedHint(t *testing.T) {
	_, _, err := Decode([]byte{0xFF, 0xD8}, "gif")
	if err == nil {
		t.Fatal("expected error for unsupported hinted format with unautodetectable data")
	}
}

func TestDetectFormat_TooShort(t *testing.T) {
	if got := DetectFormat([]byte{0x00, 0x01}); got != "" {
		t.Errorf("DetectFormat(<4 bytes) = %q, want empty", got)
	}
}

func TestDecodeByFormat_Unsupported(t *testing.T) {
	if _, err := decodeByFormat([]byte{0x00}, "xyz"); err == nil {
		t.Fatal("expected error for unsupported format")
	}
}
