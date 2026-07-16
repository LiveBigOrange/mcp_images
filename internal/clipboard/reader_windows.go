//go:build windows

package clipboard

import (
	"context"
	"encoding/binary"
	"fmt"
	"image"

	"image/png"
	"io"
	"syscall"
	"unsafe"

	"mcp_images/internal/logger"
)

var (
	user32                         = syscall.NewLazyDLL("user32.dll")
	kernel32                       = syscall.NewLazyDLL("kernel32.dll")
	procOpenClipboard              = user32.NewProc("OpenClipboard")
	procCloseClipboard             = user32.NewProc("CloseClipboard")
	procIsClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	procGetClipboardData           = user32.NewProc("GetClipboardData")
	procGlobalLock                 = kernel32.NewProc("GlobalLock")
	procGlobalUnlock               = kernel32.NewProc("GlobalUnlock")
	procGlobalSize                 = kernel32.NewProc("GlobalSize")
	procCreateStreamOnHGlobal      = syscall.NewLazyDLL("ole32.dll").NewProc("CreateStreamOnHGlobal")
)

const (
	cfPNG   = 0x000F
	cfDIB   = 8
	cfDIBv5 = 17
)

type windowsReader struct {
	logger logger.Logger
}

func (r *windowsReader) ReadImage(ctx context.Context) ([]byte, error) {
	data, err := r.readPNG()
	if err == nil && len(data) > 0 {
		return data, nil
	}

	r.logger.Debug("PNG 格式不可用，尝试 DIB 格式", logger.Field{Key: "png_error", Value: err.Error()})

	data, err = r.readDIB()
	if err != nil {
		return nil, fmt.Errorf("[剪贴板错误] 无法读取剪贴板图片。请先截图或复制图片到剪贴板。")
	}
	return data, nil
}

func (r *windowsReader) readPNG() ([]byte, error) {
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return nil, fmt.Errorf("OpenClipboard 失败")
	}
	defer procCloseClipboard.Call()

	ret, _, _ = procIsClipboardFormatAvailable.Call(cfPNG)
	if ret == 0 {
		return nil, fmt.Errorf("剪贴板中无 PNG 格式")
	}

	hData, _, _ := procGetClipboardData.Call(cfPNG)
	if hData == 0 {
		return nil, fmt.Errorf("GetClipboardData 失败")
	}

	size, _, _ := procGlobalSize.Call(hData)
	if size == 0 {
		return nil, fmt.Errorf("GlobalSize 返回 0")
	}

	ptr, _, _ := procGlobalLock.Call(hData)
	if ptr == 0 {
		return nil, fmt.Errorf("GlobalLock 失败")
	}
	defer procGlobalUnlock.Call(hData)

	data := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)
	copy(data, src)

	return data, nil
}

func (r *windowsReader) readDIB() ([]byte, error) {
	ret, _, _ := procOpenClipboard.Call(0)
	if ret == 0 {
		return nil, fmt.Errorf("OpenClipboard 失败")
	}
	defer procCloseClipboard.Call()

	format := cfDIBv5
	ret, _, _ = procIsClipboardFormatAvailable.Call(uintptr(format))
	if ret == 0 {
		format = cfDIB
		ret, _, _ = procIsClipboardFormatAvailable.Call(uintptr(format))
		if ret == 0 {
			return nil, fmt.Errorf("剪贴板中无 DIB 格式")
		}
	}

	hData, _, _ := procGetClipboardData.Call(uintptr(format))
	if hData == 0 {
		return nil, fmt.Errorf("GetClipboardData 失败")
	}

	size, _, _ := procGlobalSize.Call(hData)
	if size == 0 {
		return nil, fmt.Errorf("GlobalSize 返回 0")
	}

	ptr, _, _ := procGlobalLock.Call(hData)
	if ptr == 0 {
		return nil, fmt.Errorf("GlobalLock 失败")
	}
	defer procGlobalUnlock.Call(hData)

	dibData := make([]byte, size)
	src := unsafe.Slice((*byte)(unsafe.Pointer(ptr)), size)
	copy(dibData, src)

	return r.dibToPNG(dibData)
}

type bitmapInfoHeader struct {
	Size          uint32
	Width         int32
	Height        int32
	Planes        uint16
	BitCount      uint16
	Compression   uint32
	SizeImage     uint32
	XPelsPerMeter int32
	YPelsPerMeter int32
	ClrUsed       uint32
	ClrImportant  uint32
}

func (r *windowsReader) dibToPNG(dibData []byte) ([]byte, error) {
	if len(dibData) < 40 {
		return nil, fmt.Errorf("DIB 数据过短")
	}

	var hdr bitmapInfoHeader
	hdr.Size = binary.LittleEndian.Uint32(dibData[0:4])
	hdr.Width = int32(binary.LittleEndian.Uint32(dibData[4:8]))
	hdr.Height = int32(binary.LittleEndian.Uint32(dibData[8:12]))
	hdr.Planes = binary.LittleEndian.Uint16(dibData[12:14])
	hdr.BitCount = binary.LittleEndian.Uint16(dibData[14:16])
	hdr.Compression = binary.LittleEndian.Uint32(dibData[16:20])

	if hdr.BitCount != 32 && hdr.BitCount != 24 {
		return nil, fmt.Errorf("不支持的 DIB 位深度：%d", hdr.BitCount)
	}

	absHeight := hdr.Height
	if absHeight < 0 {
		absHeight = -absHeight
	}

	pixelOffset := hdr.Size
	if hdr.BitCount <= 8 {
		paletteSize := uint32(1) << hdr.BitCount
		if hdr.ClrUsed > 0 && hdr.ClrUsed < paletteSize {
			paletteSize = hdr.ClrUsed
		}
		pixelOffset += paletteSize * 4
	}

	img := image.NewRGBA(image.Rect(0, 0, int(hdr.Width), int(absHeight)))
	pix := img.Pix

	stride := int(hdr.Width) * int(hdr.BitCount/8)
	if stride%4 != 0 {
		stride += 4 - stride%4
	}

	for y := 0; y < int(absHeight); y++ {
		srcRow := int(pixelOffset) + y*stride
		if srcRow+stride > len(dibData) {
			break
		}

		var dstY int
		if hdr.Height > 0 {
			dstY = int(absHeight) - 1 - y
		} else {
			dstY = y
		}

		dstOffset := dstY * img.Stride

		for x := 0; x < int(hdr.Width); x++ {
			srcIdx := srcRow + x*int(hdr.BitCount/8)
			if srcIdx+int(hdr.BitCount/8) > len(dibData) {
				break
			}
			dstIdx := dstOffset + x*4

			if hdr.BitCount == 32 {
				if dstIdx+3 < len(pix) {
					pix[dstIdx] = dibData[srcIdx+2]
					pix[dstIdx+1] = dibData[srcIdx+1]
					pix[dstIdx+2] = dibData[srcIdx]
					pix[dstIdx+3] = dibData[srcIdx+3]
				}
			} else {
				if dstIdx+3 < len(pix) {
					pix[dstIdx] = dibData[srcIdx+2]
					pix[dstIdx+1] = dibData[srcIdx+1]
					pix[dstIdx+2] = dibData[srcIdx]
					pix[dstIdx+3] = 255
				}
			}
		}
	}

	var pngData []byte
	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		png.Encode(pw, img)
	}()
	go func() {
		defer pr.Close()
		chunk := make([]byte, 4096)
		for {
			n, err := pr.Read(chunk)
			if n > 0 {
				pngData = append(pngData, chunk[:n]...)
			}
			if err != nil {
				break
			}
		}
	}()

	return pngData, nil
}
