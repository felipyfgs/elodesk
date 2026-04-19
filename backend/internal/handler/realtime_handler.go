package handler

import (
	"github.com/gofiber/contrib/websocket"
	"github.com/gofiber/fiber/v2"

	"backend/internal/logger"
	"backend/internal/realtime"
	"backend/internal/service"
)

type RealtimeHandler struct {
	authSvc *service.AuthService
	hub     *realtime.Hub
}

func NewRealtimeHandler(authSvc *service.AuthService, hub *realtime.Hub) *RealtimeHandler {
	return &RealtimeHandler{authSvc: authSvc, hub: hub}
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

	client := realtime.NewClient(h.hub, c, user.ID)
	h.hub.Register(client)

	go client.WritePump()
	client.ReadPump()
}
