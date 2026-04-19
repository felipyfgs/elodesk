package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	ServerHost  string
	Environment string
	LogLevel    string

	DatabaseURL string
	RedisURL    string

	JWTSecret    string
	JWTAccessTTL string
	JWTRefreshTTL string

	BackendKEK string

	MinioEndpoint  string
	MinioPort      string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	APIURL      string
	CORSOrigins string
}

func Load() *Config {
	_ = godotenv.Load()

	cfg := &Config{
		Port:        getEnv("PORT", "3001"),
		ServerHost:  getEnv("SERVER_HOST", "0.0.0.0"),
		Environment: getEnv("GO_ENV", "development"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),

		DatabaseURL: getEnv("DATABASE_URL", ""),
		RedisURL:    getEnv("REDIS_URL", ""),

		JWTSecret:     getEnv("JWT_SECRET", ""),
		JWTAccessTTL:  getEnv("JWT_ACCESS_TTL", "15m"),
		JWTRefreshTTL: getEnv("JWT_REFRESH_TTL", "720h"),

		BackendKEK: getEnv("BACKEND_KEK", ""),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost"),
		MinioPort:      getEnv("MINIO_PORT", "9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", ""),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", ""),
		MinioBucket:    getEnv("MINIO_BUCKET", "backend-media"),
		MinioUseSSL:    getEnvAsBool("MINIO_USE_SSL", false),

		APIURL:      getEnv("API_URL", "http://localhost:3001"),
		CORSOrigins: getEnv("CORS_ORIGINS", "*"),
	}

	if err := cfg.validate(); err != nil {
		fmt.Fprintf(os.Stderr, "config validation failed: %s\n", err)
		os.Exit(1)
	}

	return cfg
}

func (c *Config) validate() error {
	var missing []string

	if c.DatabaseURL == "" {
		missing = append(missing, "DATABASE_URL")
	}
	if c.RedisURL == "" {
		missing = append(missing, "REDIS_URL")
	}
	if c.JWTSecret == "" {
		missing = append(missing, "JWT_SECRET")
	} else if len(c.JWTSecret) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters (got %d)", len(c.JWTSecret))
	}
	if c.BackendKEK == "" {
		missing = append(missing, "BACKEND_KEK")
	} else {
		kekBytes, err := base64.StdEncoding.DecodeString(c.BackendKEK)
		if err != nil {
			return fmt.Errorf("BACKEND_KEK must be valid base64: %w", err)
		}
		if len(kekBytes) < 32 {
			return fmt.Errorf("BACKEND_KEK must decode to at least 32 bytes (got %d)", len(kekBytes))
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	return nil
}

func (c *Config) CORSOriginsList() []string {
	if c.CORSOrigins == "*" {
		return []string{"*"}
	}
	parts := strings.Split(c.CORSOrigins, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getEnvAsBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return fallback
}
