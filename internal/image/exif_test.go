package image

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"
)

// buildEXIFJPEG constructs a minimal JPEG byte stream containing an APP1 EXIF
// segment with the given Orientation value. The bytes are enough to feed
// getEXIFOrientation (which only scans JPEG segments), even without valid
// entropy-coded image data.
func buildEXIFJPEG(orientation uint16) []byte {
	var tiff bytes.Buffer
	// Byte order
	tiff.WriteString("II")
	// Magic 0x002A
	binary.Write(&tiff, binary.LittleEndian, uint16(0x002A))
	// Offset to IFD0
	binary.Write(&tiff, binary.LittleEndian, uint32(8))
	// IFD0: 1 entry
	binary.Write(&tiff, binary.LittleEndian, uint16(1))
	// Entry: tag=Orientation(0x0112), type=SHORT(3), count=1, value=orientation
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0112))
	binary.Write(&tiff, binary.LittleEndian, uint16(3))
	binary.Write(&tiff, binary.LittleEndian, uint32(1))
	// value field is 4 bytes; orientation occupies first 2 bytes (LE)
	var val [4]byte
	binary.LittleEndian.PutUint16(val[:2], orientation)
	tiff.Write(val[:])
	// Next IFD offset = 0
	binary.Write(&tiff, binary.LittleEndian, uint32(0))

	var app1 bytes.Buffer
	app1.WriteString("Exif\x00\x00")
	app1.Write(tiff.Bytes())

	var out bytes.Buffer
	out.WriteString("\xFF\xD8") // SOI
	out.WriteByte(0xFF)
	out.WriteByte(0xE1) // APP1
	// length includes 2 bytes of length itself + payload
	binary.Write(&out, binary.BigEndian, uint16(2+app1.Len()))
	out.Write(app1.Bytes())
	// EOI to terminate
	out.WriteString("\xFF\xD9")
	return out.Bytes()
}

func TestGetEXIFOrientation(t *testing.T) {
	cases := []uint16{1, 2, 3, 4, 5, 6, 7, 8}
	for _, want := range cases {
		data := buildEXIFJPEG(want)
		got := getEXIFOrientation(data)
		if got != int(want) {
			t.Errorf("orientation=%d: got %d, want %d", want, got, want)
		}
	}
}

func TestGetEXIFOrientation_NoEXIF(t *testing.T) {
	// SOI + EOI only
	data := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	if got := getEXIFOrientation(data); got != 0 {
		t.Errorf("got %d, want 0 for JPEG without EXIF", got)
	}
}

func TestGetEXIFOrientation_NotJPEG(t *testing.T) {
	if got := getEXIFOrientation([]byte("not a jpeg")); got != 0 {
		t.Errorf("got %d, want 0 for non-JPEG", got)
	}
}

func TestGetEXIFOrientation_BigEndian(t *testing.T) {
	var tiff bytes.Buffer
	tiff.WriteString("MM")
	binary.Write(&tiff, binary.BigEndian, uint16(0x002A))
	binary.Write(&tiff, binary.BigEndian, uint32(8))
	binary.Write(&tiff, binary.BigEndian, uint16(1))
	binary.Write(&tiff, binary.BigEndian, uint16(0x0112))
	binary.Write(&tiff, binary.BigEndian, uint16(3))
	binary.Write(&tiff, binary.BigEndian, uint32(1))
	var val [4]byte
	binary.BigEndian.PutUint16(val[:2], 6)
	tiff.Write(val[:])
	binary.Write(&tiff, binary.BigEndian, uint32(0))

	var app1 bytes.Buffer
	app1.WriteString("Exif\x00\x00")
	app1.Write(tiff.Bytes())

	var out bytes.Buffer
	out.WriteString("\xFF\xD8")
	out.WriteByte(0xFF)
	out.WriteByte(0xE1)
	binary.Write(&out, binary.BigEndian, uint16(2+app1.Len()))
	out.Write(app1.Bytes())
	out.WriteString("\xFF\xD9")

	if got := getEXIFOrientation(out.Bytes()); got != 6 {
		t.Errorf("big-endian: got %d, want 6", got)
	}
}

func TestApplyEXIFOrientation_NoChange(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 4, 4))
	// orientation 1 → unchanged
	if got := ApplyEXIFOrientation(img, buildEXIFJPEG(1)); got != img {
		t.Error("orientation 1 should return img unchanged")
	}
	// non-JPEG → unchanged
	if got := ApplyEXIFOrientation(img, []byte("nope")); got != img {
		t.Error("non-EXIF data should return img unchanged")
	}
}

func TestApplyEXIFOrientation_Rotated(t *testing.T) {
	src := image.NewRGBA(image.Rect(0, 0, 4, 2))
	src.Set(0, 0, color.RGBA{R: 255, A: 255})
	// orientation 3 → Rotate180, should differ from src and have swapped dims
	got := ApplyEXIFOrientation(src, buildEXIFJPEG(3))
	if got == src {
		t.Fatal("orientation 3 should produce a transformed image")
	}
	b := got.Bounds()
	if b.Dx() != 4 || b.Dy() != 2 {
		t.Errorf("Rotate180 dims = %dx%d, want 4x2", b.Dx(), b.Dy())
	}
}

func TestParseOrientationInt(t *testing.T) {
	var v int
	got, err := parseOrientationInt("6", &v)
	if err != nil || got != 6 || v != 6 {
		t.Errorf("parse 6: got=%d v=%d err=%v", got, v, err)
	}
	if _, err := parseOrientationInt("9", nil); err == nil {
		t.Error("expected error for out-of-range orientation")
	}
	if _, err := parseOrientationInt("foo", nil); err == nil {
		t.Error("expected error for non-numeric orientation")
	}
	if _, err := parseOrientationInt(" 3 ", nil); err != nil {
		t.Errorf("trim-space parse should succeed: %v", err)
	}
}
