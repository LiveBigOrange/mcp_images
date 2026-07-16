//go:build windows

package clipboard

import "mcp_images/internal/logger"

func newPlatformReader(lg logger.Logger) ClipboardReader {
	return &windowsReader{logger: lg}
}
