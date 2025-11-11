package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func ValidateImageFile(file *multipart.FileHeader) error {
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/gif":  true,
	}

	contentType := file.Header.Get("Content-Type")
	if !allowedTypes[contentType] {
		return fmt.Errorf("file type not allowed: %s", contentType)
	}

	return nil
}

func GenerateUniqueFilename(originalName string) string {
	ext := filepath.Ext(originalName)
	filename := strings.TrimSuffix(originalName, ext)
	return fmt.Sprintf("%s_%s%s", filename, uuid.New().String(), ext)
}

func SaveUploadedFile(file *multipart.FileHeader, destDir, filename string) error {
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	destPath := filepath.Join(destDir, filename)
	src, err := file.Open()
	if err != nil {
		return fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(destPath)
	if err != nil {
		return fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("failed to save file: %w", err)
	}

	return nil
}
