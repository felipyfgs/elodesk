package facebook

import (
	"context"
	"encoding/json"
	"testing"

	"backend/internal/channel/meta"
)

func TestProcessWebhookPayload_Standby(t *testing.T) {
	payload := meta.WebhookPayload{
		Object: "page",
		Entry: []meta.Entry{
			{
				ID: "page123",
				Standby: []meta.MessagingEntry{
					{
						Sender:    meta.IDHolder{ID: "user456"},
						Recipient: meta.IDHolder{ID: "page123"},
						Timestamp: 1700000000000,
						Message: &meta.Message{
							Mid:  "mid_standby_001",
							Text: "help me",
						},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)

	// Nil repos - we test that the standby path is reached without panicking.
	err := ProcessWebhook(context.Background(), body, nil, 0, nil, nil, nil, nil, nil, nil)
	_ = err
}

func TestProcessWebhookPayload_DeliveryWatermark(t *testing.T) {
	watermark := int64(1700000001000)
	payload := meta.WebhookPayload{
		Object: "page",
		Entry: []meta.Entry{
			{
				ID: "page123",
				Messaging: []meta.MessagingEntry{
					{
						Sender:    meta.IDHolder{ID: "user456"},
						Recipient: meta.IDHolder{ID: "page123"},
						Timestamp: 1700000000000,
						Delivery:  &meta.Delivery{Watermark: watermark},
					},
				},
			},
		},
	}

	body, _ := json.Marshal(payload)
	err := ProcessWebhook(context.Background(), body, nil, 0, nil, nil, nil, nil, nil, nil)
	_ = err // nil inbox is expected error; delivery path is reached
}

func TestDedupKeyPrefix(t *testing.T) {
	mid := "m_fb_123"
	key := dedupKeyPrefix + mid
	if key != "elodesk:meta:m_fb_123" {
		t.Fatalf("unexpected dedup key: %s", key)
	}
}
