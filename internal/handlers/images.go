// internal/handlers/images.go
package handlers

import (
	"image"
	"image/jpeg"
	"image/png"
	"io"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

type ImageProcessor struct {
	uploadDir   string
	thumbsDir   string
	maxWidth    uint
	jpegQuality int
}

func NewImageProcessor() *ImageProcessor {
	return &ImageProcessor{
		uploadDir:   "./uploads/images",
		thumbsDir:   "./uploads/thumbnails",
		maxWidth:    2048,
		jpegQuality: 85,
	}
}

func (ip *ImageProcessor) ProcessImage(src io.Reader, hash string) error {
	// Decode image
	img, format, err := image.Decode(src)
	if err != nil {
		return err
	}

	// Create directories
	for _, dir := range []string{ip.uploadDir, ip.thumbsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Resize main image if needed
	bounds := img.Bounds()
	if bounds.Dx() > int(ip.maxWidth) {
		img = resize.Resize(ip.maxWidth, 0, img, resize.Lanczos3)
	}

	// Save main image
	ext := ".jpg"
	if format == "png" {
		ext = ".png"
	}
	mainPath := filepath.Join(ip.uploadDir, hash+ext)
	mainFile, err := os.Create(mainPath)
	if err != nil {
		return err
	}
	defer mainFile.Close()

	// Create thumbnail
	thumb := resize.Resize(300, 0, img, resize.Lanczos3)
	thumbPath := filepath.Join(ip.thumbsDir, hash+ext)
	thumbFile, err := os.Create(thumbPath)
	if err != nil {
		return err
	}
	defer thumbFile.Close()

	// Save both versions
	if format == "png" {
		png.Encode(mainFile, img)
		png.Encode(thumbFile, thumb)
	} else {
		jpeg.Encode(mainFile, img, &jpeg.Options{Quality: ip.jpegQuality})
		jpeg.Encode(thumbFile, thumb, &jpeg.Options{Quality: ip.jpegQuality})
	}

	return nil
}
