package media

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

type repoStub struct {
	images  map[string]Image
	deleted []string
}

func (r *repoStub) Create(_ context.Context, image Image) error {
	if r.images == nil {
		r.images = map[string]Image{}
	}
	r.images[image.Key] = image
	return nil
}

func (r *repoStub) FindByKey(_ context.Context, key string) (Image, error) {
	image, ok := r.images[key]
	if !ok {
		return Image{}, ErrNotFound
	}
	return image, nil
}

func (r *repoStub) SoftDelete(_ context.Context, key string) error {
	r.deleted = append(r.deleted, key)
	return nil
}

type storageStub struct {
	objects map[string][]byte
	deleted []string
}

func (s *storageStub) Put(_ context.Context, key string, data []byte) error {
	if s.objects == nil {
		s.objects = map[string][]byte{}
	}
	s.objects[key] = data
	return nil
}

func (s *storageStub) Get(_ context.Context, key string) ([]byte, error) {
	data, ok := s.objects[key]
	if !ok {
		return nil, ErrNotFound
	}
	return data, nil
}

func (s *storageStub) Delete(_ context.Context, key string) error {
	s.deleted = append(s.deleted, key)
	return nil
}

func pngBytes(t *testing.T, width, height int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.NRGBA{R: uint8(x % 255), G: uint8(y % 255), B: 120, A: 255})
		}
	}
	var buffer bytes.Buffer
	if err := png.Encode(&buffer, img); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func TestUploadContentProducesThreeVariants(t *testing.T) {
	repo := &repoStub{}
	storage := &storageStub{}
	service := NewService(repo, storage, "/media")

	data := pngBytes(t, 1400, 800)
	result, err := service.Upload(context.Background(), 7, KindContent, "图片.png", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.URLs) != 3 {
		t.Fatalf("expected 3 variants, got %#v", result.URLs)
	}
	for _, name := range []string{"original", "release", "thumb"} {
		if _, ok := storage.objects[result.Key+"/"+name+".png"]; !ok {
			t.Fatalf("missing variant %s: %#v", name, keysOf(storage.objects))
		}
		if !strings.HasPrefix(result.URLs[name], "/media/"+result.Key+"/") {
			t.Fatalf("unexpected url: %s", result.URLs[name])
		}
	}
	if result.Width != 1400 || result.Height != 800 {
		t.Fatalf("unexpected dimensions: %dx%d", result.Width, result.Height)
	}
	if repo.images[result.Key].Kind != KindContent {
		t.Fatal("image metadata not persisted")
	}
}

func TestAvatarRejectsGIFAndProducesSquareVariants(t *testing.T) {
	service := NewService(&repoStub{}, &storageStub{}, "/media")

	if _, err := service.Upload(context.Background(), 1, KindAvatar, "a.gif", []byte("GIF89a")); err == nil {
		t.Fatal("gif avatar accepted")
	}
	result, err := service.Upload(context.Background(), 1, KindAvatar, "a.png", pngBytes(t, 900, 600))
	if err != nil {
		t.Fatal(err)
	}
	if len(result.URLs) != 3 {
		t.Fatalf("expected 3 avatar variants, got %#v", result.URLs)
	}
	for _, name := range []string{"small", "middle", "large"} {
		if _, ok := result.URLs[name]; !ok {
			t.Fatalf("missing avatar variant %s", name)
		}
	}
}

func TestUploadRejectsNonImageAndOversized(t *testing.T) {
	service := NewService(&repoStub{}, &storageStub{}, "/media")
	if _, err := service.Upload(context.Background(), 1, KindContent, "x.png", []byte("<html>not an image</html>")); err != ErrInvalidImage {
		t.Fatalf("expected invalid image, got %v", err)
	}
	huge := make([]byte, contentMaxBytes+1)
	if _, err := service.Upload(context.Background(), 1, KindContent, "big.png", huge); err != ErrTooLarge {
		t.Fatalf("expected too large, got %v", err)
	}
	if _, err := service.Upload(context.Background(), 1, "unknown", "a.png", pngBytes(t, 10, 10)); err == nil {
		t.Fatal("unknown kind accepted")
	}
}

func TestOpenAndDeletePermissions(t *testing.T) {
	repo := &repoStub{}
	storage := &storageStub{}
	service := NewService(repo, storage, "/media")
	result, err := service.Upload(context.Background(), 5, KindContent, "a.png", pngBytes(t, 300, 200))
	if err != nil {
		t.Fatal(err)
	}

	data, contentType, err := service.Open(context.Background(), result.Key, "thumb")
	if err != nil {
		t.Fatal(err)
	}
	if len(data) == 0 || contentType != "image/png" {
		t.Fatalf("unexpected image payload: %d %s", len(data), contentType)
	}
	if _, _, err := service.Open(context.Background(), result.Key, "large"); err != ErrNotFound {
		t.Fatalf("invalid variant accepted: %v", err)
	}
	if _, _, err := service.Open(context.Background(), "deadbeef", "thumb"); err != ErrNotFound {
		t.Fatalf("unknown key accepted: %v", err)
	}

	if err := service.Delete(context.Background(), 6, false, result.Key); err != ErrForbidden {
		t.Fatalf("foreign delete allowed: %v", err)
	}
	if err := service.Delete(context.Background(), 5, false, result.Key); err != nil {
		t.Fatal(err)
	}
	if len(storage.deleted) != 1 || len(repo.deleted) != 1 {
		t.Fatalf("delete not propagated: %#v %#v", storage.deleted, repo.deleted)
	}
}

func TestFileStorageRejectsPathTraversal(t *testing.T) {
	storage, err := NewFileStorage(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := storage.Put(context.Background(), "../escape.png", []byte("x")); err == nil {
		t.Fatal("path traversal accepted")
	}
	if err := storage.Put(context.Background(), "abc/thumb.png", []byte("x")); err != nil {
		t.Fatal(err)
	}
	if data, err := storage.Get(context.Background(), "abc/thumb.png"); err != nil || string(data) != "x" {
		t.Fatalf("round trip failed: %v", err)
	}
}

func keysOf(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}
