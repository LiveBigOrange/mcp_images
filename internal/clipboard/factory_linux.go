//go:build linux

package clipboard

import "mcp_images/internal/logger"

func newPlatformReader(lg logger.Logger) ClipboardReader {
	return &linuxReader{logger: lg}
}
