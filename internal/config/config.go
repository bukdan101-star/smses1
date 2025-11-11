package config

import (
	"errors"
	"os"
	"strconv"
)

type Config struct {
	DBHost        string
	DBPort        string
	DBUser        string
	DBPass        string
	DBName        string
	DBSSLMode     string
	JWTSecret     string
	Port          string
	Env           string
	QRDir         string
	LogoDir       string
	MaxUploadSize int64
	LogLevel      string
}

func NewConfigFromEnv() (*Config, error) {
	maxUploadSize, _ := strconv.ParseInt(getenv("MAX_UPLOAD_SIZE", "10485760"), 10, 64)

	cfg := &Config{
		DBHost:        getenv("DB_HOST", "localhost"),
		DBPort:        getenv("DB_PORT", "5432"),
		DBUser:        getenv("DB_USER", "postgres"),
		DBPass:        getenv("DB_PASSWORD", "postgres"),
		DBName:        getenv("DB_NAME", "eventdb"),
		DBSSLMode:     getenv("DB_SSLMODE", "disable"),
		JWTSecret:     getenv("JWT_SECRET", ""),
		Port:          getenv("PORT", "3000"),
		Env:           getenv("ENV", "development"),
		QRDir:         getenv("QR_DIR", "./uploads/qrcodes"),
		LogoDir:       getenv("LOGO_DIR", "./uploads/logos"),
		MaxUploadSize: maxUploadSize,
		LogLevel:      getenv("LOG_LEVEL", "info"),
	}

	if cfg.JWTSecret == "" {
		return nil, errors.New("JWT_SECRET is required")
	}

	return cfg, nil
}

func getenv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
