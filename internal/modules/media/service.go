package media

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/voidxo/voidxo/internal/platform/imageproc"
)

const (
	contentMaxBytes = 5 << 20
	brandMaxBytes   = 2 << 20
)

type variant struct {
	Name   string
	Width  int
	Height int // 0 表示等比缩放
}

// 规格沿用旧系统：内容图片原图/702px/132x88，头像与封面各三档。
func variantsFor(kind string) []variant {
	switch kind {
	case KindAvatar:
		return []variant{{"small", 64, 64}, {"middle", 128, 128}, {"large", 256, 256}}
	case KindCover:
		return []variant{{"small", 600, 336}, {"middle", 1050, 588}, {"large", 1450, 812}}
	case KindTopicCover:
		return []variant{{"small", 360, 202}, {"middle", 720, 404}, {"large", 1080, 606}}
	default:
		return []variant{{"original", 0, 0}, {"release", 702, 0}, {"thumb", 132, 88}}
	}
}

func isBrandKind(kind string) bool {
	return kind == KindAvatar || kind == KindCover || kind == KindTopicCover
}

type Service struct {
	repo    Repository
	storage Storage
	baseURL string
}

func NewService(repo Repository, storage Storage, baseURL string) *Service {
	return &Service{repo: repo, storage: storage, baseURL: strings.TrimRight(baseURL, "/")}
}

func (s *Service) Upload(ctx context.Context, userID int64, kind, filename string, data []byte) (UploadResult, error) {
	if userID <= 0 {
		return UploadResult{}, ErrForbidden
	}
	if _, ok := map[string]bool{KindContent: true, KindAvatar: true, KindCover: true, KindTopicCover: true}[kind]; !ok {
		return UploadResult{}, &ValidationError{Message: "图片用途不正确"}
	}
	limit := contentMaxBytes
	if isBrandKind(kind) {
		limit = brandMaxBytes
	}
	if len(data) == 0 {
		return UploadResult{}, &ValidationError{Message: "请选择要上传的图片"}
	}
	if len(data) > limit {
		return UploadResult{}, ErrTooLarge
	}

	img, format, err := imageproc.Decode(data)
	if err != nil {
		return UploadResult{}, ErrInvalidImage
	}
	if isBrandKind(kind) && format == imageproc.GIF {
		return UploadResult{}, &ValidationError{Message: "头像和封面仅支持 jpg 与 png"}
	}

	key, err := randomKey()
	if err != nil {
		return UploadResult{}, fmt.Errorf("generate image key: %w", err)
	}
	urls := make(map[string]string, 3)

	for _, v := range variantsFor(kind) {
		var payload []byte
		var variantExt string
		switch {
		case v.Name == "original" && format == imageproc.GIF:
			// 动图保留原始字节，缩略图使用首帧静态图，避免丢失动画。
			payload, variantExt = data, "gif"
		case v.Name == "original":
			payload, err = imageproc.Encode(img, format)
			variantExt = extensionFor(format)
		default:
			encoded := img
			if v.Height > 0 {
				encoded = imageproc.Crop(img, v.Width, v.Height)
			} else {
				encoded = imageproc.Resize(img, v.Width)
			}
			outFormat := format
			if outFormat == imageproc.GIF {
				outFormat = imageproc.PNG
			}
			payload, err = imageproc.Encode(encoded, outFormat)
			variantExt = extensionFor(outFormat)
		}
		if err != nil {
			return UploadResult{}, err
		}
		if err := s.storage.Put(ctx, key+"/"+v.Name+"."+variantExt, payload); err != nil {
			return UploadResult{}, err
		}
		urls[v.Name] = s.baseURL + "/" + key + "/" + v.Name
	}

	bounds := img.Bounds()
	image := Image{
		Key: key, Filename: truncate(filename, 255), MIME: mimeFor(format),
		Size: len(data), Width: bounds.Dx(), Height: bounds.Dy(), Kind: kind, UserID: userID,
	}
	if err := s.repo.Create(ctx, image); err != nil {
		return UploadResult{}, err
	}
	return UploadResult{Key: key, URLs: urls, Width: image.Width, Height: image.Height}, nil
}

// Open 返回图片内容与 Content-Type，variant 必须属于该图片类型的规格集合。
func (s *Service) Open(ctx context.Context, key, variantName string) ([]byte, string, error) {
	image, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		return nil, "", err
	}
	valid := false
	for _, v := range variantsFor(image.Kind) {
		if v.Name == variantName {
			valid = true
			break
		}
	}
	if !valid {
		return nil, "", ErrNotFound
	}
	ext := extensionFor(formatFromMIME(image.MIME))
	contentType := image.MIME
	if variantName != "original" && image.MIME == "image/gif" {
		ext, contentType = "png", "image/png"
	}
	data, err := s.storage.Get(ctx, key+"/"+variantName+"."+ext)
	if err != nil {
		return nil, "", err
	}
	return data, contentType, nil
}

func (s *Service) Delete(ctx context.Context, userID int64, isManager bool, key string) error {
	image, err := s.repo.FindByKey(ctx, key)
	if err != nil {
		return err
	}
	if !isManager && image.UserID != userID {
		return ErrForbidden
	}
	if err := s.repo.SoftDelete(ctx, key); err != nil {
		return err
	}
	return s.storage.Delete(ctx, key)
}

func randomKey() (string, error) {
	buffer := make([]byte, 16)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}

func extensionFor(format imageproc.Format) string {
	switch format {
	case imageproc.PNG:
		return "png"
	case imageproc.GIF:
		return "gif"
	default:
		return "jpg"
	}
}

func mimeFor(format imageproc.Format) string {
	switch format {
	case imageproc.PNG:
		return "image/png"
	case imageproc.GIF:
		return "image/gif"
	default:
		return "image/jpeg"
	}
}

func formatFromMIME(mime string) imageproc.Format {
	switch mime {
	case "image/png":
		return imageproc.PNG
	case "image/gif":
		return imageproc.GIF
	default:
		return imageproc.JPEG
	}
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}
	return string(runes[:limit])
}
