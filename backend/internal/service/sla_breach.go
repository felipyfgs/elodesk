package service

import (
	"context"
	"time"

	"backend/internal/audit"
	"backend/internal/logger"
	"backend/internal/repo"
)

// NotificationCreator is the minimal surface SLABreachJob needs to persist a
// user notification. The full NotificationService (phase 8) implements this;
// passing nil disables notification delivery.
type NotificationCreator interface {
	Create(ctx context.Context, accountID, userID int64, ntype string, payload any) error
}

// SLABreachJob scans for conversations past their SLA due-at and flags them as
// breached. For each newly detected breach it emits `sla.breached` on the
// account realtime room, persists a notification for the assignee (when
// present) and records an audit log entry.
//
// The job is run as a tight goroutine ticker (default 60s). This keeps the
// worker path simple: no separate asynq server is needed — the main process
// owns detection. If the loop panics or the ticker stalls the next ticker
// iteration resumes work.
type SLABreachJob struct {
	slaRepo         *repo.SLARepo
	notificationSvc NotificationCreator
	realtime        *RealtimeService
	auditLogger     *audit.Logger
	interval        time.Duration

	stop chan struct{}
}

func NewSLABreachJob(slaRepo *repo.SLARepo, notificationSvc NotificationCreator, realtimeSvc *RealtimeService, auditLogger *audit.Logger, interval time.Duration) *SLABreachJob {
	if interval <= 0 {
		interval = time.Minute
	}
	return &SLABreachJob{
		slaRepo:         slaRepo,
		notificationSvc: notificationSvc,
		realtime:        realtimeSvc,
		auditLogger:     auditLogger,
		interval:        interval,
		stop:            make(chan struct{}),
	}
}

func (j *SLABreachJob) Start(ctx context.Context) {
	go j.run(ctx)
}

func (j *SLABreachJob) Stop() {
	select {
	case <-j.stop:
	default:
		close(j.stop)
	}
}

func (j *SLABreachJob) run(ctx context.Context) {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	logger.Info().Str("component", "sla-breach").Str("interval", j.interval.String()).Msg("sla breach job started")
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

func (j *SLABreachJob) tick(ctx context.Context) {
	candidates, err := j.slaRepo.ListBreachCandidates(ctx, 500)
	if err != nil {
		logger.Error().Str("component", "sla-breach").Err(err).Msg("list breach candidates failed")
		return
	}
	for _, c := range candidates {
		if err := j.slaRepo.MarkBreached(ctx, c.ID); err != nil {
			logger.Error().Str("component", "sla-breach").Err(err).Int64("conversation_id", c.ID).Msg("mark breached failed")
			continue
		}

		payload := map[string]any{
			"conversation_id": c.ID,
			"policy_id":       c.PolicyID,
			"kind":            c.Kind,
			"due_at":          c.DueAt.UTC().Format(time.RFC3339),
		}

		if j.realtime != nil {
			j.realtime.BroadcastAccountEvent(c.AccountID, "sla.breached", payload)
		}
		if j.notificationSvc != nil && c.AssigneeID != nil {
			_ = j.notificationSvc.Create(ctx, c.AccountID, *c.AssigneeID, "sla_breach", payload)
		}
		if j.auditLogger != nil {
			convID := c.ID
			j.auditLogger.Log(ctx, c.AccountID, nil, "sla.breached", "conversation", &convID, payload, "", "")
		}
	}
}
