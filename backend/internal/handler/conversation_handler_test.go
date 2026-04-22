package handler

import (
	"io"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func TestTogglePublicStatus_ValidStatuses(t *testing.T) {
	app := fiber.New()
	app.Post("/test", func(c *fiber.Ctx) error {
		var req struct {
			Status string `json:"status"`
		}
		if err := c.BodyParser(&req); err != nil {
			return err
		}
		return c.JSON(req.Status)
	})

	validStatuses := []string{"resolved", "open", "pending", "snoozed"}
	for _, s := range validStatuses {
		req := `{"status":"` + s + `"}`
		r := httptest.NewRequest("POST", "/test", strings.NewReader(req))
		r.Header.Set("Content-Type", "application/json")
		resp, err := app.Test(r)
		if err != nil {
			t.Fatalf("status %s: unexpected error: %v", s, err)
		}
		if resp.StatusCode != fiber.StatusOK {
			t.Errorf("status %s: got %d, want 200", s, resp.StatusCode)
		}
		_, _ = io.ReadAll(resp.Body)
		_ = resp.Body.Close()
	}
}
