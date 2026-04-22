package twilio

import (
	"context"
	"time"

	"backend/internal/logger"
	"backend/internal/repo"
)

// TemplatesJob periodically refreshes Twilio content templates for every
// WhatsApp-medium channel whose cache is older than maxAge. Keeps outbound
// template sends from failing due to stale template SIDs.
type TemplatesJob struct {
	repo     *repo.ChannelTwilioRepo
	channel  *Channel
	interval time.Duration
	maxAge   time.Duration

	stop chan struct{}
}

func NewTemplatesJob(r *repo.ChannelTwilioRepo, ch *Channel, interval, maxAge time.Duration) *TemplatesJob {
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	if maxAge <= 0 {
		maxAge = 24 * time.Hour
	}
	return &TemplatesJob{
		repo:     r,
		channel:  ch,
		interval: interval,
		maxAge:   maxAge,
		stop:     make(chan struct{}),
	}
}

func (j *TemplatesJob) Start(ctx context.Context) {
	go j.run(ctx)
}

func (j *TemplatesJob) Stop() {
	select {
	case <-j.stop:
	default:
		close(j.stop)
	}
}

func (j *TemplatesJob) run(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	logger.Info().Str("component", "channel.twilio.templates").Str("interval", j.interval.String()).Msg("twilio templates job started")
	for {
		select {
		case <-ctx.Done():
			return
		case <-j.stop:
			return
		case <-ticker.C:
			j.tick(ctx)
		}
	}
}

func (j *TemplatesJob) tick(ctx context.Context) {
	cutoff := time.Now().Add(-j.maxAge)
	stale, err := j.repo.ListStaleTemplates(ctx, cutoff)
	if err != nil {
		logger.Error().Str("component", "channel.twilio.templates").Err(err).Msg("list stale templates")
		return
	}
	for _, ch := range stale {
		if _, err := j.channel.SyncTemplatesForChannel(ctx, ch); err != nil {
			logger.Warn().Str("component", "channel.twilio.templates").Int64("channel_id", ch.ID).Err(err).Msg("sync templates failed")
			continue
		}
		logger.Info().Str("component", "channel.twilio.templates").Int64("channel_id", ch.ID).Msg("templates synced")
	}
}
