package service

import (
	"context"
	"time"

	"backend/internal/logger"
	"backend/internal/repo"
)

// AuditRetentionJob periodically purges audit_logs older than the configured
// retention window (default 90 days) and logs how many rows were removed.
type AuditRetentionJob struct {
	auditRepo *repo.AuditLogRepo
	days      int
	interval  time.Duration

	stop chan struct{}
}

func NewAuditRetentionJob(auditRepo *repo.AuditLogRepo, days int, interval time.Duration) *AuditRetentionJob {
	if days <= 0 {
		days = 90
	}
	if interval <= 0 {
		interval = 24 * time.Hour
	}
	return &AuditRetentionJob{
		auditRepo: auditRepo,
		days:      days,
		interval:  interval,
		stop:      make(chan struct{}),
	}
}

func (j *AuditRetentionJob) Start(ctx context.Context) {
	go j.run(ctx)
}

func (j *AuditRetentionJob) Stop() {
	select {
	case <-j.stop:
	default:
		close(j.stop)
	}
}

func (j *AuditRetentionJob) run(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	logger.Info().Str("component", "audit-retention").Int("days", j.days).Str("interval", j.interval.String()).Msg("audit retention job started")
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

func (j *AuditRetentionJob) tick(ctx context.Context) {
	n, err := j.auditRepo.DeleteOlderThan(ctx, j.days)
	if err != nil {
		logger.Error().Str("component", "audit-retention").Err(err).Msg("failed to purge old audit logs")
		return
	}
	if n > 0 {
		logger.Info().Str("component", "audit-retention").Int64("deleted", n).Int("days", j.days).Msg("purged old audit logs")
	}
}
