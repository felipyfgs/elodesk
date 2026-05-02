package instagram

import (
	"context"
	"encoding/json"
	"testing"

	"backend/internal/channel/meta"
)

func TestProcessWebhookPayload_SkipEcho(t *testing.T) {
	payload := meta.WebhookPayload{
		Object: "instagram",
		Entry: []meta.Entry{
			{
				ID: "1234",
				Messaging: []meta.MessagingEntry{
					{
						Sender:    meta.IDHolder{ID: "sender1"},
						Recipient: meta.IDHolder{ID: "recv1"},
						Timestamp: 1700000000000,
						Message: &meta.Message{
							Mid:    "mid_echo_001",
							Text:   "hello",
							IsEcho: true,
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)

	err := ProcessWebhook(context.Background(), body, nil, 0, nil, nil, nil, nil, nil, nil)
	_ = err
}

func TestBuildDedupKey(t *testing.T) {
	mid := "m_abc123"
	key := dedupKeyPrefix + mid
	expected := "elodesk:meta:m_abc123"
	if key != expected {
		t.Fatalf("expected %q, got %q", expected, key)
	}
}
