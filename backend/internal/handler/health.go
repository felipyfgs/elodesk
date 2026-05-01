package handler

import (
	"backend/internal/database"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type HealthHandler struct {
	db    *database.DB
	redis *redis.Client
}

func NewHealthHandler(db *database.DB, redisClient *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redisClient}
}

type healthResponse struct {
	Status string `json:"status"`
	DB     string `json:"db"`
	Redis  string `json:"redis"`
}

// @Summary Health check
// @Description Returns the health status of the service and its dependencies
// @Tags health
// @Produce json
// @Success 200 {object} healthResponse
// @Router /health [get]
func (h *HealthHandler) Check(c *fiber.Ctx) error {
	resp := healthResponse{}

	ctx := c.Context()

	if err := h.db.Health(ctx); err != nil {
		resp.DB = "error"
	} else {
		resp.DB = "ok"
	}

	if err := h.redis.Ping(ctx).Err(); err != nil {
		resp.Redis = "error"
	} else {
		resp.Redis = "ok"
	}

	if resp.DB == "ok" && resp.Redis == "ok" {
		resp.Status = "ok"
		return c.JSON(resp)
	}
	// Degraded: return 503 so liveness/readiness probes mark the pod unhealthy.
	resp.Status = "degraded"
	return c.Status(fiber.StatusServiceUnavailable).JSON(resp)
}
