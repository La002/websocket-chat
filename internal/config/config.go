package config

import (
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
)

// Config holds all application configuration
type Config struct {
	Server ServerConfig
	Logger LoggerConfig
	JWT    JWTConfig
}

// ServerConfig holds server-specific settings
type ServerConfig struct {
	Port         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	FrontendDir  string
}

// LoggerConfig holds logging configuration
type LoggerConfig struct {
	Level string // debug, info, warn, error
	Env   string // development, production
}

type JWTConfig struct {
	AccessPrivateKey     string        `mapstructure:"access_private_key"`
	AccessPublicKey      string        `mapstructure:"access_public_key"`
	AccessTokenExpiredIn time.Duration `mapstructure:"access_token_expired_in"`
	AccessTokenMaxAge    time.Duration `mapstructure:"access_token_max_age"`
}

// Load reads configuration from .env file and environment variables with sensible defaults
func Load() *Config {
	// Load .env file if it exists (optional - won't error if missing)
	if err := godotenv.Load(); err != nil {
		// In production, .env might not exist (using real env vars)
		// So we only log at debug level
		log.Debug().Msg("no .env file found, using system environment variables")
	}

	return &Config{
		Server: ServerConfig{
			Port:         getEnv("PORT", "8080"),
			ReadTimeout:  15 * time.Second,
			WriteTimeout: 15 * time.Second,
			FrontendDir:  getEnv("FRONTEND_DIR", "./frontend"),
		},
		Logger: LoggerConfig{
			Level: getEnv("LOG_LEVEL", "info"),
			Env:   getEnv("ENV", "development"),
		},
		JWT: JWTConfig{
			AccessPrivateKey:     "private_key",
			AccessPublicKey:      "private_key",
			AccessTokenExpiredIn: time.Minute * 60,
			AccessTokenMaxAge:    time.Minute * 60,
		},
	}
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultVal string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultVal
}
