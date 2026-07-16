//go:build darwin

package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"

	"mcp_images/internal/logger"
)

type darwinReader struct {
	logger logger.Logger
}

func (r *darwinReader) ReadImage(ctx context.Context) ([]byte, error) {
	script := `the clipboard as «class PNGf»`

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		r.logger.Debug("osascript 剪贴板读取失败", logger.Field{Key: "error", Value: err.Error()}, logger.Field{Key: "stderr", Value: stderr.String()})
		return nil, fmt.Errorf("[剪贴板错误] 无法读取剪贴板图片（osascript 执行失败）。请检查辅助功能权限设置。")
	}

	if stdout.Len() == 0 {
		return nil, fmt.Errorf("[剪贴板错误] 当前剪贴板中没有图片。请先截图或复制图片到剪贴板。")
	}

	return stdout.Bytes(), nil
}
