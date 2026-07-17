package image

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestProcessor_Process_JPEG(t *testing.T) {
	img := createTestImage(100, 100)
	data := encodeTestJPEG(img)

	p := NewProcessor()
	dataURI, err := p.Process(data, "jpeg")
	if err != nil {
		t.Fatalf("Process failed: %v", err)
	}
	if len(dataURI) == 0 {
		t.Fatal("dataURI is empty")
	}
}

func TestProcessor_Process_InvalidData(t *testing.T) {
	p := NewProcessor()
	_, err := p.Process([]byte("not an image"), "")
	if err == nil {
		t.Fatal("expected error for invalid data")
	}
}

func TestResizeIfNeeded_NoResize(t *testing.T) {
	img := createTestImage(100, 100)
	result := ResizeIfNeeded(img, 1000)
	if result != img {
		t.Error("should not resize small image")
	}
}

func TestResizeIfNeeded_LargeImage(t *testing.T) {
	img := createTestImage(4096, 3072)
	result := ResizeIfNeeded(img, 10*1024*1024)
	bounds := result.Bounds()
	if bounds.Dx() != 2048 {
		t.Errorf("width = %d, want 2048", bounds.Dx())
	}
	if bounds.Dy() != 1536 {
		t.Errorf("height = %d, want 1536", bounds.Dy())
	}
}

func TestResizeIfNeeded_BoundaryImage(t *testing.T) {
	img := createTestImage(2048, 2048)
	result := ResizeIfNeeded(img, 1*1024*1024)
	if result != img {
		t.Error("2048x2048 should not be resized")
	}
}

func TestEncodeToJPEGBase64(t *testing.T) {
	img := createTestImage(10, 10)
	dataURI, err := EncodeToJPEGBase64(img)
	if err != nil {
		t.Fatalf("EncodeToJPEGBase64 failed: %v", err)
	}
	if len(dataURI) == 0 {
		t.Fatal("dataURI is empty")
	}
}

func TestDetectFormat(t *testing.T) {
	tests := []struct {
		data     []byte
		expected string
	}{
		{[]byte{0xFF, 0xD8, 0xFF, 0xE0}, "jpeg"},
		{[]byte{0x89, 0x50, 0x4E, 0x47}, "png"},
		{[]byte{0x42, 0x4D, 0x00, 0x00}, "bmp"},
		{[]byte{0x49, 0x49, 0x2A, 0x00}, "tiff"},
		{[]byte{0x52, 0x49, 0x46, 0x46, 0x00, 0x00, 0x00, 0x00, 0x57, 0x45, 0x42, 0x50}, "webp"},
		{[]byte{0x00, 0x00, 0x00, 0x00}, ""},
	}
	for _, tt := range tests {
		got := DetectFormat(tt.data)
		if got != tt.expected {
			t.Errorf("DetectFormat(%v) = %q, want %q", tt.data[:4], got, tt.expected)
		}
	}
}

func createTestImage(w, h int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetRGBA(x, y, color.RGBA{R: 128, G: 128, B: 128, A: 255})
		}
	}
	return img
}

func encodeTestJPEG(img image.Image) []byte {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		panic(err)
	}
	return buf.Bytes()
}
