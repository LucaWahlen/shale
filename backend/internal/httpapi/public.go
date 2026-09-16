package httpapi

import (
	"net/http"

	"shale/internal/service"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handlePublicSettings(w http.ResponseWriter, r *http.Request) {
	view, err := s.deps.Settings.Get(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{
		"default_language": view.DefaultLanguage,
		"app_name":         view.AppName,
	})
}

func (s *Server) handlePublicSchedule(w http.ResponseWriter, r *http.Request) {
	id, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	ps, err := s.deps.Schedules.GetPublic(r.Context(), id)
	if err != nil {
		writeError(w, err)
		return
	}
	resp := publicScheduleDTO{
		ID:          ps.Schedule.ID,
		Title:       ps.Schedule.Title,
		Description: ps.Schedule.Description,
		Events:      make([]publicEventDTO, 0, len(ps.Events)),
	}
	for _, pe := range ps.Events {
		dto := publicEventDTO{
			ID:          pe.Event.ID,
			Name:        pe.Event.Name,
			Description: pe.Event.Description,
			Location:    pe.Event.Location,
			StartsAt:    pe.Event.StartsAt,
			AllDay:      pe.Event.AllDay,
			EndsAt:      pe.Event.EndsAt,
			Attendable:  pe.Attendable,
			Attendees:   make([]publicAttendeeDTO, 0, len(pe.Attendees)),
		}
		for _, a := range pe.Attendees {
			dto.Attendees = append(dto.Attendees, publicAttendeeDTO{ID: a.ID, Name: a.Name})
		}
		resp.Events = append(resp.Events, dto)
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleAttend(w http.ResponseWriter, r *http.Request) {
	scheduleID, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
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
	res, err := s.deps.Attendees.Attend(r.Context(), scheduleID, eventID, req.Name)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, attendResponseDTO{
		Attendee:    publicAttendeeDTO{ID: res.Attendee.ID, Name: res.Attendee.Name},
		ManageToken: res.Token,
		Created:     res.Created,
	})
}

func (s *Server) handleRevert(w http.ResponseWriter, r *http.Request) {
	scheduleID, err := pathUUID(r, "id")
	if err != nil {
		writeError(w, err)
		return
	}
	eventID, err := pathUUID(r, "eventID")
	if err != nil {
		writeError(w, err)
		return
	}
	var req struct {
		AttendeeID  string `json:"attendee_id"`
		ManageToken string `json:"manage_token"`
	}
	if err := readJSON(w, r, &req); err != nil {
		writeError(w, err)
		return
	}
	if err := s.deps.Attendees.Revert(r.Context(), scheduleID, eventID, req.AttendeeID, req.ManageToken); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func toAdminScheduleDTO(sc service.ScheduleDetail) adminScheduleDTO {
	dto := adminScheduleDTO{
		ID:          sc.Schedule.ID,
		Title:       sc.Schedule.Title,
		Description: sc.Schedule.Description,
		CreatedAt:   sc.Schedule.CreatedAt.UTC().Format(rfc3339),
		UpdatedAt:   sc.Schedule.UpdatedAt.UTC().Format(rfc3339),
		Events:      make([]adminEventDTO, 0, len(sc.Events)),
	}
	for _, ed := range sc.Events {
		dto.Events = append(dto.Events, toAdminEventDTO(ed))
	}
	return dto
}

func toAdminEventDTO(ed service.EventDetail) adminEventDTO {
	dto := adminEventDTO{
		ID:          ed.Event.ID,
		ScheduleID:  ed.Event.ScheduleID,
		Name:        ed.Event.Name,
		Description: ed.Event.Description,
		Location:    ed.Event.Location,
		StartsAt:    ed.Event.StartsAt,
		AllDay:      ed.Event.AllDay,
		EndsAt:      ed.Event.EndsAt,
		Attendees:   make([]adminAttendeeDTO, 0, len(ed.Attendees)),
		CreatedAt:   ed.Event.CreatedAt.UTC().Format(rfc3339),
		UpdatedAt:   ed.Event.UpdatedAt.UTC().Format(rfc3339),
	}
	for _, a := range ed.Attendees {
		dto.Attendees = append(dto.Attendees, adminAttendeeDTO{ID: a.ID, Name: a.Name})
	}
	return dto
}
