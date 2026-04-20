package audit

import (
	"context"
	"encoding/json"
	"net"

	"github.com/gofiber/fiber/v2"

	"backend/internal/logger"
	"backend/internal/repo"
)

type Logger struct {
	repo *repo.AuditLogRepo
}

func NewLogger(r *repo.AuditLogRepo) *Logger {
	return &Logger{repo: r}
}

func (l *Logger) Log(ctx context.Context, accountID int64, userID *int64, action, entityType string, entityID *int64, metadata any, ip, userAgent string) {
	if l == nil || l.repo == nil {
		return
	}
	metaStr := "{}"
	if metadata != nil {
		if b, err := json.Marshal(metadata); err == nil {
			metaStr = string(b)
		}
	}
	var parsedIP net.IP
	if ip != "" {
		parsedIP = net.ParseIP(ip)
	}
	if err := l.repo.Create(ctx, accountID, userID, action, entityType, entityID, metaStr, parsedIP, userAgent); err != nil {
		logger.Error().Str("component", "audit").Err(err).Str("action", action).Msg("failed to persist audit log")
	}
}

// LogFromCtx extracts accountId/user from fiber ctx and logs an event.
func (l *Logger) LogFromCtx(c *fiber.Ctx, action, entityType string, entityID *int64, metadata any) {
	if l == nil {
		return
	}
	accountID, _ := c.Locals("accountId").(int64)
	var userIDPtr *int64
	if u, ok := c.Locals("user").(*repo.AuthUser); ok && u != nil {
		id := u.ID
		userIDPtr = &id
	}
	l.Log(c.Context(), accountID, userIDPtr, action, entityType, entityID, metadata, c.IP(), c.Get("User-Agent"))
}
