package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"backend/internal/model"
)

func TestMarshalMessage_NoAttachmentsFallsBackToPlainMarshal(t *testing.T) {
	svc := &OutboundWebhookService{}
	content := "ola"
	msg := &model.Message{ID: 1, Content: &content}

	data, err := svc.marshalMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	var roundtrip model.Message
	if err := json.Unmarshal(data, &roundtrip); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if roundtrip.ID != 1 {
		t.Errorf("id: got %d", roundtrip.ID)
	}
}

func TestMarshalMessage_EnrichesAttachmentsWithDataURL(t *testing.T) {
	calls := 0
	builder := func(accountID, attachmentID int64) string {
		calls++
		return fmt.Sprintf("http://elodesk/api/v1/attachments/%d/file?token=t-%d", attachmentID, accountID)
	}

	svc := (&OutboundWebhookService{}).WithAttachmentURLBuilder(builder)

	key1 := "1/uploads/voice.webm"
	key2 := "1/uploads/photo.jpg"
	msg := &model.Message{
		ID: 7,
		Attachments: []model.Attachment{
			{ID: 11, AccountID: 5, FileType: model.FileTypeAudio, FileKey: &key1},
			{ID: 12, AccountID: 5, FileType: model.FileTypeImage, FileKey: &key2},
		},
	}

	data, err := svc.marshalMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}

	if calls != 2 {
		t.Fatalf("expected 2 builder calls, got %d", calls)
	}

	var got struct {
		Attachments []struct {
			ID      int64  `json:"id"`
			DataURL string `json:"dataUrl"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(got.Attachments) != 2 {
		t.Fatalf("expected 2 attachments, got %d", len(got.Attachments))
	}
	if got.Attachments[0].DataURL != "http://elodesk/api/v1/attachments/11/file?token=t-5" {
		t.Errorf("att[0] dataUrl: got %q", got.Attachments[0].DataURL)
	}
	if got.Attachments[1].DataURL != "http://elodesk/api/v1/attachments/12/file?token=t-5" {
		t.Errorf("att[1] dataUrl: got %q", got.Attachments[1].DataURL)
	}
}

func TestMarshalMessage_NoURLBuilderLeavesDataURLEmpty(t *testing.T) {
	svc := &OutboundWebhookService{}
	key := "1/uploads/voice.webm"
	msg := &model.Message{
		ID:          7,
		Attachments: []model.Attachment{{ID: 11, FileType: model.FileTypeAudio, FileKey: &key}},
	}

	data, err := svc.marshalMessage(context.Background(), msg)
	if err != nil {
		t.Fatalf("err: %v", err)
	}
	var got struct {
		Attachments []struct {
			DataURL string `json:"dataUrl"`
		} `json:"attachments"`
	}
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.Attachments[0].DataURL != "" {
		t.Errorf("expected empty dataUrl, got %q", got.Attachments[0].DataURL)
	}
}
