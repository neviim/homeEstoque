package config

import (
	"os"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	DBPath      string
	JWTSecret   string
	UploadDir   string
	CORSOrigins []string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Port:      getEnv("PORT", "8080"),
		DBPath:    getEnv("DB_PATH", "./data/homeestoque.db"),
		JWTSecret: getEnv("JWT_SECRET", "dev-secret-change-me"),
		UploadDir: getEnv("UPLOAD_DIR", "./uploads"),
	}

	origins := getEnv("CORS_ORIGINS", "http://localhost:5173")
	cfg.CORSOrigins = strings.Split(origins, ",")

	return cfg
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
