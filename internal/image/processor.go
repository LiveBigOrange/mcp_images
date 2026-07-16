package image

import (
	"fmt"
)

type Processor struct{}

func NewProcessor() *Processor {
	return &Processor{}
}

func (p *Processor) Process(data []byte, formatHint string) (dataURI string, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("[图片处理失败] 内存不足或处理异常：%v", r)
		}
	}()

	img, _, err := Decode(data, formatHint)
	if err != nil {
		return "", err
	}

	img = ApplyEXIFOrientation(img, data)

	img = convertToRGB(img)

	img = ResizeIfNeeded(img, len(data))

	dataURI, err = EncodeToJPEGBase64(img)
	if err != nil {
		return "", fmt.Errorf("[图片编码失败] JPEG 编码出错：%v", err)
	}

	return dataURI, nil
}
