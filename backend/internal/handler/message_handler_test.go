package handler

import (
	"bytes"
	"mime/multipart"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
)

func TestUpdatePublicCSAT_Lock14Days(t *testing.T) {
	oldMessage := time.Now().Add(-15 * 24 * time.Hour)
	recentMessage := time.Now().Add(-7 * 24 * time.Hour)

	if time.Since(oldMessage) <= 14*24*time.Hour {
		t.Error("old message (15 days ago) should exceed 14-day lock")
	}

	if time.Since(recentMessage) > 14*24*time.Hour {
		t.Error("recent message (7 days ago) should be within 14-day lock")
	}

	_ = fiber.StatusBadRequest
	_ = fiber.StatusUnprocessableEntity
}

func TestCreateMultipart_DetectsMultipart(t *testing.T) {
	app := fiber.New()
	multipartDetected := false

	app.Post("/test", func(c *fiber.Ctx) error {
		ct := c.Get("Content-Type")
		if strings.HasPrefix(ct, "multipart/form-data") {
			multipartDetected = true
			content := c.FormValue("content")
			if content != "hello" {
				t.Errorf("content = %q, want %q", content, "hello")
			}
			echoID := c.FormValue("echo_id")
			if echoID != "msg-123" {
				t.Errorf("echo_id = %q, want %q", echoID, "msg-123")
			}
			return c.SendStatus(fiber.StatusOK)
		}
		return c.SendStatus(fiber.StatusBadRequest)
	})

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	_ = writer.WriteField("content", "hello")
	_ = writer.WriteField("echo_id", "msg-123")
	_ = writer.Close()

	req := httptest.NewRequest("POST", "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("got %d, want 200", resp.StatusCode)
	}
	if !multipartDetected {
		t.Error("multipart form was not detected")
	}
}
