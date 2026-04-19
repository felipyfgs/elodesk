package server

import (
	"time"

	"github.com/gofiber/swagger"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"backend/internal/config"
	"backend/internal/database"
	"backend/internal/handler"
	"backend/internal/logger"
	"backend/internal/middleware"
	"backend/internal/model"
	"backend/internal/realtime"
	"backend/internal/repo"
	"backend/internal/service"
)

func (s *Server) SetupRoutes(cfg *config.Config, db *database.DB, redisClient *redis.Client) {
	userRepo := repo.NewUserRepo(db.Pool)
	accountRepo := repo.NewAccountRepo(db.Pool)
	refreshTokenRepo := repo.NewRefreshTokenRepo(db.Pool)
	inboxRepo := repo.NewInboxRepo(db.Pool)
	channelApiRepo := repo.NewChannelApiRepo(db.Pool)
	contactRepo := repo.NewContactRepo(db.Pool)
	contactInboxRepo := repo.NewContactInboxRepo(db.Pool)
	conversationRepo := repo.NewConversationRepo(db.Pool)
	messageRepo := repo.NewMessageRepo(db.Pool)

	accessTTL, _ := time.ParseDuration(cfg.JWTAccessTTL)
	refreshTTL, _ := time.ParseDuration(cfg.JWTRefreshTTL)

	authSvc := service.NewAuthService(userRepo, accountRepo, refreshTokenRepo, cfg.JWTSecret, accessTTL, refreshTTL)
	authHandler := handler.NewAuthHandler(authSvc)

	inboxSvc := service.NewInboxService(inboxRepo, channelApiRepo)
	inboxHandler := handler.NewInboxHandler(inboxSvc)

	contactSvc := service.NewContactService(contactRepo, contactInboxRepo, conversationRepo)
	contactHandler := handler.NewContactHandler(contactSvc, inboxRepo, contactInboxRepo)

	conversationSvc := service.NewConversationService(conversationRepo, contactInboxRepo, contactRepo)
	conversationHandler := handler.NewConversationHandler(conversationSvc, inboxRepo, contactInboxRepo)

	messageSvc := service.NewMessageService(messageRepo)
	messageHandler := handler.NewMessageHandler(messageSvc, inboxRepo, contactInboxRepo)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
	outboundWebhookSvc := service.NewOutboundWebhookService(asynqClient)

	hub := realtime.NewHub()
	go hub.Run()
	realtimeSvc := service.NewRealtimeService(hub)
	realtimeHandler := handler.NewRealtimeHandler(authSvc, hub)

	healthHandler := handler.NewHealthHandler(db, redisClient)

	s.App.Get("/docs/*", swagger.HandlerDefault)
	s.App.Get("/health", healthHandler.Check)

	realtimeHandler.RegisterRoutes(s.App)

	api := s.App.Group("/api/v1")

	auth := api.Group("/auth")
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/logout", middleware.JwtAuth(authSvc), authHandler.Logout)

	jwtAuth := middleware.JwtAuth(authSvc)
	orgScope := middleware.OrgScope(accountRepo)
	ownerAdmin := middleware.RolesRequired(model.RoleOwner, model.RoleAdmin)

	accounts := api.Group("/accounts/:aid", jwtAuth, orgScope)
	accounts.Post("/inboxes", ownerAdmin, inboxHandler.Create)
	accounts.Get("/inboxes", inboxHandler.List)
	accounts.Get("/inboxes/:id", inboxHandler.GetByID)
	accounts.Get("/contacts", contactHandler.Search)
	accounts.Get("/contacts/:id", contactHandler.Get)
	accounts.Get("/conversations", conversationHandler.List)
	accounts.Get("/conversations/:id", conversationHandler.Get)
	accounts.Get("/conversations/:conversationId/messages", messageHandler.List)
	accounts.Delete("/conversations/:conversationId/messages/:messageId", messageHandler.SoftDelete)

	public := s.App.Group("/public/api/v1")
	publicInbox := public.Group("/inboxes/:identifier", middleware.ApiToken(channelApiRepo), middleware.HmacOptional())

	publicInbox.Post("/contacts", contactHandler.CreateContact)
	publicInbox.Get("/contacts/:sourceId", contactHandler.GetContact)
	publicInbox.Put("/contacts/:sourceId", contactHandler.UpdateContact)
	publicInbox.Post("/contacts/:sourceId/conversations", conversationHandler.Create)
	publicInbox.Post("/contact_inboxes/conversations/:cid/update_last_seen", conversationHandler.UpdateLastSeen)
	publicInbox.Get("/contacts/:sourceId/conversations/:conversationId/messages", messageHandler.ListPublic)
	publicInbox.Post("/contacts/:sourceId/conversations/:conversationId/messages", messageHandler.Create)

	_ = outboundWebhookSvc
	_ = realtimeSvc

	logger.Info().Str("component", "server").Msg("Routes registered")
}
