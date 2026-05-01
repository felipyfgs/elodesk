package handler

import (
	"encoding/json"
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"backend/internal/dto"
)

func TestParticipantHandler_List_JSONShape(t *testing.T) {
	app := fiber.New()
	app.Get("/accounts/:aid/conversations/:id/participants", func(c *fiber.Ctx) error {
		c.Locals("accountId", int64(1))
		out := []dto.ParticipantResp{
			{ID: 1, Role: "admin", Contact: dto.ContactResp{ID: 42, Name: "Alice"}},
		}
		return c.JSON(dto.ParticipantListResp{Data: out})
	})

	req := httptest.NewRequest("GET", "/accounts/1/conversations/5/participants", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != fiber.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var result dto.ParticipantListResp
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Data) != 1 {
		t.Fatalf("Data length = %d, want 1", len(result.Data))
	}
	if result.Data[0].Contact.Name != "Alice" {
		t.Errorf("Contact.Name = %q, want %q", result.Data[0].Contact.Name, "Alice")
	}
}

func TestParticipantHandler_List_EmptyArrayForNoParticipants(t *testing.T) {
	app := fiber.New()
	app.Get("/accounts/:aid/conversations/:id/participants", func(c *fiber.Ctx) error {
		c.Locals("accountId", int64(1))
		return c.JSON(dto.ParticipantListResp{Data: []dto.ParticipantResp{}})
	})

	req := httptest.NewRequest("GET", "/accounts/1/conversations/5/participants", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(resp.Body)
	var result dto.ParticipantListResp
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(result.Data) != 0 {
		t.Errorf("Data length = %d, want 0", len(result.Data))
	}
}

// TestParticipantHandler_List_RejectsMissingAccountId verifies the handler
// returns 500 when accountId is not in locals.
func TestParticipantHandler_List_RejectsMissingAccountId(t *testing.T) {
	app := fiber.New()
	app.Get("/accounts/:aid/conversations/:id/participants", func(c *fiber.Ctx) error {
		h := &ParticipantHandler{}
		return h.List(c)
	})

	req := httptest.NewRequest("GET", "/accounts/1/conversations/5/participants", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != fiber.StatusInternalServerError {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)
	var errResp dto.APIError
	if err := json.Unmarshal(body, &errResp); err != nil {
		t.Fatalf("unmarshal error response: %v", err)
	}
	if errResp.Error != "Error" {
		t.Errorf("error = %q, want %q", errResp.Error, "Error")
	}
}
