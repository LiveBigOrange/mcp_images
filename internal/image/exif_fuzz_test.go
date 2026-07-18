package image

import (
	"bytes"
	"encoding/binary"
	"testing"
	"testing/quick"
)

// Fuzz-ish: ensure getEXIFOrientation never panics on arbitrary byte streams.
func TestGetEXIFOrientation_QuickNoPanic(t *testing.T) {
	err := quick.Check(func(seed uint64) bool {
		// Build deterministic pseudo-random slice from seed
		n := int(seed%256) + 1
		data := make([]byte, n)
		for i := 0; i < n; i++ {
			seed = seed*1103515245 + 12345
			data[i] = byte(seed >> 16)
		}
		_ = getEXIFOrientation(data)
		return true
	}, &quick.Config{MaxCount: 5000})
	if err != nil {
		t.Fatal(err)
	}
}

// Sanity: a JPEG with SOI + APP1 "Exif\x00\x00" but only 4 bytes of TIFF (too short
// for byte order). Should return 0, not panic, not misread.
func TestExtractEXIFTIFF_ShortTIFF(t *testing.T) {
	var out []byte
	out = append(out, 0xFF, 0xD8)
	out = append(out, 0xFF, 0xE1)
	out = append(out, 0x00, 0x0A) // len 10
	out = append(out, []byte("Exif\x00\x00")...)
	out = append(out, 'I', 'I', 0x00, 0x00) // <8 bytes of TIFF
	out = append(out, 0xFF, 0xD9)
	if v := getEXIFOrientation(out); v != 0 {
		t.Errorf("short TIFF: got %d, want 0", v)
	}
}

// Test malformed segment length
func TestExtractEXIFTIFF_BadSegLen(t *testing.T) {
	// segLen=1 (smaller than 2) should error, not panic
	out := []byte{0xFF, 0xD8, 0xFF, 0xE1, 0x00, 0x01, 0xFF, 0xD9}
	if v := getEXIFOrientation(out); v != 0 {
		t.Errorf("bad seglen: got %d, want 0", v)
	}
}

// Test APP1 marker but not EXIF signature — should keep scanning.
func TestExtractEXIFTIFF_NonExifAPP1(t *testing.T) {
	var out []byte
	out = append(out, 0xFF, 0xD8)
	out = append(out, 0xFF, 0xE1)                   // APP1
	out = append(out, 0x00, 0x06)                   // len 6
	out = append(out, []byte("XMP\x00\x00\x00")...) // non-Exif signature
	out = append(out, 0xFF, 0xD9)
	if v := getEXIFOrientation(out); v != 0 {
		t.Errorf("non-exif APP1: got %d, want 0", v)
	}
}

// Test 0xFF00 padding marker behaves like a standalone marker
func TestExtractEXIFTIFF_PaddingMarker(t *testing.T) {
	var out []byte
	out = append(out, 0xFF, 0xD8)
	out = append(out, 0xFF, 0x00) // byte stuffing / padding marker
	out = append(out, 0xFF, 0xD9)
	if v := getEXIFOrientation(out); v != 0 {
		t.Errorf("padding only: got %d, want 0", v)
	}
}

// Build a longer TIFF with multiple IFD entries — make sure scanner finds Orientation in pos N
func TestExtractEXIFTIFF_MultipleEntries(t *testing.T) {
	var tiff bytes.Buffer
	tiff.WriteString("II")
	binary.Write(&tiff, binary.LittleEndian, uint16(0x002A))
	binary.Write(&tiff, binary.LittleEndian, uint32(8))
	binary.Write(&tiff, binary.LittleEndian, uint16(2))
	// 1st entry: Make (0x010F), ASCII, count 1 (junk)
	binary.Write(&tiff, binary.LittleEndian, uint16(0x010F))
	binary.Write(&tiff, binary.LittleEndian, uint16(2))
	binary.Write(&tiff, binary.LittleEndian, uint32(1))
	tiff.Write([]byte{0x41, 0, 0, 0})
	// 2nd entry: Orientation (0x0112), SHORT, count 1, value 6
	binary.Write(&tiff, binary.LittleEndian, uint16(0x0112))
	binary.Write(&tiff, binary.LittleEndian, uint16(3))
	binary.Write(&tiff, binary.LittleEndian, uint32(1))
	var v [4]byte
	binary.LittleEndian.PutUint16(v[:2], 6)
	tiff.Write(v[:])
	binary.Write(&tiff, binary.LittleEndian, uint32(0))

	var out bytes.Buffer
	out.WriteString("\xFF\xD8")
	out.WriteByte(0xFF)
	out.WriteByte(0xE1)
	binary.Write(&out, binary.BigEndian, uint16(2+6+tiff.Len()))
	out.WriteString("Exif\x00\x00")
	out.Write(tiff.Bytes())
	out.WriteString("\xFF\xD9")

	if got := getEXIFOrientation(out.Bytes()); got != 6 {
		t.Errorf("multi-entry: got %d, want 6", got)
	}
}
