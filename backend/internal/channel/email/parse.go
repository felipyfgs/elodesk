package email

import (
	"fmt"
	"io"
	"strings"

	"github.com/jhillyerd/enmime"
)

// ParseMIME reads a raw MIME message from r and returns a parsed Envelope.
// On parse errors enmime still returns partial results; we return the partial
// envelope alongside the error so callers can decide whether to persist or drop.
func ParseMIME(r io.Reader) (*Envelope, error) {
	e, err := enmime.ReadEnvelope(r)
	if err != nil {
		// enmime may still populate fields even on error (e.g. bad charset)
		if e != nil {
			return fromEnmime(e), fmt.Errorf("mime parse partial: %w", err)
		}
		return nil, fmt.Errorf("mime parse: %w", err)
	}
	return fromEnmime(e), nil
}

// splitMessageIDs splits a space/comma-separated list of RFC 5322 message-ids
// (angle-bracket-wrapped) and returns the raw ids including brackets.
func splitMessageIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []string
	// Message-IDs are delimited by whitespace; each is "<...>".
	for _, part := range strings.Fields(raw) {
		part = strings.TrimSpace(part)
		if part != "" {
			ids = append(ids, part)
		}
	}
	return ids
}
