// internal/utils/storage.go
package utils

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
)

type Storage struct {
	baseDir string
}

func NewStorage(baseDir string) *Storage {
	return &Storage{baseDir: baseDir}
}

func (s *Storage) Save(data []byte) (string, error) {
	hash := sha256.Sum256(data)
	hashStr := hex.EncodeToString(hash[:])

	path := filepath.Join(s.baseDir, hashStr[:2], hashStr[2:4], hashStr)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return "", err
	}

	return hashStr, os.WriteFile(path, data, 0644)
}

func (s *Storage) Get(hash string) ([]byte, error) {
	path := filepath.Join(s.baseDir, hash[:2], hash[2:4], hash)
	return os.ReadFile(path)
}
