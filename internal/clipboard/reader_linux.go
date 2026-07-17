//go:build linux

package clipboard

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"mcp_images/internal/logger"
)

type linuxReader struct {
	logger logger.Logger
}

func (r *linuxReader) ReadImage(ctx context.Context) ([]byte, error) {
	if isWSL() {
		return r.readWSL(ctx)
	}

	wayland := os.Getenv("WAYLAND_DISPLAY") != "" || os.Getenv("XDG_SESSION_TYPE") == "wayland"
	if wayland {
		return r.readWayland(ctx)
	}
	return r.readX11(ctx)
}

func isWSL() bool {
	data, err := os.ReadFile("/proc/version")
	if err != nil {
		return false
	}
	lower := strings.ToLower(string(data))
	return strings.Contains(lower, "microsoft") || strings.Contains(lower, "wsl")
}

func (r *linuxReader) readWSL(ctx context.Context) ([]byte, error) {
	ps, err := findPowerShell()
	if err != nil {
		return nil, fmt.Errorf("[剪贴板错误] WSL 环境需要 PowerShell 访问 Windows 剪贴板，但未找到 powershell.exe。")
	}

	cmd := exec.CommandContext(ctx, ps, "-NoProfile", "-Command",
		"Add-Type -AssemblyName System.Drawing; Add-Type -AssemblyName System.Windows.Forms; "+
			"try { $img = [System.Windows.Forms.Clipboard]::GetImage(); } catch { Write-Host 'NO_IMAGE'; exit; } "+
			"if ($img -ne $null) { "+
			"$tempFile = [System.IO.Path]::GetTempFileName() + '.png'; "+
			"$img.Save($tempFile, [System.Drawing.Imaging.ImageFormat]::Png); "+
			"Write-Host $tempFile; "+
			"} else { Write-Host 'NO_IMAGE' }")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		r.logger.Debug("WSL PowerShell 剪贴板读取失败", logger.Field{Key: "error", Value: err.Error()}, logger.Field{Key: "stderr", Value: stderr.String()})
		return nil, fmt.Errorf("[剪贴板错误] 无法读取 Windows 剪贴板（PowerShell 执行失败）。请确认 Windows 剪贴板中有图片。")
	}

	output := strings.TrimSpace(stdout.String())
	if output == "NO_IMAGE" || output == "" {
		return nil, fmt.Errorf("[剪贴板错误] 当前 Windows 剪贴板中没有图片。请先截图或复制图片到剪贴板。")
	}

	wslPath := convertWindowsPathToWSL(output)
	defer os.Remove(wslPath)

	data, err := os.ReadFile(wslPath)
	if err != nil {
		return nil, fmt.Errorf("[剪贴板错误] 读取截屏文件失败：%v", err)
	}
	return data, nil
}

func findPowerShell() (string, error) {
	if _, err := exec.LookPath("powershell.exe"); err == nil {
		return "powershell.exe", nil
	}
	candidates := []string{
		"/mnt/c/Windows/System32/WindowsPowerShell/v1.0/powershell.exe",
		"/mnt/c/Windows/SysWOW64/WindowsPowerShell/v1.0/powershell.exe",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("powershell.exe not found")
}

func convertWindowsPathToWSL(windowsPath string) string {
	windowsPath = strings.TrimSpace(windowsPath)
	if len(windowsPath) < 2 || windowsPath[1] != ':' {
		return windowsPath
	}
	drive := strings.ToLower(string(windowsPath[0]))
	rest := strings.ReplaceAll(windowsPath[2:], "\\", "/")
	return "/mnt/" + drive + rest
}

func (r *linuxReader) readX11(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "xclip", "-selection", "clipboard", "-t", "image/png", "-o")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		r.logger.Debug("xclip 剪贴板读取失败", logger.Field{Key: "error", Value: err.Error()}, logger.Field{Key: "stderr", Value: stderr.String()})
		return nil, fmt.Errorf("[剪贴板错误] 无法读取剪贴板图片（xclip 执行失败）。请确保已安装 xclip：sudo apt install xclip")
	}

	if stdout.Len() == 0 {
		return nil, fmt.Errorf("[剪贴板错误] 当前剪贴板中没有图片。请先截图或复制图片到剪贴板。")
	}

	return stdout.Bytes(), nil
}

func (r *linuxReader) readWayland(ctx context.Context) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "wl-paste", "--type", "image/png")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		r.logger.Debug("wl-paste 剪贴板读取失败", logger.Field{Key: "error", Value: err.Error()}, logger.Field{Key: "stderr", Value: stderr.String()})
		return nil, fmt.Errorf("[剪贴板错误] 无法读取剪贴板图片（wl-paste 执行失败）。请确保已安装 wl-paste：sudo apt install wl-clipboard")
	}

	if stdout.Len() == 0 {
		return nil, fmt.Errorf("[剪贴板错误] 当前剪贴板中没有图片。请先截图或复制图片到剪贴板。")
	}

	return stdout.Bytes(), nil
}
