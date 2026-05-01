package handler

import (
	"errors"
	"testing"
)

func TestParseContactCSV_ValidBasic(t *testing.T) {
	csv := "name,email,phone\nAlice,alice@example.com,+551199999\nBob,bob@example.com,"
	parsed, err := ParseContactCSV(csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.Contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(parsed.Contacts))
	}
	if parsed.Contacts[0].Name != "Alice" {
		t.Errorf("contact[0].Name = %q, want %q", parsed.Contacts[0].Name, "Alice")
	}
	if parsed.Contacts[0].Email != "alice@example.com" {
		t.Errorf("contact[0].Email = %q, want %q", parsed.Contacts[0].Email, "alice@example.com")
	}
	if parsed.Contacts[0].Phone != "+551199999" {
		t.Errorf("contact[0].Phone = %q, want %q", parsed.Contacts[0].Phone, "+551199999")
	}
	if parsed.Contacts[1].Name != "Bob" {
		t.Errorf("contact[1].Name = %q, want %q", parsed.Contacts[1].Name, "Bob")
	}
	if len(parsed.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d", len(parsed.Errors))
	}
}

func TestParseContactCSV_MissingRequiredColumn(t *testing.T) {
	csv := "city,country\nSP,Brazil"
	_, err := ParseContactCSV(csv)
	if !errors.Is(err, ErrMissingRequiredColumn) {
		t.Errorf("expected ErrMissingRequiredColumn, got %v", err)
	}
}

func TestParseContactCSV_EmptyRows(t *testing.T) {
	csv := "name,email\n,,\n,Alice,alice@example.com\nAlice,alice@example.com,+55119999"
	parsed, err := ParseContactCSV(csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.Contacts) != 2 {
		t.Errorf("expected 2 valid contacts, got %d", len(parsed.Contacts))
	}
	if len(parsed.Errors) != 1 {
		t.Errorf("expected 1 error (empty row), got %d", len(parsed.Errors))
	}
	if parsed.Errors[0].Reason != "name and email are empty" {
		t.Errorf("error reason = %q, want %q", parsed.Errors[0].Reason, "name and email are empty")
	}
}

func TestParseContactCSV_PortugueseHeaders(t *testing.T) {
	csv := "nome,email,telefone\nJoão,joao@test.com,+55119999"
	parsed, err := ParseContactCSV(csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.Contacts) != 1 {
		t.Fatalf("expected 1 contact, got %d", len(parsed.Contacts))
	}
	if parsed.Contacts[0].Name != "João" {
		t.Errorf("name = %q, want %q", parsed.Contacts[0].Name, "João")
	}
}

func TestParseContactCSV_EmailOnly(t *testing.T) {
	csv := "email\ntest@example.com\nanother@test.com"
	parsed, err := ParseContactCSV(csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.Contacts) != 2 {
		t.Fatalf("expected 2 contacts, got %d", len(parsed.Contacts))
	}
}

func TestParseContactCSV_EmptyCSV(t *testing.T) {
	csv := ""
	_, err := ParseContactCSV(csv)
	if err == nil {
		t.Error("expected error for empty CSV")
	}
}

func TestParseContactCSV_MalformedRows(t *testing.T) {
	csv := "name,email\n\"unclosed quote,test@test.com\nAlice,alice@test.com"
	parsed, err := ParseContactCSV(csv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(parsed.Errors) == 0 {
		t.Error("expected at least 1 error for malformed row")
	}
}
