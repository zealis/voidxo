package imageproc

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"

	xdraw "golang.org/x/image/draw"
)

const (
	// MaxDimension 限制单边像素，避免超大图占用内存。
	MaxDimension = 8000
	// MaxPixels 限制总像素，抵御解压炸弹。
	MaxPixels = 40_000_000
)

type Format string

const (
	JPEG Format = "jpeg"
	PNG  Format = "png"
	GIF  Format = "gif"
)

var ErrUnsupported = errors.New("unsupported image format")

// Decode 先读取配置做尺寸/像素校验，再解码，避免直接解码超大图。
func Decode(data []byte) (image.Image, Format, error) {
	config, name, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, "", ErrUnsupported
	}
	if config.Width <= 0 || config.Height <= 0 ||
		config.Width > MaxDimension || config.Height > MaxDimension ||
		config.Width*config.Height > MaxPixels {
		return nil, "", fmt.Errorf("image dimensions %dx%d are not allowed", config.Width, config.Height)
	}
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, "", ErrUnsupported
	}
	switch name {
	case "jpeg":
		return img, JPEG, nil
	case "png":
		return img, PNG, nil
	case "gif":
		return img, GIF, nil
	}
	return nil, "", ErrUnsupported
}

// Resize 按宽度等比缩放；width 为 0 时返回原图副本。
func Resize(img image.Image, width int) *image.NRGBA {
	bounds := img.Bounds()
	if width <= 0 || width >= bounds.Dx() {
		return copyToNRGBA(img)
	}
	height := bounds.Dy() * width / bounds.Dx()
	if height < 1 {
		height = 1
	}
	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, bounds, xdraw.Over, nil)
	return dst
}

// Crop 居中裁剪到指定宽高比后再缩放，用于头像与封面。
func Crop(img image.Image, width, height int) *image.NRGBA {
	bounds := img.Bounds()
	sourceRatio := float64(bounds.Dx()) / float64(bounds.Dy())
	targetRatio := float64(width) / float64(height)

	cropWidth, cropHeight := bounds.Dx(), bounds.Dy()
	if sourceRatio > targetRatio {
		cropWidth = int(float64(bounds.Dy()) * targetRatio)
	} else {
		cropHeight = int(float64(bounds.Dx()) / targetRatio)
	}
	offsetX := bounds.Min.X + (bounds.Dx()-cropWidth)/2
	offsetY := bounds.Min.Y + (bounds.Dy()-cropHeight)/2
	rect := image.Rect(offsetX, offsetY, offsetX+cropWidth, offsetY+cropHeight)

	dst := image.NewNRGBA(image.Rect(0, 0, width, height))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), img, rect, xdraw.Over, nil)
	return dst
}

// Encode 统一重新编码，顺带剥离 EXIF 等元数据。
func Encode(img image.Image, format Format) ([]byte, error) {
	var buffer bytes.Buffer
	switch format {
	case PNG, GIF:
		if err := png.Encode(&buffer, img); err != nil {
			return nil, fmt.Errorf("encode png: %w", err)
		}
	default:
		if err := jpeg.Encode(&buffer, img, &jpeg.Options{Quality: 85}); err != nil {
			return nil, fmt.Errorf("encode jpeg: %w", err)
		}
	}
	return buffer.Bytes(), nil
}

// FirstFrame 把动图降级为首帧，用于缩略图。
func FirstFrame(data []byte) (image.Image, error) {
	decoded, err := gif.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode gif frame: %w", err)
	}
	return decoded, nil
}

func copyToNRGBA(img image.Image) *image.NRGBA {
	bounds := img.Bounds()
	dst := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	xdraw.Draw(dst, dst.Bounds(), img, bounds.Min, xdraw.Src)
	return dst
}

// ReadAllLimited 读取上传内容并限制最大字节数。
func ReadAllLimited(reader io.Reader, limit int64) ([]byte, error) {
	limited := io.LimitReader(reader, limit+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("file exceeds %d bytes", limit)
	}
	return data, nil
}
