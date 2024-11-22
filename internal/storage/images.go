// internal/storage/images.go
package storage

import (
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"

	"github.com/nfnt/resize"
)

type ImageProcessor struct {
	uploadDir string
	sizes     map[string]uint
}

func NewImageProcessor(uploadDir string) *ImageProcessor {
	return &ImageProcessor{
		uploadDir: uploadDir,
		sizes: map[string]uint{
			"thumb": 300,
			"large": 1200,
		},
	}
}

func (ip *ImageProcessor) ProcessImage(src string, hash string) error {
	img, err := os.Open(src)
	if err != nil {
		return err
	}
	defer img.Close()

	decoded, format, err := image.Decode(img)
	if err != nil {
		return err
	}

	for suffix, maxSize := range ip.sizes {
		resized := resize.Resize(maxSize, 0, decoded, resize.Lanczos3)

		outPath := filepath.Join(ip.uploadDir, hash+"_"+suffix+"."+format)
		out, err := os.Create(outPath)
		if err != nil {
			return err
		}
		defer out.Close()

		switch format {
		case "jpeg":
			err = jpeg.Encode(out, resized, &jpeg.Options{Quality: 85})
		case "png":
			err = png.Encode(out, resized)
		}
		if err != nil {
			return err
		}
	}

	return nil
}
