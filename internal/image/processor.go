package image

import (
	"context"
	"fmt"
	"runtime"
)

type Processor struct {
	MaxDimension int
}

func NewProcessor() *Processor {
	return &Processor{MaxDimension: maxDimension}
}

func NewProcessorWithMax(maxDim int) *Processor {
	if maxDim <= 0 {
		maxDim = maxDimension
	}
	return &Processor{MaxDimension: maxDim}
}

func (p *Processor) Process(ctx context.Context, data []byte, formatHint string) (dataURI string, err error) {
	defer func() {
		if r := recover(); r != nil {
			buf := make([]byte, 16384)
			n := runtime.Stack(buf, false)
			err = fmt.Errorf("[图片处理失败] 内存不足或处理异常：%v\n%s", r, buf[:n])
		}
	}()

	if err := ctx.Err(); err != nil {
		return "", err
	}

	img, _, err := Decode(data, formatHint)
	if err != nil {
		return "", err
	}

	if err := ctx.Err(); err != nil {
		return "", err
	}

	img = ApplyEXIFOrientation(img, data)

	img = convertToRGB(img)

	if err := ctx.Err(); err != nil {
		return "", err
	}

	img = ResizeIfNeededWithMax(img, len(data), p.MaxDimension)

	if err := ctx.Err(); err != nil {
		return "", err
	}

	dataURI, err = EncodeToJPEGBase64(img)
	if err != nil {
		return "", fmt.Errorf("[图片编码失败] JPEG 编码出错：%v", err)
	}

	return dataURI, nil
}
