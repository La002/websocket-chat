package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/La002/websocket-chat/internal/config"
	"github.com/La002/websocket-chat/internal/pubsub"
	"github.com/La002/websocket-chat/internal/websocket"
	"github.com/rs/zerolog/log"
)

func main() {
	cfg := config.Load()

	config.SetupLogger(cfg.Logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	redis, err := pubsub.NewRedisPubSub(cfg.Redis.Addr)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to connect to redis")
	}
	defer redis.Close()

	manager := setupAPI(cfg, ctx, redis)

	addr := fmt.Sprintf(":%s", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info().Str("port", cfg.Server.Port).Msg("starting server")
		serverErrors <- server.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		log.Fatal().Err(err).Msg("server failed to start")

	case sig := <-shutdown:
		log.Info().Str("signal", sig.String()).Msg("shutdown signal received")

		cancel()

		shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCancel()

		log.Info().Msg("closing websocket connections")
		manager.Shutdown()

		log.Info().Msg("shutting down server")
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("server shutdown failed")
			server.Close()
		}

		log.Info().Msg("server stopped gracefully")
	}
}

func setupAPI(cfg *config.Config, ctx context.Context, redis *pubsub.RedisPubSub) *websocket.Manager {
	manager := websocket.NewManager(ctx, cfg, redis)

	http.HandleFunc("/ws", manager.ServeWS)
	http.Handle("/", http.FileServer(http.Dir(cfg.Server.FrontendDir)))
	http.HandleFunc("/login", manager.LoginHandler)

	return manager
}
