package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"shale/internal/db"
	"shale/internal/domain"
	"shale/internal/repository/memory"
	"shale/internal/repository/postgres"
	"shale/internal/service"
)

type testEnv struct {
	t    *testing.T
	api  http.Handler
	auth *service.AuthService
}

func newTestEnv(t *testing.T, now time.Time) *testEnv {
	t.Helper()
	var (
		scheduleRepo domain.ScheduleRepository
		eventRepo    domain.EventRepository
		attendeeRepo domain.AttendeeRepository
		settingsRepo domain.SettingsRepository
		tx           domain.TransactionManager
	)

	if dsn := os.Getenv("SHALE_TEST_DATABASE_URL"); dsn != "" {
		conn, err := db.Open(dsn)
		if err != nil {
			t.Fatalf("open db: %v", err)
		}
		t.Cleanup(func() { conn.Close() })
		if err := db.Migrate(context.Background(), conn); err != nil {
			t.Fatalf("migrate: %v", err)
		}
		if _, err := conn.Exec(`DELETE FROM schedules; DELETE FROM settings; INSERT INTO settings (key, value) VALUES ('default_language', 'en')`); err != nil {
			t.Fatalf("reset db: %v", err)
		}
		store := postgres.New(conn)
		scheduleRepo = postgres.NewScheduleRepo(store)
		eventRepo = postgres.NewEventRepo(store)
		attendeeRepo = postgres.NewAttendeeRepo(store)
		settingsRepo = postgres.NewSettingsRepo(store)
		tx = store
	} else {
		store := memory.New()
		scheduleRepo = memory.NewScheduleRepo(store)
		eventRepo = memory.NewEventRepo(store)
		attendeeRepo = memory.NewAttendeeRepo(store)
		settingsRepo = memory.NewSettingsRepo(store)
		tx = store
	}

	clock := func() time.Time { return now }

	scheduleSvc := service.NewScheduleService(scheduleRepo, eventRepo, attendeeRepo, settingsRepo, clock)
	eventSvc := service.NewEventService(scheduleRepo, eventRepo)
	attendeeSvc := service.NewAttendeeService(scheduleRepo, eventRepo, attendeeRepo, clock)
	settingsSvc := service.NewSettingsService(settingsRepo)
	transferSvc := service.NewTransferService(scheduleRepo, eventRepo, attendeeRepo, settingsSvc, tx, clock)
	authSvc := service.NewAuthService("test-password", clock)

	api := New(Deps{
		Schedules: scheduleSvc,
		Events:    eventSvc,
		Attendees: attendeeSvc,
		Settings:  settingsSvc,
		Transfer:  transferSvc,
		Auth:      authSvc,
	}, slog.New(slog.DiscardHandler))

	return &testEnv{t: t, api: api.Handler(), auth: authSvc}
}

func (e *testEnv) do(method, path string, body any, cookie *http.Cookie) (*http.Response, map[string]any) {
	e.t.Helper()
	var out map[string]any
	res := e.doInto(method, path, body, cookie, &out)
	return res, out
}

func (e *testEnv) doInto(method, path string, body any, cookie *http.Cookie, out any) *http.Response {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		var buf bytes.Buffer
		if err := json.NewEncoder(&buf).Encode(body); err != nil {
			e.t.Fatalf("encode body: %v", err)
		}
		reader = &buf
	}
	req, err := http.NewRequest(method, path, reader)
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.api.ServeHTTP(rec, req)
	res := rec.Result()
	if res.Body != nil {
		_ = json.NewDecoder(res.Body).Decode(out)
	}
	return res
}

func (e *testEnv) login() *http.Cookie {
	e.t.Helper()
	res, _ := e.do(http.MethodPost, "/api/v1/admin/login", map[string]string{"password": "test-password"}, nil)
	if res.StatusCode != http.StatusOK {
		e.t.Fatalf("login status = %d, want 200", res.StatusCode)
	}
	for _, c := range res.Cookies() {
		if c.Name == SessionCookieName {
			return c
		}
	}
	e.t.Fatal("no session cookie returned")
	return nil
}

func (e *testEnv) seedSchedule(startsAt string) (string, string) {
	e.t.Helper()
	cookie := e.login()
	res, out := e.do(http.MethodPost, "/api/v1/admin/schedules", map[string]string{
		"title": "Test Schedule",
	}, cookie)
	if res.StatusCode != http.StatusCreated {
		e.t.Fatalf("create schedule status = %d", res.StatusCode)
	}
	scheduleID := out["id"].(string)
	res, out = e.do(http.MethodPost, "/api/v1/admin/schedules/"+scheduleID+"/events", map[string]string{
		"name": "Test Event", "starts_at": startsAt,
	}, cookie)
	if res.StatusCode != http.StatusCreated {
		e.t.Fatalf("create event status = %d", res.StatusCode)
	}
	eventID := out["id"].(string)
	return scheduleID, eventID
}
