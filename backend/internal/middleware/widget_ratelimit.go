package middleware

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
)

type WidgetRateLimiter struct {
	redisClient *redis.Client
}

func NewWidgetRateLimiter(redisClient *redis.Client) *WidgetRateLimiter {
	return &WidgetRateLimiter{redisClient: redisClient}
}

func (r *WidgetRateLimiter) LimitByIP(max int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		key := fmt.Sprintf("ratelimit:widget:ip:%s:%s", c.Path(), c.IP())
		return r.check(c, key, max, window)
	}
}

func (r *WidgetRateLimiter) LimitByToken(max int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Params("websiteToken")
		if token == "" {
			token = c.Query("websiteToken")
		}
		key := fmt.Sprintf("ratelimit:widget:token:%s:%s", c.Path(), token)
		return r.check(c, key, max, window)
	}
}

func (r *WidgetRateLimiter) LimitByIPAndToken(max int, window time.Duration) fiber.Handler {
	return func(c *fiber.Ctx) error {
		token := c.Params("websiteToken")
		if token == "" {
			token = c.Query("websiteToken")
		}
		key := fmt.Sprintf("ratelimit:widget:ip:%s:%s:%s", c.Path(), c.IP(), token)
		return r.check(c, key, max, window)
	}
}

func (r *WidgetRateLimiter) check(c *fiber.Ctx, key string, max int, window time.Duration) error {
	ctx := c.Context()
	pipe := r.redisClient.TxPipeline()
	countCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "rate limit check failed"})
	}

	count := countCmd.Val()
	if count > int64(max) {
		ttl, err := r.redisClient.TTL(ctx, key).Result()
		if err != nil {
			ttl = window
		}
		retryAfter := int(ttl.Seconds())
		if retryAfter < 1 {
			retryAfter = 1
		}
		c.Set("Retry-After", strconv.Itoa(retryAfter))
		return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
			"error":      "rate limit exceeded",
			"retryAfter": retryAfter,
		})
	}

	return c.Next()
}
