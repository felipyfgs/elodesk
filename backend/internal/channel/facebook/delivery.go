package facebook

import (
	"context"
	"fmt"
	"time"

	"backend/internal/logger"
	"backend/internal/model"
	"backend/internal/repo"
)

// ProcessDelivery updates message statuses for all messages in the conversation
// with a timestamp at or before the watermark.
func ProcessDelivery(
	ctx context.Context,
	conversationID, accountID int64,
	watermark int64,
	messageRepo *repo.MessageRepo,
) error {
	watermarkTime := time.Unix(watermark/1000, 0)
	updated, err := messageRepo.MarkDeliveredBefore(ctx, conversationID, accountID, watermarkTime, model.MessageDelivered)
	if err != nil {
		return fmt.Errorf("facebook delivery: update status: %w", err)
	}
	logger.Debug().
		Str("component", "facebook.delivery").
		Int64("conversationId", conversationID).
		Int("updated", updated).
		Msg("delivery watermark processed")
	return nil
}
