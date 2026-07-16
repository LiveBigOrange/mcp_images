//go:build darwin

package clipboard

import "mcp_images/internal/logger"

func newPlatformReader(lg logger.Logger) ClipboardReader {
	return &darwinReader{logger: lg}
}
