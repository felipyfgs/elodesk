package handler

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
	"backend/internal/service"
)

func TestListTemplates_ReturnsFour(t *testing.T) {
	app := fiber.New()
	h := &PipelineHandler{}
	app.Get("/templates", h.ListTemplates)

	req := httptest.NewRequest("GET", "/templates", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != fiber.StatusOK {
		t.Fatalf("status: got %d, want 200", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	var wrap struct {
		Success bool                   `json:"success"`
		Data    []dto.PipelineTemplate `json:"data"`
	}
	if err := json.Unmarshal(body, &wrap); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !wrap.Success {
		t.Error("success should be true")
	}
	if len(wrap.Data) != 4 {
		t.Errorf("got %d templates, want 4", len(wrap.Data))
	}
	// Quick sanity: make sure the handler exposes the same templates as the service catalog.
	if len(wrap.Data) != len(service.ListTemplates()) {
		t.Errorf("handler templates length differs from service catalog")
	}
}
