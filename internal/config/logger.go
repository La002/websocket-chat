package config

import (
	"os"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// SetupLogger configures the global logger based on config
func SetupLogger(cfg LoggerConfig) {
	// Development: pretty console output
	if cfg.Env != "production" {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stdout})
	}

	// Set log level
	switch cfg.Level {
	case "debug":
		zerolog.SetGlobalLevel(zerolog.DebugLevel)
	case "info":
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	case "warn":
		zerolog.SetGlobalLevel(zerolog.WarnLevel)
	case "error":
		zerolog.SetGlobalLevel(zerolog.ErrorLevel)
	default:
		zerolog.SetGlobalLevel(zerolog.InfoLevel)
	}
}
