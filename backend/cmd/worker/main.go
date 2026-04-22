package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"backend/internal/channel/whatsapp"
	"backend/internal/config"
	appcrypto "backend/internal/crypto"
	"backend/internal/database"
	"backend/internal/logger"
	"backend/internal/realtime"
	"backend/internal/webhook"
)

func main() {
	cfg := config.Load()
	logger.Init(cfg.LogLevel, cfg.Environment)

	ctx := context.Background()

	db, err := database.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Fatal().Err(err).Msg("Worker: failed to connect to database")
	}
	defer db.Close()

	if err := database.RunMigrations(ctx, db.Pool); err != nil {
		logger.Fatal().Err(err).Msg("Worker: failed to run database migrations")
	}

	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	defer func() { _ = redisClient.Close() }()

	if err := redisClient.Ping(ctx).Err(); err != nil {
		logger.Fatal().Err(err).Msg("Worker: failed to connect to Redis")
	}

	cipher, err := appcrypto.NewCipher(cfg.BackendKEK)
	if err != nil {
		logger.Fatal().Err(err).Msg("Worker: failed to create cipher")
	}

	hub := realtime.NewHub()
	go hub.Run()

	// -- Outbound webhook processor --
	outboundProcessor := webhook.NewOutboundProcessor(cipher)

	// -- WhatsApp send processor --
	waSendProcessor := whatsapp.NewWaSendProcessor(cipher)

	// -- Asynq server --
	srv := asynq.NewServer(
		asynq.RedisClientOpt{Addr: cfg.RedisURL},
		asynq.Config{
			Concurrency: 20,
			Queues: map[string]int{
				"default":  6,
				"critical": 10,
			},
			RetryDelayFunc: retryDelayFunc,
		},
	)

	mux := asynq.NewServeMux()
	mux.HandleFunc(webhook.TypeOutboundWebhook, outboundProcessor.HandleOutboundWebhook)
	mux.HandleFunc(whatsapp.TypeChannelWaSend, waSendProcessor.HandleWaSend)

	logger.Info().Str("component", "worker").Msg("Starting asynq worker")

	go func() {
		if err := srv.Run(mux); err != nil {
			logger.Fatal().Err(err).Msg("Worker: asynq server failed")
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info().Str("component", "worker").Msg("Shutting down worker")
	srv.Shutdown()
}

func retryDelayFunc(n int, err error, task *asynq.Task) time.Duration {
	switch task.Type() {
	case whatsapp.TypeChannelWaSend:
		return whatsapp.WaRetryDelay(n, err, task)
	default:
		return webhook.OutboundRetryDelay(n, err, task)
	}
}
