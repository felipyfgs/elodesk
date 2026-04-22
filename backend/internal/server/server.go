package server

import (
	"context"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"

	twiliochan "backend/internal/channel/twilio"
	"backend/internal/config"
	"backend/internal/dto"
	"backend/internal/logger"
	"backend/internal/service"
)

type Server struct {
	App    *fiber.App
	Config *config.Config
	ctx    context.Context
	cancel context.CancelFunc

	slaBreachJob       *service.SLABreachJob
	auditRetentionJob  *service.AuditRetentionJob
	twilioTemplatesJob *twiliochan.TemplatesJob
}

func New(cfg *config.Config) *Server {
	app := fiber.New(fiber.Config{
		ServerHeader:          "backend",
		DisableStartupMessage: true,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if fiberErr, ok := err.(*fiber.Error); ok {
				code = fiberErr.Code
			}
			return c.Status(code).JSON(dto.ErrorResp("Error", err.Error()))
		},
	})

	app.Use(recover.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: strings.Join(cfg.CORSOriginsList(), ","),
		AllowHeaders: "Origin, Content-Type, Accept, Authorization, X-Account-Id, api_access_token, user_access_token",
		AllowMethods: "GET, POST, PUT, PATCH, DELETE, OPTIONS",
	}))

	ctx, cancel := context.WithCancel(context.Background())

	return &Server{
		App:    app,
		Config: cfg,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (s *Server) Start() error {
	if s.slaBreachJob != nil {
		s.slaBreachJob.Start(s.ctx)
	}
	if s.auditRetentionJob != nil {
		s.auditRetentionJob.Start(s.ctx)
	}
	if s.twilioTemplatesJob != nil {
		s.twilioTemplatesJob.Start(s.ctx)
	}
	addr := s.Config.ServerHost + ":" + s.Config.Port
	logger.Info().Str("component", "server").Str("addr", addr).Msg("Starting API server")
	return s.App.Listen(addr)
}

func (s *Server) Shutdown(ctx context.Context) error {
	logger.Info().Str("component", "server").Msg("Shutting down API server")
	if s.slaBreachJob != nil {
		s.slaBreachJob.Stop()
	}
	if s.auditRetentionJob != nil {
		s.auditRetentionJob.Stop()
	}
	if s.twilioTemplatesJob != nil {
		s.twilioTemplatesJob.Stop()
	}
	s.cancel()

	done := make(chan error, 1)
	go func() {
		done <- s.App.Shutdown()
	}()

	select {
	case <-ctx.Done():
		logger.Warn().Str("component", "server").Msg("API server shutdown timed out")
		return ctx.Err()
	case err := <-done:
		logger.Info().Str("component", "server").Msg("API server stopped gracefully")
		return err
	}
}
