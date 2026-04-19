package telegram

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"backend/internal/model"
)

func Send(ctx context.Context, ch *model.ChannelTelegram, botToken string, to string, content string, mediaURL string, mediaType string, contentAttrsJSON string) (string, error) {
	api := NewAPIClient()

	chatID, err := strconv.ParseInt(to, 10, 64)
	if err != nil {
		return "", fmt.Errorf("telegram send: parse chat_id: %w", err)
	}

	htmlContent := MarkdownToHTML(content)
	parseMode := "HTML"

	replyTo, buttons := parseContentAttrs(contentAttrsJSON)

	if mediaURL != "" {
		switch mediaType {
		case "image":
			req := SendPhotoRequest{
				ChatID:    chatID,
				Photo:     mediaURL,
				Caption:   htmlContent,
				ParseMode: parseMode,
			}
			if replyTo > 0 {
				req.ReplyToMessageID = replyTo
			}
			if buttons != nil {
				req.ReplyMarkup = buttons
			}
			result, err := api.SendPhoto(ctx, botToken, req)
			if err != nil {
				return "", err
			}
			return strconv.FormatInt(result.MessageID, 10), nil
		default:
			req := SendDocumentRequest{
				ChatID:    chatID,
				Document:  mediaURL,
				Caption:   htmlContent,
				ParseMode: parseMode,
			}
			if replyTo > 0 {
				req.ReplyToMessageID = replyTo
			}
			if buttons != nil {
				req.ReplyMarkup = buttons
			}
			result, err := api.SendDocument(ctx, botToken, req)
			if err != nil {
				return "", err
			}
			return strconv.FormatInt(result.MessageID, 10), nil
		}
	}

	req := SendMessageRequest{
		ChatID:    chatID,
		Text:      htmlContent,
		ParseMode: parseMode,
	}
	if replyTo > 0 {
		req.ReplyToMessageID = replyTo
	}
	if buttons != nil {
		req.ReplyMarkup = buttons
	}

	result, err := api.SendMessage(ctx, botToken, req)
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(result.MessageID, 10), nil
}

type contentAttrs struct {
	InReplyToSourceID *string       `json:"in_reply_to_source_id"`
	Buttons           [][]ButtonDef `json:"buttons,omitempty"`
}

func parseContentAttrs(raw string) (replyTo int64, buttons *ReplyMarkup) {
	if raw == "" {
		return 0, nil
	}
	var attrs contentAttrs
	if err := json.Unmarshal([]byte(raw), &attrs); err != nil {
		return 0, nil
	}
	if attrs.InReplyToSourceID != nil && *attrs.InReplyToSourceID != "" {
		if id, err := strconv.ParseInt(*attrs.InReplyToSourceID, 10, 64); err == nil {
			replyTo = id
		}
	}
	if len(attrs.Buttons) > 0 {
		keyboard := make([][]InlineKeyboardButton, 0, len(attrs.Buttons))
		for _, row := range attrs.Buttons {
			inlineRow := make([]InlineKeyboardButton, 0, len(row))
			for _, btn := range row {
				ib := InlineKeyboardButton(btn)
				inlineRow = append(inlineRow, ib)
			}
			keyboard = append(keyboard, inlineRow)
		}
		buttons = &ReplyMarkup{InlineKeyboard: keyboard}
	}
	return replyTo, buttons
}
