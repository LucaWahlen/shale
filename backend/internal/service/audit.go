package service

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"shale/internal/domain"
)

type AuditService struct {
	repo     domain.AuditRepository
	settings domain.SettingsRepository
}

func NewAuditService(repo domain.AuditRepository, settings domain.SettingsRepository) *AuditService {
	return &AuditService{repo: repo, settings: settings}
}

type AuditInput struct {
	Action        string
	Actor         string
	ActorName     string
	ScheduleID    string
	ScheduleTitle string
	EventID       string
	EventName     string
	Detail        string
}

// Record writes an audit entry. It is best-effort: failures are logged and do
// not fail the underlying operation.
func (s *AuditService) Record(ctx context.Context, in AuditInput) {
	if s == nil || s.repo == nil || in.Action == "" {
		return
	}
	entry := domain.AuditEntry{
		Action:        in.Action,
		Actor:         in.Actor,
		ActorName:     in.ActorName,
		ScheduleID:    in.ScheduleID,
		ScheduleTitle: in.ScheduleTitle,
		EventID:       in.EventID,
		EventName:     in.EventName,
		Detail:        in.Detail,
	}
	if err := s.repo.Append(ctx, &entry); err != nil {
		slog.Error("failed to write audit entry", "action", in.Action, "error", err)
	}
	s.Prune(ctx)
}

// Prune deletes audit entries older than the configured retention period.
// A retention of 0 (the default) keeps entries indefinitely.
func (s *AuditService) Prune(ctx context.Context) {
	if s == nil || s.repo == nil || s.settings == nil {
		return
	}
	days := s.retentionDays(ctx)
	if days <= 0 {
		return
	}
	cutoff := time.Now().UTC().AddDate(0, 0, -days).Format(time.RFC3339)
	if _, err := s.repo.DeleteBefore(ctx, cutoff); err != nil {
		slog.Error("failed to prune audit log", "error", err)
	}
}

func (s *AuditService) retentionDays(ctx context.Context) int {
	v, err := s.settings.Get(ctx, domain.SettingAuditRetentionDays)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func (s *AuditService) List(ctx context.Context, q domain.AuditListQuery) (domain.AuditPage, error) {
	return s.repo.List(ctx, q)
}
