package main

import (
	"embed"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static/* templates/*
var content embed.FS

var dirs = []string{
	"data/db",
	"data/uploads/images",
	"data/uploads/thumbnails",
	"templates/admin", // Non-themed admin templates
	"templates/themes/default",
	"templates/themes/minimal",
}

func initializeFileSystem() error {
	// Create directories first
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
	}

	// Walk embedded files and copy with special handling
	return fs.WalkDir(content, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if path == "." {
			return nil
		}

		destPath := path
		if strings.HasPrefix(path, "templates/") {
			if strings.Contains(path, "admin/") {
				// Admin templates go to templates/admin
				destPath = filepath.Join("templates", strings.TrimPrefix(path, "templates/"))
			} else if !strings.Contains(path, "themes/") {
				// Non-admin templates go to default theme
				destPath = filepath.Join("templates/themes/default", strings.TrimPrefix(path, "templates/"))
			}
		}

		if d.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		data, err := content.ReadFile(path)
		if err != nil {
			return err
		}

		log.Printf("Creating: %s", destPath)
		return os.WriteFile(destPath, data, 0644)
	})
}
