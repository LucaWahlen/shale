package httpapi

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"shale/internal/domain"
)

const (
	defaultAuditPageSize = 25
	maxAuditPageSize     = 100
)

func (s *Server) handleListAudit(w http.ResponseWriter, r *http.Request) {
	q, err := parseAuditListQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	page, err := s.deps.Audit.List(r.Context(), q)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]auditEntryDTO, 0, len(page.Items))
	for _, e := range page.Items {
		items = append(items, auditEntryDTO{
			ID:            e.ID,
			CreatedAt:     e.CreatedAt.UTC().Format(rfc3339),
			Action:        e.Action,
			Actor:         e.Actor,
			ActorName:     e.ActorName,
			ScheduleID:    e.ScheduleID,
			ScheduleTitle: e.ScheduleTitle,
			EventID:       e.EventID,
			EventName:     e.EventName,
			Detail:        e.Detail,
		})
	}
	writeJSON(w, http.StatusOK, auditPageDTO{
		Items:    items,
		Total:    page.Total,
		Page:     q.Page,
		PageSize: q.PageSize,
	})
}

func parseAuditListQuery(r *http.Request) (domain.AuditListQuery, error) {
	query := r.URL.Query()
	q := domain.AuditListQuery{
		Search:     strings.TrimSpace(query.Get("q")),
		Action:     query.Get("action"),
		ScheduleID: query.Get("schedule_id"),
		Page:       1,
		PageSize:   defaultAuditPageSize,
	}
	if q.Action != "" && !domain.ValidAuditAction(q.Action) {
		return q, domain.NewError(domain.KindInvalid, "invalid action value")
	}
	if q.ScheduleID != "" && uuid.Validate(q.ScheduleID) != nil {
		return q, domain.NewError(domain.KindInvalid, "schedule_id must be a valid UUID")
	}
	if v := query.Get("page"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 {
			return q, domain.NewError(domain.KindInvalid, "page must be a positive integer")
		}
		q.Page = n
	}
	if v := query.Get("page_size"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > maxAuditPageSize {
			return q, domain.Errorf(domain.KindInvalid, "page_size must be between 1 and %d", maxAuditPageSize)
		}
		q.PageSize = n
	}
	return q, nil
}
