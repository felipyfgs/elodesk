// @title           Backend API
// @version         1.0
// @description     Chatwoot-compatible backend API for multi-channel messaging
// @contact.name    Backend Support
// @license.name    MIT
// @license.url     https://opensource.org/licenses/MIT
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Bearer JWT token
// @securityDefinitions.apikey ApiAccessToken
// @in header
// @name api_access_token
// @description Channel API access token

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/logger"
	"backend/internal/server"

	"github.com/redis/go-redis/v9"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, cfg.Environment)

	ctx := context.Background()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db.Pool); err != nil {
		logger.Fatal().Err(err).Msg("Failed to run database migrations")
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal().Err(err).Msg("Failed to connect to Redis")
	}
	logger.Info().Str("component", "redis").Msg("Successfully connected to Redis")

	srv := server.New(cfg)
	asynqClient, err := srv.SetupRoutes(cfg, db, redisClient)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to setup routes")
	}
	defer func() {
		if err := asynqClient.Close(); err != nil {
			logger.Warn().Err(err).Msg("asynq client close error")
		}
	}()

	go func() {
		if err := srv.Start(); err != nil {
			logger.Fatal().Err(err).Msg("Server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error().Err(err).Msg("Server forced to shutdown")
	}
}
