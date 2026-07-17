package image

import (
	"fmt"
	"image"
	"strconv"

	"github.com/disintegration/imaging"
	exif "github.com/dsoprea/go-exif/v3"
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

func getEXIFOrientation(data []byte) int {
	rawExif, err := exif.SearchAndExtractExif(data)
	if err != nil {
		return 0
	}

	tags, _, err := exif.GetFlatExifData(rawExif, nil)
	if err != nil {
		return 0
	}

	for _, tag := range tags {
		if tag.TagName == "Orientation" {
			if val, ok := tag.Value.([]uint16); ok && len(val) > 0 {
				return int(val[0])
			}
			if tag.FormattedFirst != "" {
				var orient int
				if _, err := parseOrientationInt(tag.FormattedFirst, &orient); err == nil {
					return orient
				}
			}
		}
	}
	return 0
}

func parseOrientationInt(s string, v *int) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, err
	}
	if n < 1 || n > 8 {
		return 0, fmt.Errorf("invalid orientation value: %d", n)
	}
	*v = n
	return n, nil
}
