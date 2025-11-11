package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/skip2/go-qrcode"
)

func GenerateQRCodeImage(content, dirPath string) (string, error) {
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", fmt.Errorf("failed to create QR directory: %w", err)
	}

	filename := fmt.Sprintf("%s.png", uuid.New().String())
	fullPath := filepath.Join(dirPath, filename)

	if err := qrcode.WriteFile(content, qrcode.Medium, 256, fullPath); err != nil {
		return "", fmt.Errorf("failed to generate QR code: %w", err)
	}

	return filename, nil
}

func ExtractUUIDFromQRPath(qrPath string) (string, error) {
	filename := filepath.Base(qrPath)
	uuidStr := filepath.Ext(filename)
	if uuidStr == "" {
		return "", fmt.Errorf("invalid QR path format")
	}

	// Remove extension
	uuidStr = filename[:len(filename)-len(filepath.Ext(filename))]

	if _, err := uuid.Parse(uuidStr); err != nil {
		return "", fmt.Errorf("invalid UUID in QR path: %w", err)
	}

	return uuidStr, nil
}
