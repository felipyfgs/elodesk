package handler

import (
	"context"
	"errors"

	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"backend/internal/logger"
	"backend/internal/realtime"
	"backend/internal/repo"
	"backend/internal/service"
)

type RealtimeHandler struct {
	authSvc *service.AuthService
	hub     *realtime.Hub
	checker realtime.MembershipChecker
}

func NewRealtimeHandler(
	authSvc *service.AuthService,
	hub *realtime.Hub,
	accountRepo *repo.AccountRepo,
	inboxRepo *repo.InboxRepo,
	conversationRepo *repo.ConversationRepo,
) *RealtimeHandler {
	return &RealtimeHandler{
		authSvc: authSvc,
		hub:     hub,
		checker: &dbMembershipChecker{
			accountRepo:      accountRepo,
			inboxRepo:        inboxRepo,
			conversationRepo: conversationRepo,
		},
	}
}

func (h *RealtimeHandler) RegisterRoutes(app *fiber.App) {
	app.Use("/realtime", func(c *fiber.Ctx) error {
		if !websocket.IsWebSocketUpgrade(c) {
			return c.Status(fiber.StatusUpgradeRequired).JSON(fiber.Map{
				"success": false,
				"error":   "Upgrade Required",
				"message": "websocket upgrade required",
			})
		}
		return c.Next()
	})

	app.Get("/realtime", websocket.New(h.HandleWebSocket))
}

func (h *RealtimeHandler) HandleWebSocket(c *websocket.Conn) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		tokenStr = c.Headers("Sec-WebSocket-Protocol")
		if tokenStr == "" {
			_ = c.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "missing token"))
			return
		}
	}

	user, err := h.authSvc.ValidateAccessToken(tokenStr)
	if err != nil {
		logger.Warn().Str("component", "realtime").Err(err).Msg("websocket auth failed")
		_ = c.WriteMessage(websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.ClosePolicyViolation, "invalid token"))
		return
	}

	client := realtime.NewClient(h.hub, c, user.ID, h.checker)
	h.hub.Register(client)

	go client.WritePump()
	client.ReadPump()
}

type dbMembershipChecker struct {
	accountRepo      *repo.AccountRepo
	inboxRepo        *repo.InboxRepo
	conversationRepo *repo.ConversationRepo
}

func (d *dbMembershipChecker) UserInAccount(ctx context.Context, userID, accountID int64) bool {
	_, err := d.accountRepo.FindAccountUser(ctx, accountID, userID)
	return err == nil
}

func (d *dbMembershipChecker) InboxAccount(ctx context.Context, inboxID int64) (int64, bool) {
	id, ok := inboxAccountByID(ctx, d.inboxRepo, inboxID)
	if !ok {
		return 0, false
	}
	return id, true
}

func (d *dbMembershipChecker) ConversationAccount(ctx context.Context, conversationID int64) (int64, bool) {
	id, ok := conversationAccountByID(ctx, d.conversationRepo, conversationID)
	if !ok {
		return 0, false
	}
	return id, true
}

func inboxAccountByID(ctx context.Context, r *repo.InboxRepo, id int64) (int64, bool) {
	accountID, err := r.AccountIDByID(ctx, id)
	if err != nil {
		if !errors.Is(err, repo.ErrInboxNotFound) {
			logger.Warn().Str("component", "realtime").Err(err).Msg("inbox membership lookup error")
		}
		return 0, false
	}
	return accountID, true
}

func conversationAccountByID(ctx context.Context, r *repo.ConversationRepo, id int64) (int64, bool) {
	accountID, err := r.AccountIDByID(ctx, id)
	if err != nil {
		if !errors.Is(err, repo.ErrConversationNotFound) {
			logger.Warn().Str("component", "realtime").Err(err).Msg("conversation membership lookup error")
		}
		return 0, false
	}
	return accountID, true
}
