// internal/utils/image.go
package utils

import (
	"image"
	"image/jpeg"
	"os"

	"github.com/nfnt/resize"
)

func CreateThumbnail(srcPath, dstPath string, width uint) error {
	file, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		return err
	}

	thumbnail := resize.Resize(width, 0, img, resize.Lanczos3)
	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	return jpeg.Encode(out, thumbnail, &jpeg.Options{Quality: 85})
}
