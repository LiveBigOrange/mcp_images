package clipboard

import (
	"context"
	"fmt"

	"sync"

	"mcp_images/internal/logger"
)

type ClipboardReader interface {
	ReadImage(ctx context.Context) ([]byte, error)
}

var clipboardMu sync.Mutex

func NewReader(lg logger.Logger) ClipboardReader {
	return newPlatformReader(lg)
}

func ReadWithLock(ctx context.Context, reader ClipboardReader) ([]byte, error) {
	clipboardMu.Lock()
	defer clipboardMu.Unlock()
	return reader.ReadImage(ctx)
}

func ValidateClipboardData(data []byte) error {
	if len(data) == 0 {
		return fmt.Errorf("[剪贴板错误] 当前剪贴板中没有图片。请先截图或复制图片到剪贴板。")
	}

	if len(data) < 8 {
		return fmt.Errorf("[剪贴板错误] 剪贴板中的图片数据异常，无法解码。请重新截图后重试。")
	}

	isPNG := data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47
	isJPEG := data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF

	if !isPNG && !isJPEG {
		return fmt.Errorf("[剪贴板错误] 剪贴板中的图片数据异常，无法解码。请重新截图后重试。")
	}

	return nil
}
