package tool

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	imgproc "mcp_images/internal/image"

	"mcp_images/internal/vlm"
)

const maxFileSize = 20 * 1024 * 1024
const maxPathLength = 4096

type DescribeImageFile struct {
	vlmClient *vlm.Client
	processor *imgproc.Processor
}

func NewDescribeImageFile(vlmClient *vlm.Client, processor *imgproc.Processor) *DescribeImageFile {
	return &DescribeImageFile{
		vlmClient: vlmClient,
		processor: processor,
	}
}

func (t *DescribeImageFile) Name() string {
	return "describe_image_file"
}

func (t *DescribeImageFile) Description() string {
	return `分析本地图片文件。当用户提供本地图片文件路径、截图保存路径、或提到"查看这张图片"、"分析这个截图"时，请调用此工具。支持 JPEG、PNG、BMP、TIFF、WebP 格式。`
}

func (t *DescribeImageFile) InputSchema() map[string]interface{} {
	return map[string]interface{}{
		"type": "object",
		"properties": map[string]interface{}{
			"image_path": map[string]interface{}{
				"type":        "string",
				"description": "本地图片的绝对路径",
			},
		},
		"required": []string{"image_path"},
	}
}

func (t *DescribeImageFile) Execute(ctx context.Context, args map[string]interface{}) (string, error) {
	imagePath, ok := args["image_path"].(string)
	if !ok || imagePath == "" {
		return "", fmt.Errorf("[参数错误] image_path 参数缺失或类型错误，必须为非空字符串。")
	}

	if err := validatePath(imagePath); err != nil {
		return "", err
	}

	resolved, err := filepath.EvalSymlinks(imagePath)
	if err != nil {
		return "", fmt.Errorf("[文件错误] 无法解析路径：%v。请检查路径是否正确。", err)
	}

	if err := validatePath(resolved); err != nil {
		return "", err
	}

	info, err := os.Stat(resolved)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("[文件错误] 图片文件不存在：%s", imagePath)
		}
		return "", fmt.Errorf("[文件错误] 无法访问文件：%v", err)
	}

	if info.IsDir() {
		return "", fmt.Errorf("[文件错误] 指定路径是目录，不是图片文件：%s", imagePath)
	}

	if info.Size() > maxFileSize {
		return "", fmt.Errorf("[文件错误] 图片文件超过 20 MB 上限，无法处理。当前大小：%.1f MB", float64(info.Size())/(1024*1024))
	}

	data, err := os.ReadFile(resolved)
	if err != nil {
		if os.IsPermission(err) {
			return "", fmt.Errorf("[文件错误] 无权限读取文件：%s", imagePath)
		}
		return "", fmt.Errorf("[文件错误] 读取文件失败：%v", err)
	}

	dataURI, err := t.processor.Process(ctx, data, "")
	if err != nil {
		return "", err
	}

	result, err := t.vlmClient.AnalyzeImage(ctx, dataURI)
	if err != nil {
		return "", err
	}

	return result, nil
}

func validatePath(path string) error {
	if !filepath.IsAbs(path) {
		return fmt.Errorf("[路径错误] image_path 必须为绝对路径，当前为相对路径：%s", path)
	}

	normalized := filepath.Clean(path)
	for _, part := range strings.Split(normalized, string(filepath.Separator)) {
		if part == ".." {
			return fmt.Errorf("[路径错误] 图片路径不合法，禁止包含路径遍历字符（..）。")
		}
	}

	cleaned := strings.ReplaceAll(path, "/", string(filepath.Separator))
	for _, part := range strings.Split(cleaned, string(filepath.Separator)) {
		if part == ".." {
			return fmt.Errorf("[路径错误] 图片路径不合法，禁止包含路径遍历字符（..）。")
		}
	}

	if len(path) > maxPathLength {
		return fmt.Errorf("[路径错误] 图片路径过长（超过 %d 字符）。", maxPathLength)
	}
	return nil
}
