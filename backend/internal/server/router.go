package server

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"github.com/gofiber/swagger"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"

	"backend/internal/audit"
	appchannel "backend/internal/channel"
	apichan "backend/internal/channel/api"
	fbchan "backend/internal/channel/facebook"
	igchan "backend/internal/channel/instagram"
	linechan "backend/internal/channel/line"
	"backend/internal/channel/reauth"
	smschan "backend/internal/channel/sms"
	bandwidth "backend/internal/channel/sms/bandwidth"
	smstwilio "backend/internal/channel/sms/twilio"
	zenvia "backend/internal/channel/sms/zenvia"
	tgchan "backend/internal/channel/telegram"
	tiktokchan "backend/internal/channel/tiktok"
	twiliochan "backend/internal/channel/twilio"
	twitterchan "backend/internal/channel/twitter"
	"backend/internal/channel/webwidget"
	whatsappchan "backend/internal/channel/whatsapp"
	"backend/internal/config"
	appcrypto "backend/internal/crypto"
	"backend/internal/database"
	"backend/internal/handler"
	"backend/internal/logger"
	"backend/internal/media"
	"backend/internal/middleware"
	"backend/internal/model"
	"backend/internal/realtime"
	"backend/internal/repo"
	"backend/internal/service"
)

func (s *Server) SetupRoutes(cfg *config.Config, db *database.DB, redisClient *redis.Client) (*asynq.Client, error) {
	cipher, err := appcrypto.NewCipher(cfg.BackendKEK)
	if err != nil {
		return nil, err
	}

	userRepo := repo.NewUserRepo(db.Pool)
	accountRepo := repo.NewAccountRepo(db.Pool)
	refreshTokenRepo := repo.NewRefreshTokenRepo(db.Pool)
	userAccessTokenRepo := repo.NewUserAccessTokenRepo(db.Pool)
	inboxRepo := repo.NewInboxRepo(db.Pool)
	channelApiRepo := repo.NewChannelAPIRepo(db.Pool)
	channelWhatsAppRepo := repo.NewChannelWhatsAppRepo(db.Pool)
	inboxAgentRepo := repo.NewInboxAgentRepo(db.Pool)
	inboxBusinessHoursRepo := repo.NewInboxBusinessHoursRepo(db.Pool)
	contactRepo := repo.NewContactRepo(db.Pool)
	contactInboxRepo := repo.NewContactInboxRepo(db.Pool)
	conversationRepo := repo.NewConversationRepo(db.Pool)
	messageRepo := repo.NewMessageRepo(db.Pool)
	attachmentRepo := repo.NewAttachmentRepo(db.Pool)

	labelRepo := repo.NewLabelRepo(db.Pool)
	teamRepo := repo.NewTeamRepo(db.Pool)
	teamMemberRepo := repo.NewTeamMemberRepo(db.Pool)
	cannedResponseRepo := repo.NewCannedResponseRepo(db.Pool)
	noteRepo := repo.NewNoteRepo(db.Pool)
	customAttrDefRepo := repo.NewCustomAttributeDefinitionRepo(db.Pool)
	customFilterRepo := repo.NewCustomFilterRepo(db.Pool)
	passwordResetTokenRepo := repo.NewPasswordResetTokenRepo(db.Pool)
	mfaRecoveryCodeRepo := repo.NewMfaRecoveryCodeRepo(db.Pool)
	agentRepo := repo.NewAgentRepo(db.Pool)
	agentInvitationRepo := repo.NewAgentInvitationRepo(db.Pool)
	auditLogRepo := repo.NewAuditLogRepo(db.Pool)
	notificationRepo := repo.NewNotificationRepo(db.Pool)

	mfaTokenStore := service.NewInMemoryMfaTokenStore()
	mfaSvc := service.NewMfaService(userRepo, mfaRecoveryCodeRepo, refreshTokenRepo, cipher, mfaTokenStore)

	authSvc := service.NewAuthService(userRepo, accountRepo, refreshTokenRepo, userAccessTokenRepo, mfaSvc, cfg.JWTSecret, cfg.JWTAccessTTL, cfg.JWTRefreshTTL)
	authHandler := handler.NewAuthHandler(authSvc)
	accountHandler := handler.NewAccountHandler(accountRepo)

	passwordRecoverySvc := service.NewPasswordRecoveryService(userRepo, passwordResetTokenRepo, refreshTokenRepo)
	passwordRecoveryHandler := handler.NewPasswordRecoveryHandler(passwordRecoverySvc)

	mfaHandler := handler.NewMfaHandler(mfaSvc, authSvc)

	auditLogger := audit.NewLogger(auditLogRepo)

	agentSvc := service.NewAgentService(agentRepo, agentInvitationRepo, userRepo, accountRepo, authSvc)
	agentsHandler := handler.NewAgentHandler(agentSvc, agentInvitationRepo, auditLogger)

	userProfileSvc := service.NewUserProfileService(userRepo, refreshTokenRepo, auditLogRepo)
	userProfileHandler := handler.NewUserProfileHandler(userProfileSvc)

	userAccessTokenHandler := handler.NewUserAccessTokenHandler(userAccessTokenRepo)

	macroRepo := repo.NewMacroRepo(db.Pool)
	macroSvc := service.NewMacroService(macroRepo, db.Pool)
	macrosHandler := handler.NewMacroHandler(macroSvc, auditLogger)

	slaRepo := repo.NewSLARepo(db.Pool)
	slaSvc := service.NewSLAService(slaRepo)
	slaHandler := handler.NewSLAHandler(slaSvc)

	outboundWebhookRepo := repo.NewOutboundWebhookRepo(db.Pool)
	webhooksHandler := handler.NewWebhookHandler(outboundWebhookRepo, auditLogger, cipher)

	auditLogsHandler := handler.NewAuditLogHandler(auditLogRepo)

	reportsRepo := repo.NewReportsRepo(db.Pool)
	reportsHandler := handler.NewReportHandler(reportsRepo, slaRepo)

	inboxSvc := service.NewInboxService(db.Pool, inboxRepo, channelApiRepo, inboxAgentRepo, inboxBusinessHoursRepo, cipher)
	inboxHandler := handler.NewInboxHandler(inboxSvc, auditLogger)

	contactSvc := service.NewContactService(contactRepo, contactInboxRepo, conversationRepo).
		WithAudit(auditLogger, auditLogRepo)
	contactHandler := handler.NewContactHandler(contactSvc, inboxRepo, contactInboxRepo)
	contactHandler.SetCipher(cipher)

	conversationSvc := service.NewConversationService(conversationRepo, contactInboxRepo, contactRepo, slaRepo, nil)

	messageSvc := service.NewMessageService(messageRepo, attachmentRepo)
	messageSvc.SetConversationRepo(conversationRepo)
	messageSvc.SetContactRepo(contactRepo)
	messageSvc.SetUserRepo(userRepo)
	messageHandler := handler.NewMessageHandler(messageSvc, inboxRepo, contactInboxRepo, messageRepo)
	messageHandler.SetConversationRepo(conversationRepo)
	messageHandler.SetAttachmentRepo(attachmentRepo)

	forwardSvc := service.NewForwardService(messageRepo, attachmentRepo, conversationRepo, contactInboxRepo, contactRepo, inboxRepo, messageSvc, conversationSvc)
	forwardHandler := handler.NewForwardHandler(forwardSvc)

	conversationSvc.SetMessageService(messageSvc)
	conversationHandler := handler.NewConversationHandler(conversationSvc, messageSvc, inboxRepo, contactInboxRepo, conversationRepo, agentRepo, teamRepo, auditLogger)

	participantRepo := repo.NewParticipantRepo(db.Pool)
	participantSvc := service.NewParticipantService(participantRepo)
	participantHandler := handler.NewParticipantHandler(participantSvc)

	asynqClient := asynq.NewClient(asynq.RedisClientOpt{Addr: cfg.RedisURL})
	outboundWebhookSvc := service.NewOutboundWebhookService(asynqClient, cipher).
		WithContactInboxRepo(contactInboxRepo)

	outboundNotifier := service.NewOutboundWebhookNotifier(outboundWebhookSvc, channelApiRepo, inboxRepo, conversationRepo)
	messageSvc.SetOnOutboundHandler(outboundNotifier)
	conversationSvc.SetNotifier(outboundNotifier)

	hub := realtime.NewHub()
	go hub.Run()
	realtimeSvc := service.NewRealtimeService(hub)
	messageSvc.SetRealtimeNotifier(realtimeSvc)
	conversationSvc.SetRealtimeNotifier(realtimeSvc)
	realtimeHandler := handler.NewRealtimeHandler(authSvc, hub, accountRepo, inboxRepo, conversationRepo)

	notificationSvc := service.NewNotificationService(notificationRepo, hub)
	notificationsHandler := handler.NewNotificationHandler(notificationSvc)
	conversationSvc.SetNotifications(notificationSvc)

	s.slaBreachJob = service.NewSLABreachJob(slaRepo, notificationSvc, realtimeSvc, auditLogger, 60*time.Second)
	s.auditRetentionJob = service.NewAuditRetentionJob(auditLogRepo, 90, 24*time.Hour)

	labelsSvc := service.NewLabelService(labelRepo, realtimeSvc).WithAudit(auditLogger)
	labelsHandler := handler.NewLabelHandler(labelsSvc)

	teamsSvc := service.NewTeamService(teamRepo, teamMemberRepo, accountRepo)
	teamsHandler := handler.NewTeamHandler(teamsSvc)

	cannedSvc := service.NewCannedResponseService(cannedResponseRepo)
	cannedHandler := handler.NewCannedResponseHandler(cannedSvc)

	notesSvc := service.NewNoteService(noteRepo, realtimeSvc).WithAudit(auditLogger)
	notesHandler := handler.NewNoteHandler(notesSvc)

	customAttrsSvc := service.NewCustomAttributeService(customAttrDefRepo, contactRepo, conversationRepo).
		WithContactAuditFn(func(ctx context.Context, accountID int64, action string, contactID int64, metadata any) {
			cid := contactID
			auditLogger.Log(ctx, accountID, nil, action, "contact", &cid, metadata, "", "")
		})
	customAttrsHandler := handler.NewCustomAttributeHandler(customAttrsSvc)

	savedFiltersSvc := service.NewSavedFilterService(customFilterRepo, customAttrDefRepo, contactRepo, conversationRepo)
	savedFiltersHandler := handler.NewSavedFilterHandler(savedFiltersSvc, customAttrDefRepo, conversationRepo, db.Pool)

	minioClient, err := media.NewWithPublic(cfg.MinioEndpoint, cfg.MinioPort, cfg.MinioUseSSL, cfg.MinioAccessKey, cfg.MinioSecretKey, cfg.MinioBucket, cfg.MinioPublicEndpoint, cfg.MinioPublicPort, cfg.MinioPublicUseSSL)
	if err != nil {
		return nil, err
	}
	if err := minioClient.EnsureBucket(context.Background()); err != nil {
		return nil, err
	}

	uploadHandler := handler.NewUploadHandler(minioClient, attachmentRepo)
	contactSvc.WithMinio(minioClient)
	messageHandler.SetMinio(minioClient)

	// Padrão Chatwoot/ActiveStorage: o elodesk hospeda a URL pública da mídia
	// e proxia bytes do MinIO. Integradores externos só veem a URL do elodesk.
	// Token HMAC-SHA256 com a KEK (já validada como ≥32 bytes base64) — não há
	// criptografia, só integridade + expiração.
	tokenSecret, _ := base64.StdEncoding.DecodeString(cfg.BackendKEK)
	uploadHandler.SetAttachmentTokenSecret(tokenSecret)
	outboundWebhookSvc.WithAttachmentURLBuilder(func(accountID, attachmentID int64) string {
		token := handler.SignAttachmentToken(tokenSecret, accountID, attachmentID, 15*time.Minute)
		return fmt.Sprintf("%s/api/v1/attachments/%d/file?token=%s", cfg.APIURL, attachmentID, token)
	})

	healthHandler := handler.NewHealthHandler(db, redisClient)

	s.App.Get("/docs/*", swagger.HandlerDefault)
	s.App.Get("/health", healthHandler.Check)

	realtimeHandler.RegisterRoutes(s.App)

	api := s.App.Group("/api/v1")

	// Endpoint público de download de attachment (padrão Chatwoot/ActiveStorage).
	// Sem Bearer — autenticação é o token HMAC na query. Registrado fora do
	// grupo `/auth` e do grupo `/accounts/:aid` (que tem JwtAuth) pra ser
	// alcançável por integradores externos (wzap, n8n) com URL estável.
	api.Get("/attachments/:id/file", uploadHandler.PublicAttachmentDownload)

	auth := api.Group("/auth")
	auth.Get("/setup", authHandler.SetupStatus)
	auth.Post("/register", authHandler.Register)
	auth.Post("/login", authHandler.Login)
	auth.Post("/refresh", authHandler.Refresh)
	auth.Post("/logout", middleware.JwtAuth(authSvc), authHandler.Logout)
	auth.Post("/forgot", passwordRecoveryHandler.Forgot)
	auth.Get("/reset/:token/validate", passwordRecoveryHandler.ValidateResetToken)
	auth.Post("/reset", passwordRecoveryHandler.Reset)
	auth.Post("/mfa/setup", middleware.JwtAuth(authSvc), mfaHandler.Setup)
	auth.Post("/mfa/enable", middleware.JwtAuth(authSvc), mfaHandler.Enable)
	auth.Post("/mfa/verify", mfaHandler.Verify)
	auth.Post("/mfa/disable", middleware.JwtAuth(authSvc), mfaHandler.Disable)
	auth.Post("/invitations/:token/accept", agentsHandler.AcceptInvitation)

	jwtAuth := middleware.JwtAuth(authSvc)
	userAccessTokenAuth := middleware.UserAccessTokenAuth(userAccessTokenRepo, userRepo)
	orgScope := middleware.OrgScope(accountRepo)
	ownerAdmin := middleware.RolesRequired(model.RoleOwner, model.RoleAdmin)
	agentPlus := middleware.RolesRequired(model.RoleOwner, model.RoleAdmin, model.RoleAgent)

	accounts := api.Group("/accounts/:aid", jwtAuth, userAccessTokenAuth, orgScope)

	accounts.Get("", accountHandler.Get)
	accounts.Patch("", ownerAdmin, accountHandler.Update)

	accounts.Post("/inboxes", ownerAdmin, inboxHandler.Create)
	accounts.Post("/inboxes/api", ownerAdmin, inboxHandler.Create)
	accounts.Get("/inboxes/api/:id", agentPlus, inboxHandler.GetChannelAPI)
	accounts.Put("/inboxes/api/:id", ownerAdmin, inboxHandler.UpdateChannelAPI)
	accounts.Post("/inboxes/:id/rotate_token", ownerAdmin, inboxHandler.RotateAPIToken)
	accounts.Get("/inboxes", inboxHandler.List)
	accounts.Get("/inboxes/:id", inboxHandler.GetByID)
	accounts.Put("/inboxes/:id", ownerAdmin, inboxHandler.Update)
	accounts.Delete("/inboxes/:id", ownerAdmin, inboxHandler.Delete)
	accounts.Get("/inboxes/:id/business_hours", agentPlus, inboxHandler.GetBusinessHours)
	accounts.Put("/inboxes/:id/business_hours", ownerAdmin, inboxHandler.UpdateBusinessHours)
	accounts.Get("/inboxes/:id/agents", inboxHandler.ListAgents)
	accounts.Put("/inboxes/:id/agents", agentPlus, inboxHandler.SetAgents)
	accounts.Get("/contacts", contactHandler.Search)
	accounts.Post("/contacts", agentPlus, contactHandler.Create)
	accounts.Post("/contacts/import", ownerAdmin, contactHandler.Import)
	accounts.Get("/contacts/:id", contactHandler.Get)
	accounts.Delete("/contacts/:id", ownerAdmin, contactHandler.Delete)
	accounts.Post("/contacts/:id/merge", ownerAdmin, contactHandler.Merge)
	accounts.Patch("/contacts/:id/block", ownerAdmin, contactHandler.Block)
	accounts.Post("/contacts/:id/avatar", agentPlus, contactHandler.SetAvatar)
	accounts.Delete("/contacts/:id/avatar", agentPlus, contactHandler.DeleteAvatar)
	accounts.Get("/contacts/:id/events", agentPlus, contactHandler.Events)
	accounts.Get("/conversations", conversationHandler.List)
	accounts.Get("/conversations/meta", conversationHandler.Meta)
	accounts.Post("/conversations", agentPlus, conversationHandler.CreateAuthenticated)
	accounts.Get("/conversations/:id", conversationHandler.Get)
	accounts.Delete("/conversations/:id", ownerAdmin, conversationHandler.Delete)
	accounts.Get("/conversations/:conversationId/messages", messageHandler.List)
	accounts.Post("/conversations/:conversationId/messages", agentPlus, messageHandler.CreateAuthenticated)
	accounts.Post("/messages/forward", agentPlus, forwardHandler.Forward)
	accounts.Delete("/conversations/:conversationId/messages/:messageId", messageHandler.SoftDelete)
	accounts.Post("/uploads/signed-url", uploadHandler.SignedUploadURL)
	accounts.Post("/uploads", uploadHandler.ProxyUpload)
	accounts.Get("/uploads/download", uploadHandler.ProxyDownload)
	accounts.Get("/uploads/signed-url", uploadHandler.SignedObjectDownloadURL)
	accounts.Get("/attachments/:id/signed-url", uploadHandler.SignedDownloadURL)

	accounts.Patch("/contacts/:id", contactHandler.UpdateContactByID)
	accounts.Get("/contacts/:id/conversations", contactHandler.ListContactConversations)
	accounts.Get("/conversations/:id/participants", participantHandler.List)
	accounts.Post("/conversations/:id/participants/sync", participantHandler.Sync)
	accounts.Post("/conversations/:id/assignments", agentPlus, conversationHandler.Assign)
	accounts.Patch("/conversations/:id/status", agentPlus, conversationHandler.ToggleStatus)
	accounts.Post("/conversations/:id/update_last_seen", agentPlus, conversationHandler.MarkRead)

	accounts.Post("/conversations/:id/labels", agentPlus, labelsHandler.ApplyToConversation)
	accounts.Delete("/conversations/:id/labels/:labelId", agentPlus, labelsHandler.RemoveFromConversation)
	accounts.Get("/conversations/:id/labels", agentPlus, labelsHandler.ListConversationLabels)

	accounts.Post("/contacts/:id/labels", agentPlus, labelsHandler.ApplyToContact)
	accounts.Delete("/contacts/:id/labels/:labelId", agentPlus, labelsHandler.RemoveFromContact)
	accounts.Get("/contacts/:id/labels", agentPlus, labelsHandler.ListContactLabels)

	accounts.Get("/labels", labelsHandler.List)
	accounts.Post("/labels", ownerAdmin, labelsHandler.Create)
	accounts.Patch("/labels/:id", ownerAdmin, labelsHandler.Update)
	accounts.Delete("/labels/:id", ownerAdmin, labelsHandler.Delete)

	accounts.Get("/teams", teamsHandler.List)
	accounts.Post("/teams", ownerAdmin, teamsHandler.Create)
	accounts.Patch("/teams/:id", ownerAdmin, teamsHandler.Update)
	accounts.Delete("/teams/:id", ownerAdmin, teamsHandler.Delete)
	accounts.Get("/teams/:id/team_members", teamsHandler.ListMembers)
	accounts.Post("/teams/:id/team_members", ownerAdmin, teamsHandler.AddMembers)
	accounts.Delete("/teams/:id/team_members", ownerAdmin, teamsHandler.RemoveMembers)

	accounts.Get("/canned_responses", cannedHandler.List)
	accounts.Post("/canned_responses", ownerAdmin, cannedHandler.Create)
	accounts.Patch("/canned_responses/:id", ownerAdmin, cannedHandler.Update)
	accounts.Delete("/canned_responses/:id", ownerAdmin, cannedHandler.Delete)

	accounts.Get("/contacts/:cid/notes", agentPlus, notesHandler.List)
	accounts.Post("/contacts/:cid/notes", agentPlus, notesHandler.Create)
	accounts.Patch("/contacts/:cid/notes/:nid", agentPlus, notesHandler.Update)
	accounts.Delete("/contacts/:cid/notes/:nid", agentPlus, notesHandler.Delete)

	accounts.Get("/custom_attribute_definitions", customAttrsHandler.ListDefinitions)
	accounts.Post("/custom_attribute_definitions", ownerAdmin, customAttrsHandler.CreateDefinition)
	accounts.Patch("/custom_attribute_definitions/:id", ownerAdmin, customAttrsHandler.UpdateDefinition)
	accounts.Delete("/custom_attribute_definitions/:id", ownerAdmin, customAttrsHandler.DeleteDefinition)

	accounts.Post("/contacts/:id/custom_attributes", agentPlus, customAttrsHandler.SetContactAttributes)
	accounts.Delete("/contacts/:id/custom_attributes", agentPlus, customAttrsHandler.RemoveContactAttributes)
	accounts.Post("/conversations/:id/custom_attributes", agentPlus, customAttrsHandler.SetConversationAttributes)
	accounts.Delete("/conversations/:id/custom_attributes", agentPlus, customAttrsHandler.RemoveConversationAttributes)

	accounts.Get("/custom_filters", savedFiltersHandler.List)
	accounts.Post("/custom_filters", agentPlus, savedFiltersHandler.Create)
	accounts.Patch("/custom_filters/:id", agentPlus, savedFiltersHandler.Update)
	accounts.Delete("/custom_filters/:id", agentPlus, savedFiltersHandler.Delete)

	accounts.Post("/conversations/filter", agentPlus, savedFiltersHandler.FilterConversations)
	accounts.Post("/contacts/filter", agentPlus, savedFiltersHandler.FilterContacts)

	accounts.Get("/agents", ownerAdmin, agentsHandler.List)
	accounts.Post("/agents/invite", ownerAdmin, agentsHandler.Invite)
	accounts.Patch("/agents/:userId", ownerAdmin, agentsHandler.Update)
	accounts.Delete("/agents/:userId", ownerAdmin, agentsHandler.Remove)

	api.Put("/users/:id", jwtAuth, userProfileHandler.UpdateProfile)
	accounts.Get("/profile/access_token", userAccessTokenHandler.GetAccessToken)
	accounts.Post("/profile/access_token/reset", userAccessTokenHandler.ResetAccessToken)

	accounts.Get("/macros", agentPlus, macrosHandler.List)
	accounts.Post("/macros", ownerAdmin, macrosHandler.Create)
	accounts.Get("/macros/:id", agentPlus, macrosHandler.Get)
	accounts.Patch("/macros/:id", ownerAdmin, macrosHandler.Update)
	accounts.Delete("/macros/:id", ownerAdmin, macrosHandler.Delete)
	accounts.Post("/conversations/:convId/apply_macro/:macroId", agentPlus, macrosHandler.Apply)

	accounts.Get("/slas", ownerAdmin, slaHandler.List)
	accounts.Post("/slas", ownerAdmin, slaHandler.Create)
	accounts.Get("/slas/:id", ownerAdmin, slaHandler.Get)
	accounts.Patch("/slas/:id", ownerAdmin, slaHandler.Update)
	accounts.Delete("/slas/:id", ownerAdmin, slaHandler.Delete)
	accounts.Get("/reports/sla", ownerAdmin, slaHandler.Report)
	accounts.Get("/reports/overview", ownerAdmin, reportsHandler.Overview)
	accounts.Get("/reports/conversations", ownerAdmin, reportsHandler.Conversations)
	accounts.Get("/reports/csat", ownerAdmin, reportsHandler.CSAT)
	accounts.Get("/reports/:entity", ownerAdmin, reportsHandler.Entity)

	accounts.Get("/webhooks", ownerAdmin, webhooksHandler.List)
	accounts.Post("/webhooks", ownerAdmin, webhooksHandler.Create)
	accounts.Patch("/webhooks/:id", ownerAdmin, webhooksHandler.Update)
	accounts.Delete("/webhooks/:id", ownerAdmin, webhooksHandler.Delete)

	accounts.Get("/audit_logs", ownerAdmin, auditLogsHandler.List)

	accounts.Get("/notifications", agentPlus, notificationsHandler.List)
	accounts.Post("/notifications/mark_all_read", agentPlus, notificationsHandler.MarkAllRead)
	accounts.Post("/notifications/:id/read", agentPlus, notificationsHandler.MarkRead)

	api.Get("/users/:id/notification_preferences", jwtAuth, notificationsHandler.GetPreferences)
	api.Put("/users/:id/notification_preferences", jwtAuth, notificationsHandler.SetPreferences)

	public := s.App.Group("/public/api/v1")
	publicInbox := public.Group("/inboxes/:identifier", middleware.ApiToken(channelApiRepo), middleware.HmacOptional(cipher))

	publicInbox.Post("/contacts", contactHandler.CreateContact)
	publicInbox.Get("/contacts/:sourceId", contactHandler.GetContact)
	publicInbox.Put("/contacts/:sourceId", contactHandler.UpdateContact)
	publicInbox.Get("/contacts/:sourceId/conversations", conversationHandler.ListByContact)
	publicInbox.Post("/contacts/:sourceId/conversations", conversationHandler.Create)
	publicInbox.Get("/contacts/:sourceId/conversations/:id", conversationHandler.ShowPublic)
	publicInbox.Post("/contacts/:sourceId/conversations/:id/toggle_status", conversationHandler.TogglePublicStatus)
	publicInbox.Post("/contacts/:sourceId/conversations/:id/toggle_typing", conversationHandler.ToggleTyping)
	publicInbox.Post("/contacts/:sourceId/conversations/:id/update_last_seen", conversationHandler.UpdateLastSeenPublic)
	publicInbox.Get("/contacts/:sourceId/conversations/:conversationId/messages", messageHandler.ListPublic)
	publicInbox.Post("/contacts/:sourceId/conversations/:conversationId/messages", messageHandler.Create)
	publicInbox.Put("/contacts/:sourceId/conversations/:convId/messages/:id", messageHandler.UpdatePublic)

	channelInstagramRepo := repo.NewChannelInstagramRepo(db.Pool)
	channelFacebookRepo := repo.NewChannelFacebookRepo(db.Pool)
	channelTelegramRepo := repo.NewChannelTelegramRepo(db.Pool)
	channelLineRepo := repo.NewChannelLineRepo(db.Pool)
	channelTiktokRepo := repo.NewChannelTiktokRepo(db.Pool)
	channelTwitterRepo := repo.NewChannelTwitterRepo(db.Pool)
	channelTwilioRepo := repo.NewChannelTwilioRepo(db.Pool)
	channelSMSRepo := repo.NewChannelSMSRepo(db.Pool)

	channelRegistry := appchannel.NewRegistry()
	channelRegistry.Register(appchannel.KindApi, apichan.NewChannel())
	dedupLock := appchannel.NewDedupLock(redisClient)
	defaultHTTPClient := &http.Client{}

	waReauthTracker := reauth.NewTracker(redisClient)
	waSvc := whatsappchan.NewService(
		channelWhatsAppRepo, inboxRepo, messageRepo, conversationRepo,
		contactSvc, messageSvc, realtimeSvc, cipher, dedupLock, waReauthTracker,
		asynqClient, defaultHTTPClient,
	)
	waChannel := whatsappchan.NewWhatsApp(channelWhatsAppRepo, inboxRepo, cipher, defaultHTTPClient)
	channelRegistry.Register(appchannel.KindWhatsapp, waChannel)

	whatsAppInboxHandler := handler.NewWhatsAppInboxHandler(inboxRepo, channelWhatsAppRepo, cipher, waSvc)
	whatsAppWebhookHandler := handler.NewWhatsAppWebhookHandler(waSvc, inboxRepo, channelWhatsAppRepo)
	accounts.Post("/inboxes/whatsapp", ownerAdmin, whatsAppInboxHandler.Create)
	accounts.Get("/inboxes/:id/whatsapp", agentPlus, whatsAppInboxHandler.GetByID)
	accounts.Post("/inboxes/:id/whatsapp/sync_templates", ownerAdmin, whatsAppInboxHandler.SyncTemplates)
	s.App.Get("/webhooks/whatsapp/:identifier", whatsAppWebhookHandler.HandleHandshake)
	s.App.Post("/webhooks/whatsapp/:identifier", whatsAppWebhookHandler.HandleDelivery)

	igChannel := igchan.NewChannel(
		channelInstagramRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, redisClient, asynqClient,
		cfg.MetaAppSecret, cfg.InstagramVerifyToken,
	)
	channelRegistry.Register(appchannel.KindInstagram, igChannel)

	igWebhookHandler := handler.NewInstagramWebhookHandler(
		channelInstagramRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, dedupLock, asynqClient,
		cfg.MetaAppSecret, cfg.InstagramVerifyToken,
	)

	fbChannel := fbchan.NewChannel(
		channelFacebookRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, redisClient, asynqClient,
		cfg.MetaAppSecret, cfg.FacebookVerifyToken,
	)
	channelRegistry.Register(appchannel.KindFacebookPage, fbChannel)

	fbWebhookHandler := handler.NewFacebookWebhookHandler(
		channelFacebookRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, dedupLock, asynqClient,
		cfg.MetaAppSecret, cfg.FacebookVerifyToken,
	)

	tgChannel := tgchan.NewChannel(
		channelTelegramRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, redisClient, asynqClient,
	)
	channelRegistry.Register(appchannel.KindTelegram, tgChannel)

	tgAPI := tgchan.NewAPIClient()
	tgWebhookHandler := handler.NewTelegramWebhookHandler(
		channelTelegramRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, dedupLock, asynqClient, tgAPI,
	)

	tgMediaResolver := tgchan.NewMediaResolver(
		tgAPI, minioClient.Client(), minioClient.Bucket(),
		channelTelegramRepo, attachmentRepo, messageRepo, inboxRepo, cipher,
	)
	uploadHandler.SetMediaResolver(tgMediaResolver.ResolveMedia)

	lineChannel := linechan.NewChannel(
		channelLineRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, redisClient, asynqClient,
	)
	channelRegistry.Register(appchannel.KindLine, lineChannel)

	lineAPI := linechan.NewAPIClient()
	lineWebhookHandler := handler.NewLineWebhookHandler(
		channelLineRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, dedupLock, asynqClient, lineAPI,
	)

	tiktokReauthTracker := reauth.NewTracker(redisClient)
	tiktokRedirectURL := cfg.APIURL + "/api/v1/accounts/tiktok/oauth/callback"
	tiktokOAuth := tiktokchan.NewOAuthClient(cfg.TiktokClientKey, cfg.TiktokClientSecret, tiktokRedirectURL)
	tiktokTokens := tiktokchan.NewTokenService(tiktokOAuth, channelTiktokRepo, cipher, tiktokReauthTracker)
	if cfg.FeatureChannelTiktok {
		tiktokChannel := tiktokchan.NewChannel(
			channelTiktokRepo, inboxRepo, contactRepo, contactInboxRepo,
			conversationRepo, messageRepo, cipher, redisClient, asynqClient,
			tiktokTokens, cfg.TiktokClientSecret,
		)
		channelRegistry.Register(appchannel.KindTiktok, tiktokChannel)
	}
	tiktokHandler := handler.NewTiktokHandler(
		channelTiktokRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, dedupLock, asynqClient,
		tiktokOAuth, cfg.TiktokClientSecret, redisClient, cfg.FeatureChannelTiktok,
	)

	twitterCallbackURL := cfg.APIURL + "/api/v1/accounts/twitter/oauth/callback"
	twitterOAuth := twitterchan.NewOAuthClient(cfg.TwitterConsumerKey, cfg.TwitterConsumerSecret, twitterCallbackURL)
	if cfg.FeatureChannelTwitter {
		twitterChannel := twitterchan.NewChannel(
			channelTwitterRepo, inboxRepo, contactRepo, contactInboxRepo,
			conversationRepo, messageRepo, cipher, redisClient,
			cfg.TwitterConsumerKey, cfg.TwitterConsumerSecret,
		)
		channelRegistry.Register(appchannel.KindTwitter, twitterChannel)
	}
	twitterHandler := handler.NewTwitterHandler(
		channelTwitterRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, dedupLock,
		twitterOAuth, cfg.TwitterConsumerSecret, redisClient, cfg.FeatureChannelTwitter,
	)

	twilioHTTP := &http.Client{}
	twilioClient := twiliochan.NewClient(twilioHTTP)
	twilioChannel := twiliochan.NewChannel(
		channelTwilioRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, redisClient, twilioClient,
	)
	channelRegistry.Register(appchannel.KindTwilio, twilioChannel)
	twilioWebhookHandler := handler.NewTwilioWebhookHandler(
		channelTwilioRepo, inboxRepo, contactRepo, contactInboxRepo,
		conversationRepo, messageRepo, cipher, dedupLock, twilioClient, twilioChannel, cfg,
	)
	s.twilioTemplatesJob = twiliochan.NewTemplatesJob(channelTwilioRepo, twilioChannel, 24*time.Hour, 24*time.Hour)

	smsRegistry := smschan.NewRegistry()
	smsRegistry.Register("twilio", smstwilio.New(defaultHTTPClient, cipher))
	smsRegistry.Register("bandwidth", bandwidth.New(defaultHTTPClient, cipher))
	smsRegistry.Register("zenvia", zenvia.New(defaultHTTPClient, cipher))

	smsMediaHandler := smschan.NewMediaHandler(minioClient, attachmentRepo)
	smsIngestSvc := smschan.NewIngestService(
		channelSMSRepo, inboxRepo, contactSvc, contactRepo,
		conversationRepo, messageRepo, messageSvc, dedupLock, smsMediaHandler,
	)
	smsDedupLock := appchannel.NewDedupLock(redisClient)

	smsChannel := smschan.NewChannel(
		channelSMSRepo, inboxRepo, messageRepo, smsRegistry,
		cipher, smsDedupLock, reauth.NewTracker(redisClient), asynqClient,
		smsMediaHandler, realtimeSvc, defaultHTTPClient,
	)
	channelRegistry.Register(appchannel.KindSms, smsChannel)

	smsWebhookHandler := handler.NewSMSWebhookHandler(
		channelSMSRepo, messageRepo, smsRegistry, smsIngestSvc, messageSvc,
	)

	smsBaseURL := cfg.APIURL
	smsInboxHandler := handler.NewSMSInboxHandler(
		channelSMSRepo, inboxRepo, smsRegistry, cipher, smsBaseURL,
	)

	s.App.Post("/webhooks/sms/:provider/:identifier", smsWebhookHandler.Receive)
	s.App.Post("/webhooks/sms/:provider/:identifier/status", smsWebhookHandler.Status)

	accounts.Post("/inboxes/sms", ownerAdmin, smsInboxHandler.Provision)

	s.App.Get("/webhooks/instagram/:identifier", igWebhookHandler.Verify)
	s.App.Post("/webhooks/instagram/:identifier", igWebhookHandler.Receive)
	s.App.Get("/webhooks/facebook/:identifier", fbWebhookHandler.Verify)
	s.App.Post("/webhooks/facebook/:identifier", fbWebhookHandler.Receive)
	s.App.Post("/webhooks/telegram/:identifier", tgWebhookHandler.Receive)
	s.App.Post("/webhooks/line/:line_channel_id", lineWebhookHandler.Receive)
	s.App.Post("/webhooks/tiktok/:business_id", tiktokHandler.Receive)
	s.App.Post("/webhooks/twilio/:identifier", twilioWebhookHandler.Receive)
	s.App.Post("/webhooks/twilio/:identifier/status", twilioWebhookHandler.Status)
	s.App.Get("/webhooks/twitter/:profile_id", twitterHandler.CRC)
	s.App.Post("/webhooks/twitter/:profile_id", twitterHandler.Receive)

	accounts.Post("/inboxes/instagram", ownerAdmin, igWebhookHandler.Provision)
	accounts.Post("/inboxes/facebook_page", ownerAdmin, fbWebhookHandler.Provision)

	channelWebWidgetRepo := repo.NewChannelWebWidgetRepo(db.Pool)
	jwtSvc := webwidget.NewVisitorJWTService(cfg.WidgetJWTSecret, cfg.WidgetSessionTTL)
	sessionSvc := webwidget.NewSessionService(channelWebWidgetRepo, contactRepo, contactInboxRepo, conversationRepo, jwtSvc)
	identifySvc := webwidget.NewIdentifyService(channelWebWidgetRepo, contactRepo, contactInboxRepo, conversationRepo, cipher, jwtSvc)
	sseHandler := webwidget.NewSSEHandler(redisClient, conversationRepo, channelWebWidgetRepo)
	widgetPublicHandler := handler.NewWidgetPublicHandler(sessionSvc, identifySvc, channelWebWidgetRepo, conversationRepo, messageRepo, jwtSvc, sseHandler, cfg)

	webWidgetChannel := webwidget.NewChannel(channelWebWidgetRepo, conversationRepo, messageRepo, redisClient)
	channelRegistry.Register(appchannel.KindWebWidget, webWidgetChannel)

	widgetRateLimiter := middleware.NewWidgetRateLimiter(redisClient)
	widgetCORS := middleware.WidgetCORS()

	s.App.Get("/widget/:websiteToken", widgetCORS, widgetPublicHandler.EmbedScript)
	s.App.Get("/widget/:websiteToken/ws", widgetCORS, widgetPublicHandler.SSE)

	widgetAPI := s.App.Group("/api/v1/widget", widgetCORS)
	widgetAPI.Post("/sessions", widgetRateLimiter.LimitByIP(10, time.Minute), widgetPublicHandler.CreateSession)
	widgetAPI.Post("/messages", widgetRateLimiter.LimitByIP(60, time.Minute), widgetPublicHandler.SendMessage)
	widgetAPI.Post("/identify", widgetRateLimiter.LimitByIP(20, time.Minute), widgetPublicHandler.Identify)
	widgetAPI.Post("/attachments", widgetRateLimiter.LimitByIP(30, time.Minute), widgetPublicHandler.GetAttachmentPresigned)
	widgetAPI.Get("/messages", widgetRateLimiter.LimitByIP(60, time.Minute), widgetPublicHandler.PollMessages)

	widgetInboxHandler := handler.NewWebWidgetInboxHandler(channelWebWidgetRepo, inboxRepo, cipher, cfg)
	accounts.Post("/inboxes/web_widget", ownerAdmin, widgetInboxHandler.Create)
	accounts.Get("/inboxes/web_widget/:id", agentPlus, widgetInboxHandler.GetByInboxID)
	accounts.Post("/inboxes/:id/rotate_hmac", ownerAdmin, widgetInboxHandler.RotateHmac)
	accounts.Post("/inboxes/telegram", ownerAdmin, tgWebhookHandler.Provision)
	accounts.Delete("/inboxes/:id/telegram", ownerAdmin, tgWebhookHandler.Delete)
	accounts.Post("/inboxes/line", ownerAdmin, lineWebhookHandler.Provision)
	accounts.Get("/inboxes/:id/line", agentPlus, lineWebhookHandler.GetByInboxID)
	accounts.Put("/inboxes/:id/line", ownerAdmin, lineWebhookHandler.Update)
	accounts.Delete("/inboxes/:id/line", ownerAdmin, lineWebhookHandler.Delete)
	api.Get("/accounts/tiktok/oauth/callback", tiktokHandler.Callback)
	api.Get("/accounts/twitter/oauth/callback", twitterHandler.Callback)
	accounts.Post("/inboxes/tiktok/authorize", ownerAdmin, tiktokHandler.Authorize)
	accounts.Get("/inboxes/:id/tiktok", agentPlus, tiktokHandler.GetByInboxID)
	accounts.Delete("/inboxes/:id/tiktok", ownerAdmin, tiktokHandler.Delete)
	accounts.Post("/inboxes/twilio", ownerAdmin, twilioWebhookHandler.Provision)
	accounts.Get("/inboxes/:id/twilio", agentPlus, twilioWebhookHandler.GetByInboxID)
	accounts.Put("/inboxes/:id/twilio", ownerAdmin, twilioWebhookHandler.Update)
	accounts.Post("/inboxes/:id/twilio_templates", ownerAdmin, twilioWebhookHandler.SyncTemplates)
	accounts.Delete("/inboxes/:id/twilio", ownerAdmin, twilioWebhookHandler.Delete)
	accounts.Post("/inboxes/twitter/authorize", ownerAdmin, twitterHandler.Authorize)
	accounts.Get("/inboxes/:id/twitter", agentPlus, twitterHandler.GetByInboxID)
	accounts.Put("/inboxes/:id/twitter", ownerAdmin, twitterHandler.Update)
	accounts.Delete("/inboxes/:id/twitter", ownerAdmin, twitterHandler.Delete)

	s.App.Use(NotFoundHandler)

	_ = channelRegistry

	// Backfill user access tokens for existing users (one-time migration)
	go func() {
		if err := authSvc.BackfillUserAccessTokens(context.Background()); err != nil {
			logger.Error().Str("component", "server").Err(err).Msg("failed to backfill user access tokens")
		}
	}()

	logger.Info().Str("component", "server").Msg("Routes registered")
	return asynqClient, nil
}
