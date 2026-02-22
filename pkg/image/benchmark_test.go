package image

import (
	"bytes"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"
)

func BenchmarkAllowedFormats(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = AllowedFormats()
	}
}

func BenchmarkDetectFormat_JPEG(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = detectFormat("image.jpg")
	}
}

func BenchmarkDetectFormat_PNG(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = detectFormat("photo.png")
	}
}

func BenchmarkDetectFormat_Unknown(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = detectFormat("document.pdf")
	}
}

func BenchmarkDetectFormat_LongPath(b *testing.B) {
	filename := "/very/long/path/to/some/deeply/nested/directory/image.jpeg"
	for i := 0; i < b.N; i++ {
		_ = detectFormat(filename)
	}
}

func BenchmarkNewStore_Default(b *testing.B) {
	dir := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStore(dir)
	}
}

func BenchmarkNewStore_WithOptions(b *testing.B) {
	dir := b.TempDir()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewStore(dir,
			WithMaxSize(20*1024*1024),
			WithMinSize(50),
			WithFormats([]string{"jpeg", "png"}),
		)
	}
}

func BenchmarkWithMaxSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = WithMaxSize(10 * 1024 * 1024)
	}
}

func BenchmarkWithMinSize(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = WithMinSize(100)
	}
}

func BenchmarkWithFormats(b *testing.B) {
	formats := []string{"jpeg", "png", "gif", "webp"}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = WithFormats(formats)
	}
}

func BenchmarkStore_Init(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(filepath.Join(dir, "images"))
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Init()
	}
}

func BenchmarkStore_Save_SmallImage(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	// Create small test data (1KB)
	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, _ = store.Save("test.png", reader)
	}
}

func BenchmarkStore_Save_MediumImage(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	// Create medium test data (100KB)
	data := make([]byte, 100*1024)
	_, _ = rand.Read(data)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(data)
		_, _ = store.Save("test.jpeg", reader)
	}
}

func BenchmarkStore_Exists_Found(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	// Save an image first
	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	img, _ := store.Save("test.png", bytes.NewReader(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Exists(img.ID, img.Format)
	}
}

func BenchmarkStore_Exists_NotFound(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Exists("nonexistent1234", "png")
	}
}

func BenchmarkStore_Get(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	// Save an image first
	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	img, _ := store.Save("test.png", bytes.NewReader(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = store.Get(img.ID, img.Format)
	}
}

func BenchmarkStore_Open(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	// Save an image first
	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	img, _ := store.Save("test.png", bytes.NewReader(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rc, err := store.Open(img.ID, img.Format)
		if err == nil {
			_ = rc.Close()
		}
	}
}

func BenchmarkStore_Delete(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	// Pre-create files to delete
	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	images := make([]*Image, b.N)
	for i := 0; i < b.N; i++ {
		img, _ := store.Save("test.png", bytes.NewReader(data))
		images[i] = img
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = store.Delete(images[i].ID, images[i].Format)
	}
}

func BenchmarkStore_Save_Parallel(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			reader := bytes.NewReader(data)
			_, _ = store.Save("test.png", reader)
		}
	})
}

func BenchmarkStore_Exists_Parallel(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	// Create multiple images
	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	img, _ := store.Save("test.png", bytes.NewReader(data))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = store.Exists(img.ID, img.Format)
		}
	})
}

// BenchmarkStoragePath measures path construction overhead
func BenchmarkStoragePath(b *testing.B) {
	baseDir := "/var/data/images"
	id := "abcdef1234567890"
	format := "png"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = filepath.Join(baseDir, id[:2], id[2:4], id+"."+format)
	}
}

// BenchmarkCleanup ensures cleanup doesn't affect benchmark timing
func setupStoreWithImages(b *testing.B, count int) (*Store, []*Image) {
	b.Helper()
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	data := make([]byte, 1024)
	_, _ = rand.Read(data)

	images := make([]*Image, count)
	for i := 0; i < count; i++ {
		img, err := store.Save("test.png", bytes.NewReader(data))
		if err != nil {
			// Images may have same hash, that's ok
			continue
		}
		images[i] = img
	}
	return store, images
}

func BenchmarkStore_MultipleExists(b *testing.B) {
	store, images := setupStoreWithImages(b, 100)

	// Filter out nil images
	var validImages []*Image
	for _, img := range images {
		if img != nil {
			validImages = append(validImages, img)
		}
	}

	if len(validImages) == 0 {
		b.Skip("no images created")
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		img := validImages[i%len(validImages)]
		_ = store.Exists(img.ID, img.Format)
	}
}

func BenchmarkDetectFormat_AllFormats(b *testing.B) {
	filenames := []string{
		"image.jpg",
		"image.jpeg",
		"image.png",
		"image.gif",
		"image.webp",
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = detectFormat(filenames[i%len(filenames)])
	}
}

func BenchmarkStore_DirectoryTraversal(b *testing.B) {
	dir := b.TempDir()
	store := NewStore(dir)
	_ = store.Init()

	// Create some test images
	data := make([]byte, 1024)
	_, _ = rand.Read(data)
	_, _ = store.Save("test.png", bytes.NewReader(data))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate checking multiple IDs that might exist
		_ = store.Exists("0123456789abcdef", "png")
		_ = store.Exists("fedcba9876543210", "png")
		_ = store.Exists("abcdef0123456789", "png")
	}
}

func BenchmarkImageInit_CleanDirectory(b *testing.B) {
	for i := 0; i < b.N; i++ {
		dir := filepath.Join(os.TempDir(), "bench-image-init")
		store := NewStore(dir)
		_ = store.Init()
		_ = os.RemoveAll(dir)
	}
}
