package image

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"

	"golang.org/x/image/bmp"
	"golang.org/x/image/tiff"
	"golang.org/x/image/webp"
)

func Decode(data []byte, formatHint string) (img image.Image, format string, err error) {
	if formatHint != "" {
		img, err = decodeByFormat(data, formatHint)
		if err == nil {
			return img, formatHint, nil
		}
	}

	img, format, err = image.Decode(bytes.NewReader(data))
	if err == nil {
		return img, format, nil
	}

	if formatHint == "" {
		if detected := DetectFormat(data); detected != "" {
			img, err2 := decodeByFormat(data, detected)
			if err2 == nil {
				return img, detected, nil
			}
		}
	}

	if formatHint != "" {
		return nil, "", fmt.Errorf("[图片解码失败] 无法解码图片（提示格式：%s，自动检测也失败）：%v", formatHint, err)
	}
	return nil, "", fmt.Errorf("[图片解码失败] 无法识别图片格式或图片已损坏：%v", err)
}

func decodeByFormat(data []byte, format string) (image.Image, error) {
	r := bytes.NewReader(data)
	switch format {
	case "jpeg", "jpg":
		return jpeg.Decode(r)
	case "png":
		return png.Decode(r)
	case "bmp":
		return bmp.Decode(r)
	case "tiff":
		return tiff.Decode(r)
	case "webp":
		return webp.Decode(r)
	default:
		return nil, fmt.Errorf("不支持的图片格式：%s", format)
	}
}

func DetectFormat(data []byte) string {
	if len(data) < 4 {
		return ""
	}
	switch {
	case bytes.HasPrefix(data, []byte{0xFF, 0xD8, 0xFF}):
		return "jpeg"
	case bytes.HasPrefix(data, []byte{0x89, 0x50, 0x4E, 0x47}):
		return "png"
	case bytes.HasPrefix(data, []byte{0x42, 0x4D}):
		return "bmp"
	case bytes.HasPrefix(data, []byte{0x49, 0x49, 0x2A, 0x00}) || bytes.HasPrefix(data, []byte{0x4D, 0x4D, 0x00, 0x2A}):
		return "tiff"
	case len(data) >= 12 && bytes.HasPrefix(data, []byte{0x52, 0x49, 0x46, 0x46}) && bytes.Equal(data[8:12], []byte{0x57, 0x45, 0x42, 0x50}):
		return "webp"
	}
	return ""
}
