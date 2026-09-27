package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"shale/internal/domain"
	"shale/internal/service"
)

const rfc3339 = time.RFC3339

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Password string `json:"password"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	value, expiry, err := s.deps.Auth.Login(req.Password)
	if err != nil {
		writeError(w, err)
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Secure:   r.TLS != nil,
		Expires:  expiry,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleListSchedules(w http.ResponseWriter, r *http.Request) {
	q, err := parseScheduleListQuery(r)
	if err != nil {
		writeError(w, err)
		return
	}
	page, err := s.deps.Schedules.ListDetailed(r.Context(), q)
	if err != nil {
		writeError(w, err)
		return
	}
	items := make([]adminScheduleDTO, 0, len(page.Items))
	for _, detail := range page.Items {
		dto := toAdminScheduleDTO(detail)
		dto.EventCount = len(detail.Events)
		if len(detail.Events) > 0 {
			dto.FirstStartsAt = detail.Events[0].Event.StartsAt
			dto.LastStartsAt = detail.Events[len(detail.Events)-1].Event.StartsAt
		}
		items = append(items, dto)
	}
	writeJSON(w, http.StatusOK, adminSchedulePageDTO{
		Items:    items,
		Total:    page.Total,
		Page:     q.Page,
		PageSize: q.PageSize,
	})
}

const (
	defaultSchedulePageSize = 10
	maxSchedulePageSize     = 100
)

var validScheduleSorts = map[string]struct{}{
	"newest":     {},
	"oldest":     {},
	"title":      {},
	"title_desc": {},
	"updated":    {},
	"soonest":    {},
	"events":     {},
}

func parseScheduleListQuery(r *http.Request) (domain.ScheduleListQuery, error) {
	query := r.URL.Query()
	q := domain.ScheduleListQuery{
		Search:   strings.TrimSpace(query.Get("q")),
		Sort:     query.Get("sort"),
		Page:     1,
		PageSize: defaultSchedulePageSize,
	}
	if q.Sort == "" {
		q.Sort = "newest"
	}
	if _, ok := validScheduleSorts[q.Sort]; !ok {
		return q, domain.NewError(domain.KindInvalid, "invalid sort value")
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
		if err != nil || n < 1 || n > maxSchedulePageSize {
			return q, domain.Errorf(domain.KindInvalid, "page_size must be between 1 and %d", maxSchedulePageSize)
		}
		q.PageSize = n
	}
	switch query.Get("include_past") {
	case "1", "true":
		q.IncludePast = true
	}
	return q, nil
}

func (s *Server) handleCreateSchedule(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	sched, err := s.deps.Schedules.Create(r.Context(), service.ScheduleInput{
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminScheduleDTO{
		ID:          sched.ID,
		Title:       sched.Title,
		Description: sched.Description,
		Events:      []adminEventDTO{},
		CreatedAt:   sched.CreatedAt.UTC().Format(rfc3339),
		UpdatedAt:   sched.UpdatedAt.UTC().Format(rfc3339),
	})
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	detail, err := s.deps.Schedules.Get(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toAdminScheduleDTO(detail))
}

func (s *Server) handleUpdateSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Title       string `json:"title"`
		Description string `json:"description"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	sched, err := s.deps.Schedules.Update(r.Context(), id, service.ScheduleInput{
		Title:       req.Title,
		Description: req.Description,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, adminScheduleDTO{
		ID:          sched.ID,
		Title:       sched.Title,
		Description: sched.Description,
		CreatedAt:   sched.CreatedAt.UTC().Format(rfc3339),
		UpdatedAt:   sched.UpdatedAt.UTC().Format(rfc3339),
	})
}

func (s *Server) handleDeleteSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.deps.Schedules.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleDuplicateSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Title string `json:"title"`
	}

	if r.Body != nil && r.ContentLength != 0 {
		if err := readJSON(w, r, &req); err != nil {
			writeError(w, err)
			return
		}
	}
	sched, err := s.deps.Schedules.Duplicate(r.Context(), id, req.Title)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminScheduleDTO{
		ID:          sched.ID,
		Title:       sched.Title,
		Description: sched.Description,
		CreatedAt:   sched.CreatedAt.UTC().Format(rfc3339),
		UpdatedAt:   sched.UpdatedAt.UTC().Format(rfc3339),
	})
}

func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Name        string  `json:"name"`
		Description string  `json:"description"`
		Location    string  `json:"location"`
		StartsAt    string  `json:"starts_at"`
		AllDay      bool    `json:"all_day"`
		EndsAt      *string `json:"ends_at"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	endsAt := ""
	if req.EndsAt != nil {
		endsAt = *req.EndsAt
	}
	ev, err := s.deps.Events.Create(r.Context(), id, service.EventInput{
		Name:        req.Name,
		Description: req.Description,
		Location:    req.Location,
		StartsAt:    req.StartsAt,
		AllDay:      req.AllDay,
		EndsAt:      endsAt,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminEventDTO{
		ID:          ev.ID,
		ScheduleID:  ev.ScheduleID,
		Name:        ev.Name,
		Description: ev.Description,
		Location:    ev.Location,
		StartsAt:    ev.StartsAt,
		AllDay:      ev.AllDay,
		EndsAt:      ev.EndsAt,
		Attendees:   []adminAttendeeDTO{},
		CreatedAt:   ev.CreatedAt.UTC().Format(rfc3339),
		UpdatedAt:   ev.UpdatedAt.UTC().Format(rfc3339),
	})
}

func (s *Server) handlePatchEvent(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "eventID")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Description *string `json:"description"`
		Location    *string `json:"location"`
		StartsAt    *string `json:"starts_at"`
		AllDay      *bool   `json:"all_day"`
		EndsAt      *string `json:"ends_at"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	ev, err := s.deps.Events.Patch(r.Context(), id, service.EventPatch{
		Name:        req.Name,
		Description: req.Description,
		Location:    req.Location,
		StartsAt:    req.StartsAt,
		AllDay:      req.AllDay,
		EndsAt:      req.EndsAt,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, adminEventDTO{
		ID:          ev.ID,
		ScheduleID:  ev.ScheduleID,
		Name:        ev.Name,
		Description: ev.Description,
		Location:    ev.Location,
		StartsAt:    ev.StartsAt,
		AllDay:      ev.AllDay,
		EndsAt:      ev.EndsAt,
		CreatedAt:   ev.CreatedAt.UTC().Format(rfc3339),
		UpdatedAt:   ev.UpdatedAt.UTC().Format(rfc3339),
	})
}

func (s *Server) handleDeleteEvent(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "eventID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.deps.Events.Delete(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleAddAttendee(w http.ResponseWriter, r *http.Request) {
	eventID, err := pathUUID(r, "eventID")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		Name string `json:"name"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	att, err := s.deps.Attendees.AdminAdd(r.Context(), eventID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, adminAttendeeDTO{ID: att.ID, Name: att.Name})
}

func (s *Server) handleRemoveAttendee(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "attendeeID")
	if err != nil {
		writeError(w, err)
		return
	}
	if err := s.deps.Attendees.AdminRemove(r.Context(), id); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) handleGetAdminSettings(w http.ResponseWriter, r *http.Request) {
	view, err := s.deps.Settings.Get(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"default_language":     view.DefaultLanguage,
		"app_name":             view.AppName,
		"imprint_text":         view.ImprintText,
		"privacy_text":         view.PrivacyText,
		"audit_retention_days": view.AuditRetentionDays,
	})
}

func (s *Server) handlePutAdminSettings(w http.ResponseWriter, r *http.Request) {
	var req struct {
		DefaultLanguage    string  `json:"default_language"`
		AppName            *string `json:"app_name"`
		ImprintText        *string `json:"imprint_text"`
		PrivacyText        *string `json:"privacy_text"`
		AuditRetentionDays *int    `json:"audit_retention_days"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.deps.Settings.SetDefaultLanguage(r.Context(), req.DefaultLanguage); err != nil {
		writeError(w, err)
		return
	}
	if req.AppName != nil {
		if err := s.deps.Settings.SetAppName(r.Context(), *req.AppName); err != nil {
			writeError(w, err)
			return
		}
	}
	if req.ImprintText != nil {
		if err := s.deps.Settings.SetImprintText(r.Context(), *req.ImprintText); err != nil {
			writeError(w, err)
			return
		}
	}
	if req.PrivacyText != nil {
		if err := s.deps.Settings.SetPrivacyText(r.Context(), *req.PrivacyText); err != nil {
			writeError(w, err)
			return
		}
	}
	if req.AuditRetentionDays != nil {
		if err := s.deps.Settings.SetAuditRetentionDays(r.Context(), *req.AuditRetentionDays); err != nil {
			writeError(w, err)
			return
		}
	}
	view, err := s.deps.Settings.Get(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"default_language":     view.DefaultLanguage,
		"app_name":             view.AppName,
		"imprint_text":         view.ImprintText,
		"privacy_text":         view.PrivacyText,
		"audit_retention_days": view.AuditRetentionDays,
	})
}

func (s *Server) handleExport(w http.ResponseWriter, r *http.Request) {
	doc, err := s.deps.Transfer.Export(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="shale-export-%s.json"`, time.Now().UTC().Format("20060102")))
	w.WriteHeader(http.StatusOK)
	_ = writePrettyJSON(w, doc)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	raw, err := readRawBody(w, r, 20<<20)
	if err != nil {
		writeError(w, err)
		return
	}
	counts, err := s.deps.Transfer.Import(r.Context(), raw)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, importResponseDTO{Imported: importCountsDTO{
		Schedules: counts.Schedules,
		Events:    counts.Events,
		Attendees: counts.Attendees,
	}})
}

func writePrettyJSON(w http.ResponseWriter, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func domainError(msg string) error {
	return domain.NewError(domain.KindInvalid, msg)
}
