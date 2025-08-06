package storage

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
)

type Storage interface {
	Store(data []byte, filename string) (url string, hash []byte, err error)
	Get(url string) ([]byte, error)
	Delete(url string) error
}

type localStorage struct {
	basePath string
	baseURL  string
}

func NewLocalStorage(basePath, baseURL string) Storage {
	return &localStorage{
		basePath: basePath,
		baseURL:  baseURL,
	}
}

func (s *localStorage) Store(data []byte, filename string) (string, []byte, error) {
	hash := sha256.Sum256(data)
	hashStr := fmt.Sprintf("%x", hash)

	// organize by first 2 chars of hash to avoid too many files per directory
	dir := filepath.Join(s.basePath, hashStr[:2])
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", nil, fmt.Errorf("creating storage directory: %w", err)
	}

	filePath := filepath.Join(dir, hashStr+filepath.Ext(filename))
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return "", nil, fmt.Errorf("writing file: %w", err)
	}

	url := s.baseURL + "/" + hashStr[:2] + "/" + hashStr + filepath.Ext(filename)
	return url, hash[:], nil
}

func (s *localStorage) Get(url string) ([]byte, error) {
	relativePath := url[len(s.baseURL):]
	fullPath := filepath.Join(s.basePath, relativePath)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	return data, nil
}

func (s *localStorage) Delete(url string) error {
	relativePath := url[len(s.baseURL):]
	fullPath := filepath.Join(s.basePath, relativePath)

	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("deleting file: %w", err)
	}

	return nil
}
