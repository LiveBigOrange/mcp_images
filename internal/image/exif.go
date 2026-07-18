package image

import (
	"encoding/binary"
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/disintegration/imaging"
)

func ApplyEXIFOrientation(img image.Image, data []byte) image.Image {
	orientation := getEXIFOrientation(data)
	if orientation == 1 || orientation == 0 {
		return img
	}

	switch orientation {
	case 2:
		return imaging.FlipH(img)
	case 3:
		return imaging.Rotate180(img)
	case 4:
		return imaging.FlipV(img)
	case 5:
		return imaging.FlipV(imaging.Rotate90(img))
	case 6:
		return imaging.Rotate270(img)
	case 7:
		return imaging.FlipH(imaging.Rotate90(img))
	case 8:
		return imaging.Rotate90(img)
	default:
		return img
	}
}

// getEXIFOrientation extracts the EXIF Orientation tag (0x0112) from a JPEG
// image's APP1 segment without pulling in a full EXIF library. Returns 0 if
// the tag is absent or the data is not a JPEG containing EXIF metadata.
func getEXIFOrientation(data []byte) int {
	tiffData, bo, err := extractEXIFTIFF(data)
	if err != nil {
		return 0
	}

	ifd0, err := readIFD(tiffData, bo, 8)
	if err != nil {
		return 0
	}

	for _, entry := range ifd0 {
		if entry.tag == 0x0112 && entry.typ == 3 && entry.count == 1 {
			val := bo.Uint16(entry.value[:2])
			if val >= 1 && val <= 8 {
				return int(val)
			}
			return 0
		}
	}
	return 0
}

// extractEXIFTIFF scans JPEG APP1 segments to find the EXIF TIFF block.
func extractEXIFTIFF(data []byte) (tiff []byte, bo binary.ByteOrder, err error) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, nil, fmt.Errorf("not a JPEG")
	}

	i := 2
	for i+2 <= len(data) {
		if data[i] != 0xFF {
			// Not a marker; bail out rather than scanning into entropy data.
			return nil, nil, fmt.Errorf("expected marker prefix 0xFF at %d", i)
		}
		marker := data[i+1]
		// Standalone markers (no length field): RSTn, SOI, EOI, TEM, and
		// the 0xFF00 byte-stuffing padding marker.
		if marker == 0x00 || marker == 0xD8 || marker == 0xD9 || (marker >= 0xD0 && marker <= 0xD7) || marker == 0x01 {
			i += 2
			continue
		}
		// SOS marks the start of entropy-coded data — no APP segments after it.
		if marker == 0xDA {
			return nil, nil, fmt.Errorf("no EXIF APP1 found")
		}
		if i+4 > len(data) {
			return nil, nil, fmt.Errorf("truncated marker %d at %d", marker, i)
		}
		segLen := int(binary.BigEndian.Uint16(data[i+2 : i+4]))
		segStart := i + 4
		segEnd := i + 2 + segLen
		if segLen < 2 || segEnd > len(data) {
			return nil, nil, fmt.Errorf("segment %d overflows data", marker)
		}

		if marker == 0xE1 && segEnd-segStart >= 6 && string(data[segStart:segStart+6]) == "Exif\x00\x00" {
			tiffStart := segStart + 6
			tiffData := data[tiffStart:segEnd]
			if len(tiffData) < 8 {
				return nil, nil, fmt.Errorf("EXIF TIFF header too short")
			}
			switch string(tiffData[0:2]) {
			case "II":
				bo = binary.LittleEndian
			case "MM":
				bo = binary.BigEndian
			default:
				return nil, nil, fmt.Errorf("bad TIFF byte order")
			}
			return tiffData, bo, nil
		}

		i = segEnd
	}
	return nil, nil, fmt.Errorf("no EXIF APP1 found")
}

// ifdEntry describes a single TIFF IFD entry relevant to our search.
type ifdEntry struct {
	tag   uint16
	typ   uint16
	count uint32
	value [4]byte
}

// readIFD parses an IFD at the given offset within the TIFF block.
func readIFD(tiff []byte, bo binary.ByteOrder, offset int) ([]ifdEntry, error) {
	if offset+2 > len(tiff) {
		return nil, fmt.Errorf("IFD count out of range")
	}
	count := int(bo.Uint16(tiff[offset : offset+2]))
	entriesStart := offset + 2
	if entriesStart+count*12 > len(tiff) {
		return nil, fmt.Errorf("IFD entries overflow tiff data")
	}
	entries := make([]ifdEntry, 0, count)
	for i := 0; i < count; i++ {
		base := entriesStart + i*12
		e := ifdEntry{
			tag:   bo.Uint16(tiff[base : base+2]),
			typ:   bo.Uint16(tiff[base+2 : base+4]),
			count: bo.Uint32(tiff[base+4 : base+8]),
		}
		copy(e.value[:], tiff[base+8:base+12])
		entries = append(entries, e)
	}
	return entries, nil
}

// parseOrientationInt parses a numeric orientation string in the range [1,8].
// Kept for backwards compatibility / external callers; returns the value as
// both the result and via the optional out-pointer.
func parseOrientationInt(s string, v *int) (int, error) {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0, err
	}
	if n < 1 || n > 8 {
		return 0, fmt.Errorf("invalid orientation value: %d", n)
	}
	if v != nil {
		*v = n
	}
	return n, nil
}
