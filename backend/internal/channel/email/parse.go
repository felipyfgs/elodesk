package email

import (
	"fmt"
	"io"
	"strings"

	"github.com/jhillyerd/enmime"
)

func ParseMIME(r io.Reader) (*Envelope, error) {
	e, err := enmime.ReadEnvelope(r)
	if err != nil {
		if e != nil {
			return fromEnmime(e), fmt.Errorf("mime parse partial: %w", err)
		}
		return nil, fmt.Errorf("mime parse: %w", err)
	}
	return fromEnmime(e), nil
}

func splitMessageIDs(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var ids []string
	for _, part := range strings.Fields(raw) {
		part = strings.TrimSpace(part)
		if part != "" {
			ids = append(ids, part)
		}
	}
	return ids
}
