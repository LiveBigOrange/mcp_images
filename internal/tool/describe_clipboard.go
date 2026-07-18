package tool

import (
	"context"

	"mcp_images/internal/clipboard"
	imgproc "mcp_images/internal/image"

	"mcp_images/internal/vlm"
)

type DescribeClipboardImage struct {
	vlmClient *vlm.Client
	processor *imgproc.Processor
	clipboard clipboard.ClipboardReader
}

func NewDescribeClipboardImage(vlmClient *vlm.Client, processor *imgproc.Processor, clipReader clipboard.ClipboardReader) *DescribeClipboardImage {
	return &DescribeClipboardImage{
		vlmClient: vlmClient,
		processor: processor,
		clipboard: clipReader,
	}
}

func (t *DescribeClipboardImage) Name() string {
	return "describe_clipboard_image"
}

func (t *DescribeClipboardImage) Description() string {
	return `读取系统剪贴板中的图片并分析。当用户提到"截图"、"粘贴图片"、"剪贴板中有图片"、"刚才截的图"时，请调用此工具。注意：MCP 协议无法直接获取对话中嵌入的图片，如果用户在对话中粘贴了图片，该图片通常同时存在于系统剪贴板中，可使用此工具读取。`
}

func (t *DescribeClipboardImage) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
		"required":   []string{},
	}
}

func (t *DescribeClipboardImage) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	data, err := clipboard.ReadWithLock(ctx, t.clipboard)
	if err != nil {
		return "", err
	}

	if err := clipboard.ValidateClipboardData(data); err != nil {
		return "", err
	}

	dataURI, err := t.processor.Process(ctx, data, "png")
	if err != nil {
		return "", err
	}

	result, err := t.vlmClient.AnalyzeImage(ctx, dataURI)
	if err != nil {
		return "", err
	}

	return result, nil
}
