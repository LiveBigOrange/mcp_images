package tool

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	imgproc "mcp_images/internal/image"

	"mcp_images/internal/vlm"
)

const maxBase64Len = 28 * 1024 * 1024
const maxDecodedSize = 20 * 1024 * 1024

var validFormats = map[string]string{
	"jpeg": "jpeg", "jpg": "jpeg",
	"png":  "png",
	"bmp":  "bmp",
	"tiff": "tiff",
	"webp": "webp",
}

type DescribeBase64Image struct {
	vlmClient *vlm.Client
	processor *imgproc.Processor
}

func NewDescribeBase64Image(vlmClient *vlm.Client, processor *imgproc.Processor) *DescribeBase64Image {
	return &DescribeBase64Image{
		vlmClient: vlmClient,
		processor: processor,
	}
}

func (t *DescribeBase64Image) Name() string {
	return "describe_base64_image"
}

func (t *DescribeBase64Image) Description() string {
	return `分析 Base64 编码的图片数据。当 IDE 将对话中的图片以 Base64 形式传递、或需要分析 Base64 编码的图片数据时，请调用此工具。支持 JPEG、PNG、BMP、TIFF、WebP 格式。`
}

func (t *DescribeBase64Image) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"image_base64": map[string]interface{}{
				"type":        "string",
				"description": "Base64 编码的图片数据（纯 Base64 字符串，禁止包含 data:image/ 前缀）",
			},
			"image_format": map[string]interface{}{
				"type":        "string",
				"description": "图片格式提示（jpeg/png/bmp/tiff/webp），默认 jpeg",
				"enum":        []string{"jpeg", "png", "bmp", "tiff", "webp"},
			},
		},
		"required": []string{"image_base64"},
	}
}

func (t *DescribeBase64Image) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	b64Str, ok := args["image_base64"].(string)
	if !ok || b64Str == "" {
		return "", fmt.Errorf("[参数错误] image_base64 参数缺失或类型错误，必须为非空字符串。")
	}

	if strings.HasPrefix(b64Str, "data:image/") {
		return "", fmt.Errorf("[参数错误] image_base64 禁止包含 Data URI 前缀（data:image/...），请仅传入纯 Base64 字符串。")
	}

	if len(b64Str) > maxBase64Len {
		return "", fmt.Errorf("[参数错误] Base64 字符串过长（超过 28 MB），无法处理。建议改用 describe_image_file 传入文件路径，或 describe_clipboard_image 读取剪贴板。")
	}

	data, err := base64.StdEncoding.DecodeString(b64Str)
	if err != nil {
		return "", fmt.Errorf("[参数错误] Base64 解码失败：%v。请确认输入为合法的 Base64 字符串。", err)
	}

	if len(data) > maxDecodedSize {
		return "", fmt.Errorf("[参数错误] 解码后图片数据超过 20 MB 上限，无法处理。")
	}

	formatHint := "jpeg"
	if fmtVal, ok := args["image_format"].(string); ok && fmtVal != "" {
		normalized, valid := validFormats[strings.ToLower(fmtVal)]
		if !valid {
			return "", fmt.Errorf("[参数错误] image_format 值无效（%s），支持：jpeg、png、bmp、tiff、webp。", fmtVal)
		}
		formatHint = normalized
	}

	dataURI, err := t.processor.Process(ctx, data, formatHint)
	if err != nil {
		return "", err
	}

	result, err := t.vlmClient.AnalyzeImage(ctx, dataURI)
	if err != nil {
		return "", err
	}

	return result, nil
}
