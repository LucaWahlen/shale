package httpapi

import (
	"log/slog"
	"net/http"

	"shale/internal/service"
)

const SessionCookieName = "shale_session"

type Deps struct {
	Schedules *service.ScheduleService
	Events    *service.EventService
	Attendees *service.AttendeeService
	Settings  *service.SettingsService
	Transfer  *service.TransferService
	Auth      *service.AuthService
	Audit     *service.AuditService
}

type Server struct {
	deps    Deps
	log     *slog.Logger
	limiter *loginLimiter
}

func New(deps Deps, log *slog.Logger) *Server {
	return &Server{deps: deps, log: log, limiter: newLoginLimiter(5, defaultWindow)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /api/v1/health", s.handleHealth)
	mux.HandleFunc("GET /api/v1/settings", s.handlePublicSettings)
	mux.HandleFunc("GET /api/v1/schedules/{id}", s.handlePublicSchedule)
	mux.HandleFunc("POST /api/v1/schedules/{id}/events/{eventID}/attend", s.handleAttend)
	mux.HandleFunc("DELETE /api/v1/schedules/{id}/events/{eventID}/attend", s.handleRevert)

	mux.Handle("POST /api/v1/admin/login", s.limiter.middleware(http.HandlerFunc(s.handleLogin)))
	mux.HandleFunc("POST /api/v1/admin/logout", s.handleLogout)

	mux.Handle("GET /api/v1/admin/schedules", s.admin(s.handleListSchedules))
	mux.Handle("POST /api/v1/admin/schedules", s.admin(s.handleCreateSchedule))
	mux.Handle("GET /api/v1/admin/schedules/{id}", s.admin(s.handleGetSchedule))
	mux.Handle("PUT /api/v1/admin/schedules/{id}", s.admin(s.handleUpdateSchedule))
	mux.Handle("DELETE /api/v1/admin/schedules/{id}", s.admin(s.handleDeleteSchedule))
	mux.Handle("POST /api/v1/admin/schedules/{id}/duplicate", s.admin(s.handleDuplicateSchedule))
	mux.Handle("POST /api/v1/admin/schedules/{id}/events", s.admin(s.handleCreateEvent))

	mux.Handle("PATCH /api/v1/admin/events/{eventID}", s.admin(s.handlePatchEvent))
	mux.Handle("DELETE /api/v1/admin/events/{eventID}", s.admin(s.handleDeleteEvent))
	mux.Handle("POST /api/v1/admin/events/{eventID}/attendees", s.admin(s.handleAddAttendee))
	mux.Handle("DELETE /api/v1/admin/attendees/{attendeeID}", s.admin(s.handleRemoveAttendee))

	mux.Handle("GET /api/v1/admin/settings", s.admin(s.handleGetAdminSettings))
	mux.Handle("PUT /api/v1/admin/settings", s.admin(s.handlePutAdminSettings))
	mux.Handle("GET /api/v1/admin/export", s.admin(s.handleExport))
	mux.Handle("POST /api/v1/admin/import", s.admin(s.handleImport))
	mux.Handle("GET /api/v1/admin/audit", s.admin(s.handleListAudit))

	mux.HandleFunc("/api/{path...}", func(w http.ResponseWriter, r *http.Request) {
		writeError(w, errNotFoundRoute)
	})

	return s.logMiddleware(s.recoverMiddleware(mux))
}
