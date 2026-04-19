package meta

import (
	"io"
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

func newFiberApp() *fiber.App {
	app := fiber.New()
	return app
}

func TestHandleVerifyChallenge_Valid(t *testing.T) {
	app := newFiberApp()
	const token = "my-verify-token"
	app.Get("/webhook", func(c *fiber.Ctx) error {
		return HandleVerifyChallenge(c, token)
	})

	req := httptest.NewRequest("GET",
		"/webhook?hub.mode=subscribe&hub.verify_token=my-verify-token&hub.challenge=abc123",
		nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("expected 200, got %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if string(body) != "abc123" {
		t.Fatalf("expected 'abc123' challenge, got %q", string(body))
	}
}

func TestHandleVerifyChallenge_WrongToken(t *testing.T) {
	app := newFiberApp()
	app.Get("/webhook", func(c *fiber.Ctx) error {
		return HandleVerifyChallenge(c, "expected-token")
	})

	req := httptest.NewRequest("GET",
		"/webhook?hub.mode=subscribe&hub.verify_token=wrong-token&hub.challenge=xyz",
		nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}

func TestHandleVerifyChallenge_WrongMode(t *testing.T) {
	app := newFiberApp()
	app.Get("/webhook", func(c *fiber.Ctx) error {
		return HandleVerifyChallenge(c, "token")
	})

	req := httptest.NewRequest("GET",
		"/webhook?hub.mode=unsubscribe&hub.verify_token=token&hub.challenge=abc",
		nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("expected 401, got %d", resp.StatusCode)
	}
}
