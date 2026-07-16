package image

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/jpeg"
)

func EncodeToJPEGBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	buf.WriteString("data:image/jpeg;base64,")

	b64Writer := base64.NewEncoder(base64.StdEncoding, &buf)
	if err := jpeg.Encode(b64Writer, img, &jpeg.Options{Quality: 85}); err != nil {
		return "", err
	}
	if err := b64Writer.Close(); err != nil {
		return "", err
	}

	return buf.String(), nil
}
