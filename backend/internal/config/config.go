package config

import (
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	ServerHost  string
	Environment string
	LogLevel    string

	DatabaseURL string
	RedisURL    string

	JWTSecret     string
	JWTAccessTTL  time.Duration
	JWTRefreshTTL time.Duration

	BackendKEK string

	MinioEndpoint  string
	MinioPort      string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool

	APIURL      string
	CORSOrigins string

	MetaAppID            string
	MetaAppSecret        string
	MetaGraphVersion     string
	InstagramVerifyToken string
	FacebookVerifyToken  string
	MetaAllowUnsigned    bool

	WidgetPublicBaseURL string
	WidgetJWTSecret     string
	WidgetSessionTTL    int

	FeatureChannelTiktok         bool
	FeatureChannelTwitter        bool
	FeatureChannelTwilioWhatsapp bool
	FeatureTwilioSmsMedium       bool

	TiktokClientKey       string
	TiktokClientSecret    string
	TwitterConsumerKey    string
	TwitterConsumerSecret string
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
		JWTAccessTTL:  mustDuration("JWT_ACCESS_TTL", "15m"),
		JWTRefreshTTL: mustDuration("JWT_REFRESH_TTL", "720h"),

		BackendKEK: getEnv("BACKEND_KEK", ""),

		MinioEndpoint:  getEnv("MINIO_ENDPOINT", "localhost"),
		MinioPort:      getEnv("MINIO_PORT", "9000"),
		MinioAccessKey: getEnv("MINIO_ACCESS_KEY", ""),
		MinioSecretKey: getEnv("MINIO_SECRET_KEY", ""),
		MinioBucket:    getEnv("MINIO_BUCKET", "backend-media"),
		MinioUseSSL:    getEnvAsBool("MINIO_USE_SSL", false),

		APIURL:      getEnv("API_URL", "http://localhost:3001"),
		CORSOrigins: getEnv("CORS_ORIGINS", "*"),

		MetaAppID:            getEnv("META_APP_ID", ""),
		MetaAppSecret:        getEnv("META_APP_SECRET", ""),
		MetaGraphVersion:     getEnv("META_GRAPH_VERSION", "v22.0"),
		InstagramVerifyToken: getEnv("INSTAGRAM_VERIFY_TOKEN", ""),
		FacebookVerifyToken:  getEnv("FB_VERIFY_TOKEN", ""),
		MetaAllowUnsigned:    getEnvAsBool("META_ALLOW_UNSIGNED", false),

		WidgetPublicBaseURL: getEnv("WIDGET_PUBLIC_BASE_URL", "http://localhost:3001"),
		WidgetJWTSecret:     getEnv("WIDGET_JWT_SECRET", ""),
		WidgetSessionTTL:    getEnvAsInt("WIDGET_SESSION_TTL_DAYS", 30),

		FeatureChannelTiktok:         getEnvAsBool("FEATURE_CHANNEL_TIKTOK", false),
		FeatureChannelTwitter:        getEnvAsBool("FEATURE_CHANNEL_TWITTER", false),
		FeatureChannelTwilioWhatsapp: getEnvAsBool("FEATURE_CHANNEL_TWILIO_WHATSAPP", true),
		FeatureTwilioSmsMedium:       getEnvAsBool("FEATURE_TWILIO_SMS_MEDIUM", false),

		TiktokClientKey:       getEnv("TIKTOK_CLIENT_KEY", ""),
		TiktokClientSecret:    getEnv("TIKTOK_CLIENT_SECRET", ""),
		TwitterConsumerKey:    getEnv("TWITTER_CONSUMER_KEY", ""),
		TwitterConsumerSecret: getEnv("TWITTER_CONSUMER_SECRET", ""),
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

	if c.FeatureChannelTiktok {
		if c.TiktokClientKey == "" {
			missing = append(missing, "TIKTOK_CLIENT_KEY (required when FEATURE_CHANNEL_TIKTOK=true)")
		}
		if c.TiktokClientSecret == "" {
			missing = append(missing, "TIKTOK_CLIENT_SECRET (required when FEATURE_CHANNEL_TIKTOK=true)")
		}
	}

	if c.FeatureChannelTwitter {
		if c.TwitterConsumerKey == "" {
			missing = append(missing, "TWITTER_CONSUMER_KEY (required when FEATURE_CHANNEL_TWITTER=true)")
		}
		if c.TwitterConsumerSecret == "" {
			missing = append(missing, "TWITTER_CONSUMER_SECRET (required when FEATURE_CHANNEL_TWITTER=true)")
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

// mustDuration parses env[key] as time.Duration or exits with a clear error.
// Never silently falls back on parse failure (a malformed TTL would issue
// zero-TTL JWTs without this guard).
func mustDuration(key, fallback string) time.Duration {
	raw := getEnv(key, fallback)
	d, err := time.ParseDuration(raw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "config: %s must be a valid Go duration (e.g. 15m, 720h); got %q: %s\n", key, raw, err)
		os.Exit(1)
	}
	if d <= 0 {
		fmt.Fprintf(os.Stderr, "config: %s must be > 0; got %s\n", key, d)
		os.Exit(1)
	}
	return d
}

func getEnvAsBool(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return fallback
}

func getEnvAsInt(key string, fallback int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return fallback
}
